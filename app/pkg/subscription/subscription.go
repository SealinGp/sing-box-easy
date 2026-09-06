package subscription

import (
	"time"
)

type SubscriptionManager interface {
	Init() error
	List() ([]*Subscription, error)
	Get(id string) (*Subscription, error)
	Add(sub Subscription) error
	Update(id string, sub Subscription) error
	Delete(id string) error
	UpdateLastUpdate(id string) error
	UpdateInfo(id string, info []SubInfo) error
	UpdateOfficialURL(id string, officialURL string) error
}

const (
	DefaultSubscriptionPath = "/etc/sing-box/subscriptions.json"
)

// SubInfo is one generic account-metadata entry extracted from a subscription's
// "info nodes" (loopback-server pseudo-nodes whose name is a "key：value" pair,
// e.g. "剩余流量：4.59 TB", "套餐到期：2026-10-19"). Keys are provider-defined and
// language-specific, so they are stored verbatim rather than mapped to fixed
// fields.
type SubInfo struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Fetch modes control how a subscription's URL is retrieved — useful on
// censored networks where the default direct fetch is DNS-poisoned or RST'd.
const (
	// FetchModeDirect is the default: fetch with the host's normal resolver and
	// routing (which, with sing-box in TUN mode, already tunnels the request).
	FetchModeDirect = ""
	// FetchModeCleanDNS resolves the subscription host over DoH (un-poisonable)
	// and dials the resulting IP directly, preserving SNI. Non-proxy.
	FetchModeCleanDNS = "clean_dns"
	// FetchModeProxy fetches through a user-supplied proxy (ProxyURL), falling
	// back to a direct fetch if the proxy is unreachable.
	FetchModeProxy = "proxy"
)

// Subscription represents a node subscription
type Subscription struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	AutoUpdate     bool      `json:"auto_update"`
	UpdateInterval string    `json:"update_interval"` // e.g., "24h"
	LastUpdate     time.Time `json:"last_update"`
	Info           []SubInfo `json:"info,omitempty"`
	// FetchMode selects how the URL is retrieved (see FetchMode* constants).
	FetchMode string `json:"fetch_mode"`
	// ProxyURL is the proxy to use when FetchMode is "proxy", e.g.
	// "socks5://127.0.0.1:7893" or "http://127.0.0.1:7890". Empty otherwise.
	ProxyURL string `json:"proxy_url"`
	// OfficialURL is the provider's own site — where an operator tops up or
	// renews. Auto-filled from the feed on refresh (see DetectOfficialURL) only
	// while it is empty, so an operator's own edit is never overwritten by a
	// provider that reports a different page.
	OfficialURL string `json:"official_url"`
	// ProbeEnabled turns the quality prober on for this subscription. Default
	// true for a new subscription: the metric is the reason the feature exists
	// and an opt-in chart is an empty chart on every fresh install. Turning it
	// off is how an operator excludes a feed they do not want probed (a metered
	// one, or a provider that rate-limits).
	ProbeEnabled bool `json:"probe_enabled"`
	// ProbeURL is the https latency target for THIS subscription. Empty means
	// DefaultProbeURL. Per-subscription because a provider whose nodes are
	// blocked from Google reads as 0% available against gstatic while working
	// perfectly — the operator has to be able to point the test somewhere the
	// feed can actually reach.
	//
	// Must be https: NormalizeProbeURL explains why at length.
	ProbeURL  string    `json:"probe_url"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
