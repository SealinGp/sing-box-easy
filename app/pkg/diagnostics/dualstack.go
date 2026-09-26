package diagnostics

// The dual-stack probe's service layer: it assembles the collaborators the
// pure package declares as interfaces, and owns their lifetimes.
//
// Everything optional degrades rather than fails. A stopped sing-box costs the
// live answer and the correlation but not the config read; a config the pinned
// schema cannot decode costs the prediction but not the dial. A diagnostic
// that refuses to run when one of its four sources is unavailable is least
// useful exactly when it is most needed.

import (
	"context"
	"errors"
	"net/netip"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/dns"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/dualstack"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/route"
	"github.com/SealinGp/sing-box-easy/app/pkg/diagnostics/ruleset"
	"github.com/SealinGp/sing-box-easy/app/pkg/fault"
	"github.com/SealinGp/sing-box-easy/app/pkg/integrations/clashapi"
	"github.com/sagernet/sing-box/option"
)

// DualStackRequest asks whether a domain works over both address families.
type DualStackRequest struct {
	// Domains is the list to test, capped by dualstack.MaxDomains.
	Domains []string `json:"domains"`
	// Port defaults to 443.
	Port uint16 `json:"port"`
	// TLS completes a handshake after connecting, proving the path reaches
	// the intended server rather than merely reaching something.
	TLS bool `json:"tls"`
	// SkipDial reports DNS and prediction only, sending no traffic.
	SkipDial bool `json:"skip_dial"`
	// TimeoutMS bounds one dial. Zero uses the package default.
	TimeoutMS int `json:"timeout_ms"`
}

// maxDialTimeout bounds what a caller may ask for.
//
// A request that dials 10 domains over 2 families serialises up to 20 dials,
// so an unbounded per-dial timeout is an unbounded request — and this one
// holds an HTTP goroutine and a connection open for its whole length.
const maxDialTimeout = 15 * time.Second

// DualStackRun captures a config projection and the clients built from it.
//
// Like DNSRun it owns a rule-set loader that may hold a handle on sing-box's
// cache file, so every caller MUST Close it.
type DualStackRun struct {
	endpoint dualstack.DNSEndpoint
	options  dualstack.Options
	sets     *ruleset.Loader
}

func (h *Service) PrepareDualStack(req DualStackRequest) (*DualStackRun, error) {
	// Raw sections, not the decoded schema: this walk reads only `action`,
	// `inbound`, `listen` and `listen_port`, all stable across versions, and
	// must keep working against a config a newer sing-box accepts.
	routeRaw, _, routeErr := h.configManager.GetConfigSection("route")
	inboundsRaw, _, inboundErr := h.configManager.GetConfigSection("inbounds")
	if routeErr != nil {
		return nil, fault.New(fault.Configuration, routeErr.Error())
	}
	if inboundErr != nil {
		return nil, fault.New(fault.Configuration, inboundErr.Error())
	}

	timeout := time.Duration(req.TimeoutMS) * time.Millisecond
	if timeout > maxDialTimeout {
		timeout = maxDialTimeout
	}

	sets := h.ruleSetLoader()
	run := &DualStackRun{
		endpoint: dualstack.DiscoverDNSEndpoint(routeRaw, inboundsRaw),
		sets:     sets,
		options: dualstack.Options{
			Domains:  req.Domains,
			Port:     req.Port,
			TLS:      req.TLS,
			SkipDial: req.SkipDial,
			Timeout:  timeout,
		},
	}

	// Both live sources come from the same Clash API. Absent, the probe still
	// reports the config's intent; it just cannot say what happened.
	if clash, err := h.readClashAPISettings(); err == nil {
		if client, clientErr := clashapi.NewFromValues(clash.ExternalController, clash.Secret); clientErr == nil {
			run.options.Observe = client
		}
		if resolver, resolverErr := dns.NewClashClientFromValues(
			clash.ExternalController, clash.Secret); resolverErr == nil {
			run.options.Resolve = clashResolverAdapter{resolver}
		}
	}

	if options, err := h.routeOptions(); err == nil {
		run.options.Predict = h.predictor(options, sets)
	}

	return run, nil
}

// Close releases the rule-set loader's cache handle.
func (r *DualStackRun) Close() {
	if r != nil && r.sets != nil {
		r.sets.Close()
		r.sets = nil
	}
}

func (r *DualStackRun) Run(ctx context.Context) (*dualstack.Result, error) {
	return dualstack.Run(ctx, r.endpoint, r.options)
}

func (r *DualStackRun) Stream(
	ctx context.Context, emit func(dualstack.Stage, *dualstack.Result) error,
) (*dualstack.Result, error) {
	return dualstack.RunStaged(ctx, r.endpoint, r.options, emit)
}

// routeOptions projects the config for the route walk, degrading to an error
// the caller drops rather than one it surfaces: no prediction is a smaller
// loss than no probe.
func (h *Service) routeOptions() (*option.Options, error) {
	cfg, err := h.configManager.GetConfigSubset("route", "outbounds", "endpoints")
	if err != nil {
		return nil, err
	}
	return &cfg.Options, nil
}

// predictor runs the offline route walk for one literal address.
//
// No resolver is supplied, and that is correct rather than a shortcut: the
// address is already known — it is the one about to be dialled — so asking
// sing-box to resolve anything here could only introduce a second, different
// address and predict for traffic that will never be sent.
func (h *Service) predictor(options *option.Options, sets *ruleset.Loader) dualstack.Predictor {
	clashMode := ""
	if clash, err := h.readClashAPISettings(); err == nil {
		if client, clientErr := dns.NewClashClientFromValues(
			clash.ExternalController, clash.Secret); clientErr == nil {
			if mode, modeErr := client.Mode(); modeErr == nil {
				clashMode = mode
			}
		}
	}

	return func(address netip.Addr, port uint16) (*dualstack.Prediction, error) {
		result, err := route.Run(options, route.Options{
			Destination: address.String(),
			Port:        port,
			Network:     route.DefaultNetwork,
			ClashMode:   clashMode,
			Sets:        sets,
		})
		if err != nil {
			return nil, err
		}
		return &dualstack.Prediction{
			Outbound:     result.Outbound,
			Source:       result.OutboundSource,
			MatchedIndex: result.MatchedIndex,
			Exact:        result.Exact,
		}, nil
	}
}

// clashResolverAdapter narrows the DNS probe's client to the two values the
// dual-stack probe needs, so that package depends on an interface it declares
// rather than on the DNS probe's result type.
type clashResolverAdapter struct {
	client *dns.ClashClient
}

func (a clashResolverAdapter) Query(name, qType string) ([]string, int64, error) {
	live, err := a.client.Query(name, qType)
	if err != nil {
		return nil, 0, err
	}
	if live == nil {
		return nil, 0, errors.New("no response from sing-box")
	}

	addresses := make([]string, 0, len(live.Answers))
	for _, answer := range live.Answers {
		// CNAMEs ride along in an A/AAAA response; only address records are
		// dialable, and a CNAME in the list would be parsed as neither family
		// and silently dropped later anyway. Filtering here keeps the reported
		// `addresses` honest about what was actually returned as an address.
		if _, parseErr := netip.ParseAddr(answer.Data); parseErr != nil {
			continue
		}
		addresses = append(addresses, answer.Data)
	}
	return addresses, live.ElapsedMS, nil
}
