package settings

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/logger"
	"go.uber.org/zap"
)

// Subscription quality-probe settings.
//
// These live in the generic settings table rather than in app.yml because the
// two that matter — how often to sample and how much to keep — are the disk
// budget, and the disk they are budgeting varies from a 32 MB router overlay to
// a workstation SSD. An operator has to be able to change them from the panel
// on the box that is filling up, without an SSH session and a restart.
const (
	KeyProbeInterval      = "probe_interval_seconds"
	KeyProbeTimeout       = "probe_timeout_ms"
	KeyProbeRetentionDays = "probe_retention_days"
	KeyProbeMaxPoints     = "probe_max_points"
)

// Defaults and bounds. These mirror app/pkg/subprobe's own clamps: the prober
// re-normalizes whatever it is handed, so a value that slipped past here (a row
// written by an older build, or edited in the database directly) still cannot
// make it spin. Duplicated deliberately — this layer is where a BAD value is
// reported to the operator, and the prober's layer is where it is survived.
const (
	DefaultProbeInterval = 10 * time.Minute
	MinProbeInterval     = time.Minute
	MaxProbeInterval     = 24 * time.Hour

	DefaultProbeTimeoutMs = 5000
	MinProbeTimeoutMs     = 1000
	// Capped by the Clash API client's own 10s request timeout: a longer probe
	// timeout would be cut off by the transport before sing-box could report
	// its own 504, turning every slow node into an unexplained error.
	MaxProbeTimeoutMs = 8000

	DefaultProbeRetentionDays = 7
	MinProbeRetentionDays     = 1
	MaxProbeRetentionDays     = 90

	DefaultProbeMaxPoints = 2016 // 7 days at the default 10-minute interval
	MinProbeMaxPoints     = 60
	MaxProbeMaxPoints     = 20000
)

// Validate* check a value without writing it.
//
// Separate from the Set* functions because these four knobs are edited as a
// GROUP: each Set commits its own transaction, so validating inside them means
// a request whose third field is out of range has already persisted the first
// two. The caller validates all four, then writes all four.
func ValidateProbeInterval(d time.Duration) error {
	if d < MinProbeInterval || d > MaxProbeInterval {
		return fmt.Errorf("probe interval must be between %s and %s", MinProbeInterval, MaxProbeInterval)
	}
	return nil
}

// ValidateProbeTimeoutMs bounds the per-node URL-test timeout.
func ValidateProbeTimeoutMs(ms int) error {
	if ms < MinProbeTimeoutMs || ms > MaxProbeTimeoutMs {
		return fmt.Errorf("probe timeout must be between %d and %d ms", MinProbeTimeoutMs, MaxProbeTimeoutMs)
	}
	return nil
}

// ValidateProbeRetentionDays bounds the retention window.
func ValidateProbeRetentionDays(days int) error {
	if days < MinProbeRetentionDays || days > MaxProbeRetentionDays {
		return fmt.Errorf("probe retention must be between %d and %d days", MinProbeRetentionDays, MaxProbeRetentionDays)
	}
	return nil
}

// ValidateProbeMaxPoints bounds the per-subscription sample cap.
func ValidateProbeMaxPoints(points int) error {
	if points < MinProbeMaxPoints || points > MaxProbeMaxPoints {
		return fmt.Errorf("probe max points must be between %d and %d", MinProbeMaxPoints, MaxProbeMaxPoints)
	}
	return nil
}

// GetProbeInterval returns the sweep interval, clamped, defaulting when unset.
func (m *ManagerXORM) GetProbeInterval() time.Duration {
	seconds := m.getInt(KeyProbeInterval, int(DefaultProbeInterval/time.Second))
	interval := time.Duration(seconds) * time.Second
	if interval < MinProbeInterval {
		return MinProbeInterval
	}
	if interval > MaxProbeInterval {
		return MaxProbeInterval
	}
	return interval
}

// SetProbeInterval validates and stores the sweep interval.
func (m *ManagerXORM) SetProbeInterval(d time.Duration) error {
	if err := ValidateProbeInterval(d); err != nil {
		return err
	}
	return m.Set(KeyProbeInterval, strconv.Itoa(int(d/time.Second)))
}

// GetProbeTimeout returns the per-node URL-test timeout.
func (m *ManagerXORM) GetProbeTimeout() time.Duration {
	ms := clampInt(m.getInt(KeyProbeTimeout, DefaultProbeTimeoutMs), MinProbeTimeoutMs, MaxProbeTimeoutMs)
	return time.Duration(ms) * time.Millisecond
}

// SetProbeTimeoutMs validates and stores the per-node timeout in milliseconds.
func (m *ManagerXORM) SetProbeTimeoutMs(ms int) error {
	if err := ValidateProbeTimeoutMs(ms); err != nil {
		return err
	}
	return m.Set(KeyProbeTimeout, strconv.Itoa(ms))
}

// GetProbeRetentionDays returns how long samples are kept.
func (m *ManagerXORM) GetProbeRetentionDays() int {
	return clampInt(m.getInt(KeyProbeRetentionDays, DefaultProbeRetentionDays), MinProbeRetentionDays, MaxProbeRetentionDays)
}

// SetProbeRetentionDays validates and stores the retention window.
func (m *ManagerXORM) SetProbeRetentionDays(days int) error {
	if err := ValidateProbeRetentionDays(days); err != nil {
		return err
	}
	return m.Set(KeyProbeRetentionDays, strconv.Itoa(days))
}

// GetProbeMaxPoints returns the per-subscription sample cap.
func (m *ManagerXORM) GetProbeMaxPoints() int {
	return clampInt(m.getInt(KeyProbeMaxPoints, DefaultProbeMaxPoints), MinProbeMaxPoints, MaxProbeMaxPoints)
}

// SetProbeMaxPoints validates and stores the per-subscription sample cap.
func (m *ManagerXORM) SetProbeMaxPoints(points int) error {
	if err := ValidateProbeMaxPoints(points); err != nil {
		return err
	}
	return m.Set(KeyProbeMaxPoints, strconv.Itoa(points))
}

// getInt reads an integer setting, falling back to `fallback` when the key is
// absent or holds something that is not an integer. A malformed value is logged
// rather than returned as an error: these are read on the probe's hot path, and
// a single bad row must not stop the sweep.
func (m *ManagerXORM) getInt(key string, fallback int) int {
	raw, err := m.Get(key)
	if err != nil {
		if !errors.Is(err, ErrSettingNotFound) {
			logger.Warn("failed to read setting, using default",
				zap.String("key", key), zap.Error(err))
		}
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		logger.Warn("setting is not an integer, using default",
			zap.String("key", key), zap.String("value", raw), zap.Error(err))
		return fallback
	}
	return n
}

func clampInt(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
