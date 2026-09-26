package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/dns"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
	configpkg "github.com/SealinGp/sing-box-easy/app/pkg/singbox/config"
	"github.com/sagernet/sing-box/option"
	singjson "github.com/sagernet/sing/common/json"
)

type DNSProbeRequest struct {
	Domain string `json:"domain"`
	// Type is a DNS record type; defaults to A.
	Type string `json:"type"`
	// CompareServers additionally queries each directly reachable configured
	// resolver so their answers can be compared. Off by default because it
	// sends real traffic to every upstream.
	CompareServers bool `json:"compare_servers"`
}

func (h *Service) logTailer() dns.LogTailer {
	return func(lines int, afterCursor string) ([]string, string, error) {
		chunk, err := h.serviceController.TailLogs(lines, afterCursor)
		if err != nil {
			return nil, "", err
		}
		return chunk.Lines, chunk.Cursor, nil
	}
}

// loadDNSProbeConfig projects the dns section twice, for two consumers with
// opposite needs.
//
// The RULE WALK takes the section raw, so a host running a newer sing-box than
// this build still produces a full ladder. The SERVER list has to be typed —
// the live query and the upstream comparison dial those servers — so it is
// narrowed to servers/final/strategy before decoding, keeping a newer
// section-level key (1.14's `optimistic`) away from the strict decoder.
func (h *Service) loadDNSProbeConfig() (*configpkg.SingBoxConfig, json.RawMessage, error) {
	rawDNS, ok, err := h.configManager.GetConfigSection("dns")
	if err != nil {
		return nil, nil, err
	}
	if !ok {
		rawDNS = json.RawMessage("{}")
	}

	cfg := &configpkg.SingBoxConfig{}
	if servers, serversErr := decodeProbeServers(rawDNS); serversErr == nil {
		cfg.DNS = servers
	}
	// A server list that will not decode costs the live query and the upstream
	// comparison, and nothing else. The ladder is unaffected, which is the
	// whole point of reading it raw.

	clash, err := h.readClashAPISettings()
	if err != nil {
		return nil, nil, err
	}
	cfg.Experimental = &option.ExperimentalOptions{ClashAPI: &option.ClashAPIOptions{
		ExternalController: clash.ExternalController,
		Secret:             clash.Secret,
	}}
	return cfg, rawDNS, nil
}

// decodeProbeServers narrows the dns section to the keys the live query and
// the server comparison need, then decodes those through the pinned schema.
//
// Narrowing first is not belt-and-braces: sing-box's decoder is strict, so one
// unrecognised SIBLING key — `optimistic`, or a rule using a 1.14 action —
// rejects the whole section and takes the server list with it.
func decodeProbeServers(rawDNS json.RawMessage) (*option.DNSOptions, error) {
	var section map[string]json.RawMessage
	if err := json.Unmarshal(rawDNS, &section); err != nil {
		return nil, fmt.Errorf("failed to parse dns section: %w", err)
	}

	narrowed := make(map[string]json.RawMessage, 3)
	for _, field := range []string{"servers", "final", "strategy"} {
		if value, exists := section[field]; exists {
			narrowed[field] = value
		}
	}
	encoded, err := json.Marshal(narrowed)
	if err != nil {
		return nil, err
	}

	var options option.DNSOptions
	if err := singjson.UnmarshalContext(
		configpkg.CreateContext(context.Background()), encoded, &options); err != nil {
		return nil, fmt.Errorf("failed to parse DNS probe settings: %w", err)
	}
	return &options, nil
}

// DNSRun captures a consistent configuration projection before streaming starts.
//
// It owns a rule-set loader, which may hold an open handle on sing-box's cache
// file (or a temporary copy of it), so every caller MUST Close the run. The
// loader is built once per run rather than per rule: a real config reaches the
// same sets from many rules, and its memo is what keeps the binary fallback
// affordable.
type DNSRun struct {
	cfg     *configpkg.SingBoxConfig
	options dns.Options
	sets    *ruleset.Loader
}

func (h *Service) PrepareDNS(req DNSProbeRequest) (*DNSRun, error) {
	cfg, rawDNS, err := h.loadDNSProbeConfig()
	if err != nil {
		return nil, err
	}
	sets := h.ruleSetLoader()
	return &DNSRun{
		cfg:  cfg,
		sets: sets,
		options: dns.Options{
			Domain:         req.Domain,
			QueryType:      req.Type,
			CompareServers: req.CompareServers,
			Tailer:         h.logTailer(),
			RawDNS:         rawDNS,
			Sets:           sets,
		},
	}, nil
}

// Close releases the cache-file handle the loader may hold. Safe to call more
// than once.
func (r *DNSRun) Close() {
	if r != nil && r.sets != nil {
		r.sets.Close()
		r.sets = nil
	}
}

func (r *DNSRun) Run() (*dns.Result, error) { return dns.Run(&r.cfg.Options, r.options) }
func (r *DNSRun) Stream(emit func(dns.Stage, *dns.Result) error) (*dns.Result, error) {
	return dns.RunStaged(&r.cfg.Options, r.options, emit)
}
