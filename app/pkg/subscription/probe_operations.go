package subscription

import (
	"context"
	"errors"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/integrations/clashapi"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription/internal/probe"
	"strconv"
	"strings"
	"time"
)

type ProbeSettingsCommand struct {
	// Pointers so "not sent" is distinguishable from "set to zero" — a
	// partial update from any other API client must not silently reset the
	// three knobs it did not mention.
	IntervalSeconds *int `json:"interval_seconds"`
	TimeoutMs       *int `json:"timeout_ms"`
	RetentionDays   *int `json:"retention_days"`
	MaxPoints       *int `json:"max_points"`
}

// Ranges the history endpoint accepts, and the bucket each one is served at.
//
// The bucket is chosen from the RANGE, not from how densely the operator
// sampled: a month at the 1-minute floor is ~43k rows, and a chart ~600px wide
// cannot draw them. Bucketing server-side keeps the wire cost a function of
// what was asked for rather than of a setting the reader cannot see.
//
// Each row is sized so a full range lands near 200-300 points, which is roughly
// one per pixel-pair at the widths this chart is drawn at.
var probeRanges = map[string]struct {
	window time.Duration
	bucket time.Duration
}{
	"1h":  {time.Hour, 0}, // unbucketed: at most 60 rows at the interval floor
	"6h":  {6 * time.Hour, 2 * time.Minute},
	"24h": {24 * time.Hour, 10 * time.Minute},
	"7d":  {7 * 24 * time.Hour, time.Hour},
	"30d": {30 * 24 * time.Hour, 4 * time.Hour},
}

const defaultProbeRange = "24h"

// GetProbeStatus reports the scheduler's state plus the latest sample for every
// subscription — one call, because the list page and the Overview card both
// want the whole set and a call per row would be N round trips to draw a
// summary.
func (h *Service) GetProbeStatus(ctx context.Context) (map[string]any, error) {
	if h.probeRunner == nil || h.probeStore == nil {
		return nil, fault.New(fault.Unavailable, "the subscription prober is not running")
	}

	latest, err := h.probeStore.Latest()
	if err != nil {
		return nil, fault.New(fault.Failed, err.Error())
	}

	// Reported so the Settings page can state the disk cost of the current
	// retention settings as a measured number rather than an estimate.
	count, err := h.probeStore.CountAll()
	if err != nil {
		return nil, fault.New(fault.Failed, err.Error())
	}

	return map[string]any{
		"status":        h.probeRunner.Status(),
		"latest":        latest,
		"sample_count":  count,
		"retention":     h.settingsManager.GetProbeRetentionDays(),
		"max_points":    h.settingsManager.GetProbeMaxPoints(),
		"timeout_ms":    h.settingsManager.GetProbeTimeout().Milliseconds(),
		"interval_secs": int(h.settingsManager.GetProbeInterval().Seconds()),
	}, nil
}

// GetProbeHistory returns one subscription's series for a named range.
func (h *Service) GetProbeHistory(ctx context.Context, id, rangeKey string) (map[string]any, error) {
	if h.probeRunner == nil || h.probeStore == nil {
		return nil, fault.New(fault.Unavailable, "the subscription prober is not running")
	}

	rangeKey = strings.TrimSpace(rangeKey)
	if rangeKey == "" {
		rangeKey = defaultProbeRange
	}
	spec, ok := probeRanges[rangeKey]
	if !ok {
		// Named ranges rather than free-form from/to: the bucket has to be
		// chosen alongside the window, and a client picking one of them
		// independently is how a 30-day request arrives unbucketed.
		return nil, fault.New(fault.Invalid, "range must be one of 1h, 6h, 24h, 7d, 30d")
	}

	now := time.Now()
	points, err := h.probeStore.Series(id, now.Add(-spec.window), now.Add(time.Minute), spec.bucket)
	if err != nil {
		return nil, fault.New(fault.Failed, err.Error())
	}

	bucketSeconds := 0
	if spec.bucket > 0 {
		bucketSeconds = int(spec.bucket.Seconds())
	}

	return map[string]any{
		"id":             id,
		"range":          rangeKey,
		"bucket_seconds": bucketSeconds,
		// Always a list, never null: a chart that receives null has to special
		// -case it, and "no samples yet" is a legitimate state on a fresh
		// install rather than an error.
		"points": points,
	}, nil
}

// GetProbeNodes returns the per-node detail of the most recent run.
//
// In-memory only, so it is empty until the first sweep after a restart. That
// is stated in the payload (`available`) rather than served as an empty list,
// which would read as "every node is fine".
func (h *Service) GetProbeNodes(ctx context.Context, id string) (map[string]any, error) {
	if h.probeRunner == nil || h.probeStore == nil {
		return nil, fault.New(fault.Unavailable, "the subscription prober is not running")
	}

	snapshot := h.probeRunner.LatestNodes(id)
	if snapshot == nil {
		return map[string]any{"available": false}, nil
	}
	return map[string]any{
		"available": true,
		"at":        snapshot.At,
		"sample":    snapshot.Sample,
		"results":   snapshot.Results,
	}, nil
}

// RunProbe probes one subscription now — the per-row "test" button.
//
// It shares the runner's single-run gate with the scheduled sweep, so a long
// manual probe makes that interval's sweep return "already in progress" and
// every OTHER subscription miss one sample. That is deliberate — two concurrent
// runs would double the real load on a device that is also routing traffic —
// but it is the reason this button is not a good thing to hammer.
//
// Synchronous: the caller asked for a measurement and the useful reply is the
// measurement.
//
// The ceiling is sized against the settings an operator can actually choose,
// not the defaults. At the maximum per-node timeout (8s, plus the 2s transport
// slack probeOne adds) a 250-node subscription at 8-way concurrency is
// ~32 rounds x 10s, so roughly 5.5 minutes — which the previous 5-minute
// ceiling would have cut short. A truncated run is still reported rather than
// hidden: the nodes it never reached come back as `skipped` in the sample.
//
// A run this long is a pathological subscription, not the normal case (a
// healthy 37-node feed measured 5.2s), so this bounds the damage rather than
// describing the expectation.
func (h *Service) RunProbe(ctx context.Context, id string) (map[string]any, error) {
	if h.probeRunner == nil || h.probeStore == nil {
		return nil, fault.New(fault.Unavailable, "the subscription prober is not running")
	}

	runCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	sample, err := h.probeRunner.RunSubscription(runCtx, id)
	if err != nil {
		return nil, probeFailure(err)
	}

	return map[string]any{"sample": sample}, nil
}

// probeErrorMessage turns the two configuration failures into the sentence that
// names the fix. "clash api is not enabled" tells an operator what is wrong;
// it does not tell them where to go.
func probeErrorMessage(err error) string {
	switch {
	case errors.Is(err, clashapi.ErrDisabled):
		return "node probing needs experimental.clash_api.external_controller to be set, and sing-box to be running"
	case errors.Is(err, clashapi.ErrUnauthorized):
		return "sing-box rejected the request: check experimental.clash_api.secret"
	case errors.Is(err, subprobe.ErrNoOwnedNodes):
		return "this subscription has no nodes in the config yet — refresh it first"
	case errors.Is(err, subprobe.ErrNoNodesTestable):
		// Names the fix, because the two empty-result cases are otherwise
		// indistinguishable and lead to opposite actions.
		return "this subscription's nodes are in the config but not in the running sing-box — restart it to apply the config"
	default:
		return err.Error()
	}
}

// UpdateProbeSettings persists the probe knobs.
//
// A separate endpoint from PUT /settings because these four are validated as a
// group and reported back with the resulting storage estimate, which the
// generic settings endpoint has no shape for.
func (h *Service) UpdateProbeSettings(ctx context.Context, req ProbeSettingsCommand) (map[string]any, error) {

	if h.probeRunner == nil || h.probeStore == nil {
		return nil, fault.New(fault.Unavailable, "the subscription prober is not running")
	}

	// Validate the WHOLE group before writing any of it.
	//
	// Each Set* commits its own transaction, so writing as we validate meant a
	// request whose third field was out of range had already persisted the
	// first two — the client got an error and reasonably assumed nothing had
	// changed, while the interval had in fact moved. These four are edited
	// together in one form; they have to fail together too.
	if req.IntervalSeconds != nil {
		if err := settings.ValidateProbeInterval(time.Duration(*req.IntervalSeconds) * time.Second); err != nil {
			return nil, fault.New(fault.Invalid, err.Error())
		}
	}
	if req.TimeoutMs != nil {
		if err := settings.ValidateProbeTimeoutMs(*req.TimeoutMs); err != nil {
			return nil, fault.New(fault.Invalid, err.Error())
		}
	}
	if req.RetentionDays != nil {
		if err := settings.ValidateProbeRetentionDays(*req.RetentionDays); err != nil {
			return nil, fault.New(fault.Invalid, err.Error())
		}
	}
	if req.MaxPoints != nil {
		if err := settings.ValidateProbeMaxPoints(*req.MaxPoints); err != nil {
			return nil, fault.New(fault.Invalid, err.Error())
		}
	}

	values := map[string]string{}
	if req.IntervalSeconds != nil {
		values[settings.KeyProbeInterval] = strconv.Itoa(*req.IntervalSeconds)
	}
	if req.TimeoutMs != nil {
		values[settings.KeyProbeTimeout] = strconv.Itoa(*req.TimeoutMs)
	}
	if req.RetentionDays != nil {
		values[settings.KeyProbeRetentionDays] = strconv.Itoa(*req.RetentionDays)
	}
	if req.MaxPoints != nil {
		values[settings.KeyProbeMaxPoints] = strconv.Itoa(*req.MaxPoints)
	}
	if err := h.settingsManager.SetMany(values); err != nil {
		return nil, err
	}

	// The runner re-reads its interval after every sweep, so nothing has to be
	// restarted here. Tightened retention, though, is only applied by the next
	// write — an operator who just cut retention from 30 days to 1 expects the
	// disk back now, not in ten minutes.
	h.trimAllProbeHistory()

	count, err := h.probeStore.CountAll()
	if err != nil {
		return nil, fault.New(fault.Failed, err.Error())
	}

	return map[string]any{
		"interval_secs": int(h.settingsManager.GetProbeInterval().Seconds()),
		"timeout_ms":    h.settingsManager.GetProbeTimeout().Milliseconds(),
		"retention":     h.settingsManager.GetProbeRetentionDays(),
		"max_points":    h.settingsManager.GetProbeMaxPoints(),
		"sample_count":  count,
	}, nil
}

// trimAllProbeHistory applies the current retention bounds to every
// subscription. Best-effort: a failure here leaves rows in place, which is
// recoverable, and must not fail the settings write that caused it.
func (h *Service) trimAllProbeHistory() {
	if h.probeStore == nil {
		return
	}
	subs, err := h.List()
	if err != nil {
		return
	}
	maxAge := 24 * time.Hour * time.Duration(h.settingsManager.GetProbeRetentionDays())
	maxPoints := h.settingsManager.GetProbeMaxPoints()
	for _, sub := range subs {
		_, _ = h.probeStore.Trim(sub.ID, maxAge, maxPoints)
	}
}

func probeFailure(err error) error {
	kind := fault.Failed
	switch {
	case errors.Is(err, subprobe.ErrNoSuchTarget), errors.Is(err, subprobe.ErrNoOwnedNodes):
		kind = fault.Missing
	case errors.Is(err, subprobe.ErrNoNodesTestable), errors.Is(err, clashapi.ErrDisabled), errors.Is(err, clashapi.ErrUnauthorized):
		kind = fault.Unavailable
	}
	return &fault.Error{Kind: kind, Message: probeErrorMessage(err), Cause: err}
}
