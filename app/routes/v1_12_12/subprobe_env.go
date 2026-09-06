package v1_13_0

import (
	"fmt"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/clashapi"
	"github.com/SealinGp/sing-box-easy/app/pkg/config"
	"github.com/SealinGp/sing-box-easy/app/pkg/settings"
	"github.com/SealinGp/sing-box-easy/app/pkg/subprobe"
	"github.com/SealinGp/sing-box-easy/app/pkg/subscription"
)

// probeEnvironment adapts the panel's managers to subprobe.Environment.
//
// It lives here rather than inside subprobe so the prober depends on three
// small method sets instead of on the config manager, the settings store and
// the Clash client — which is what keeps its tests free of a database and a
// running sing-box.
//
// Every method resolves from live state on each call. That is the contract the
// runner relies on: an operator can add a subscription, change the interval or
// restart sing-box between two sweeps, and the next sweep must see all three
// without the panel being restarted.
type probeEnvironment struct {
	subscriptions subscription.SubscriptionManager
	configManager *config.Manager
	settings      *settings.ManagerXORM
}

// Targets lists the subscriptions with probing enabled.
func (e *probeEnvironment) Targets() ([]subprobe.Target, error) {
	subs, err := e.subscriptions.List()
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	targets := make([]subprobe.Target, 0, len(subs))
	for _, sub := range subs {
		if !sub.ProbeEnabled {
			continue
		}
		targets = append(targets, subprobe.Target{
			SubID: sub.ID,
			// Resolved (and re-validated) at read time: a row written before
			// the https rule existed must not reach sing-box, which would
			// silently swap it and measure a different endpoint.
			URL: subscription.EffectiveProbeURL(sub.ProbeURL),
		})
	}
	return targets, nil
}

// OutboundTags returns every outbound tag in the config on disk.
//
// The config, not the running instance: the config is what says which nodes a
// subscription owns. Whether the running sing-box has them is a separate
// question, and the prober answers it per node — a tag the controller does not
// know is reported as untestable rather than as down.
func (e *probeEnvironment) OutboundTags() ([]string, error) {
	cfg, err := e.configManager.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	tags := make([]string, 0, len(cfg.Outbounds))
	for _, outbound := range cfg.Outbounds {
		if outbound.Tag != "" {
			tags = append(tags, outbound.Tag)
		}
	}
	return tags, nil
}

// Prober builds a client against the running sing-box.
func (e *probeEnvironment) Prober() (subprobe.Prober, error) {
	cfg, err := e.configManager.GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}
	client, err := clashapi.New(cfg.Options.Experimental)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// Settings reads the current knobs.
func (e *probeEnvironment) Settings() subprobe.Settings {
	return subprobe.Settings{
		Interval:  e.settings.GetProbeInterval(),
		Timeout:   e.settings.GetProbeTimeout(),
		MaxAge:    24 * time.Hour * time.Duration(e.settings.GetProbeRetentionDays()),
		MaxPoints: e.settings.GetProbeMaxPoints(),
		// Concurrency is deliberately not operator-tunable. It trades probe
		// duration against load on a device that is also routing traffic, and
		// there is no way for an operator to observe that trade-off from the
		// panel — so it is a constant chosen for the worst case (a 233-node
		// config on a router).
		Concurrency: subprobe.DefaultConcurrency,
	}
}
