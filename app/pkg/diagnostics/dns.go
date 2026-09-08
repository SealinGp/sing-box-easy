package diagnostics

import (
	"context"
	"encoding/json"
	"fmt"

	configpkg "github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/dns"
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

func (h *Service) logTailer() dnsprobe.LogTailer {
	return func(lines int, afterCursor string) ([]string, string, error) {
		chunk, err := h.serviceController.TailLogs(lines, afterCursor)
		if err != nil {
			return nil, "", err
		}
		return chunk.Lines, chunk.Cursor, nil
	}
}
func (h *Service) loadDNSProbeConfig() (*configpkg.SingBoxConfig, string, error) {
	cfg, typedErr := h.configManager.GetConfigSubset("dns")
	attributionError := ""
	if typedErr != nil {
		raw, ok, err := h.configManager.GetConfigSection("dns")
		if err != nil {
			return nil, "", err
		}
		if !ok {
			cfg = &configpkg.SingBoxConfig{}
		} else {
			var dnsObject map[string]json.RawMessage
			if err := json.Unmarshal(raw, &dnsObject); err != nil {
				return nil, "", fmt.Errorf("failed to parse dns section: %w", err)
			}
			// Project only the settings consumed by live/comparison probes.
			// Removing rules alone still feeds newer fields (e.g. optimistic)
			// into the compiled core's strict decoder.
			probeSettings := make(map[string]json.RawMessage)
			for _, field := range []string{"servers", "final", "strategy"} {
				if value, exists := dnsObject[field]; exists {
					probeSettings[field] = value
				}
			}
			sanitized, err := json.Marshal(probeSettings)
			if err != nil {
				return nil, "", err
			}
			var dnsOptions option.DNSOptions
			if err := singjson.UnmarshalContext(configpkg.CreateContext(context.Background()), sanitized, &dnsOptions); err != nil {
				return nil, "", fmt.Errorf("failed to parse DNS probe settings: %w", err)
			}
			cfg = &configpkg.SingBoxConfig{}
			cfg.DNS = &dnsOptions
		}
		attributionError = "offline attribution unavailable: DNS configuration is incompatible with the compiled schema; live and log evidence are still authoritative"
	}

	clash, err := h.readClashAPISettings()
	if err != nil {
		return nil, "", err
	}
	cfg.Experimental = &option.ExperimentalOptions{ClashAPI: &option.ClashAPIOptions{
		ExternalController: clash.ExternalController,
		Secret:             clash.Secret,
	}}
	return cfg, attributionError, nil
}

// DNSRun captures a consistent configuration projection before streaming starts.
type DNSRun struct {
	cfg     *configpkg.SingBoxConfig
	options dnsprobe.Options
}

func (h *Service) PrepareDNS(req DNSProbeRequest) (*DNSRun, error) {
	cfg, attribution, err := h.loadDNSProbeConfig()
	if err != nil {
		return nil, err
	}
	return &DNSRun{cfg: cfg, options: dnsprobe.Options{Domain: req.Domain, QueryType: req.Type, CompareServers: req.CompareServers, Tailer: h.logTailer(), AttributionError: attribution}}, nil
}
func (r *DNSRun) Run() (*dnsprobe.Result, error) { return dnsprobe.Run(&r.cfg.Options, r.options) }
func (r *DNSRun) Stream(emit func(dnsprobe.Stage, *dnsprobe.Result) error) (*dnsprobe.Result, error) {
	return dnsprobe.RunStaged(&r.cfg.Options, r.options, emit)
}
