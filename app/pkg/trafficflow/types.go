// Package trafficflow turns sing-box's connection snapshots into the frames
// the Overview's live traffic overlay draws.
//
// The expected-flow diagram (frontend `RouteFlowDiagram`) is a fixed picture
// of the config: inbounds → rules → outbounds. This package lights it. Each
// frame says, for every edge in that picture, how many bytes per second are
// crossing it RIGHT NOW and through how many connections — aggregated here so
// the browser receives a few kilobytes per second rather than one kilobyte per
// connection per second, and so the Clash API secret never leaves the server.
//
// THREE THINGS THE RAW PAYLOAD DOES NOT SAY
// ─────────────────────────────────────────
// Speed: a connection carries cumulative bytes only. `Differ` derives a rate
// from two samples and the wall-clock time between them — per second, not per
// tick, so a poll that slips does not read as a burst.
//
// Rule index: a connection names its rule as a STRING (`rule.String() => action`).
// `RuleIndex` maps that back to a config position using sing-box's own
// `/rules` list, which is the same slice in the same order.
//
// The exit: `chains` is reversed — the outbound the rule chose is the LAST
// element, and the first is the leaf node the bytes actually used. The frame
// reports both, as the exit and its `via`.
package trafficflow

import "github.com/SealinGp/sing-box-easy/app/pkg/clashapi"

// Filter narrows the connections a frame is built from.
//
// Both fields are what an operator types while chasing "why is THIS device /
// THIS site slow"; matching is deliberately loose on the host and exact on the
// source, because a source is an address and a host is a search term.
type Filter struct {
	// SourceIP matches `metadata.sourceIP` exactly.
	SourceIP string
	// Host is a case-insensitive substring of the sniffed host, falling back
	// to the destination IP for connections that never had a domain.
	Host string
}

// Empty reports whether the filter admits everything.
func (f Filter) Empty() bool {
	return f.SourceIP == "" && f.Host == ""
}

// Live is one connection with its rates derived across two samples.
type Live struct {
	clashapi.Connection
	// DownRate and UpRate are bytes per second since the previous sample.
	DownRate float64
	UpRate   float64
	// Fresh marks a first sighting: there is no baseline, so the rates are 0
	// and must not be read as "idle".
	Fresh bool
}

// Frame is one tick of the live overlay, as sent to the browser.
type Frame struct {
	// At is the sample time, unix milliseconds.
	At     int64  `json:"at"`
	Totals Totals `json:"totals"`
	// Inbounds, Rules and Exits are the three edge families of the diagram.
	// Rules and Exits are sorted by download rate, highest first, so a client
	// wanting the top N takes a prefix.
	Inbounds []InboundFlow `json:"inbounds"`
	Rules    []RuleFlow    `json:"rules"`
	Exits    []ExitFlow    `json:"exits"`
	// Sources are the client addresses currently holding connections, for the
	// filter picker. Collected BEFORE the filter — see SourceFlow.
	Sources []SourceFlow `json:"sources"`
	// Filtered is true when a Filter narrowed this frame.
	Filtered bool `json:"filtered"`
	// Unmatched counts connections whose rule string has no index — the
	// running rule list differs from the one the panel mapped, which happens
	// when the config was edited after sing-box started.
	Unmatched int `json:"unmatched"`
}

// Totals are frame-wide sums.
type Totals struct {
	// Down and Up are bytes per second over the connections in this frame.
	Down float64 `json:"down"`
	Up   float64 `json:"up"`
	// Connections is the count after filtering; All is the count before.
	Connections int `json:"connections"`
	All         int `json:"all"`
	// Closed is how many connections vanished since the previous sample.
	Closed int `json:"closed"`
}

// SourceFlow is one client address currently holding connections.
//
// It exists so the operator picks a device from a list instead of typing an
// address the panel already knows: the sources of the moment are exactly the
// candidates worth filtering on.
//
// Collected BEFORE the filter is applied, and that is load-bearing. A list
// built from the filtered connections collapses to the one selected address
// the instant it is chosen, leaving the picker with a single entry and no way
// back to another device.
type SourceFlow struct {
	IP          string  `json:"ip"`
	Down        float64 `json:"down"`
	Up          float64 `json:"up"`
	Connections int     `json:"connections"`
}

// InboundFlow is traffic entering through one inbound.
type InboundFlow struct {
	Tag         string  `json:"tag"`
	Down        float64 `json:"down"`
	Up          float64 `json:"up"`
	Connections int     `json:"connections"`
}

// RuleFlow is traffic that one rule decided.
type RuleFlow struct {
	// Kind is "rule" (Index is valid), "final" (fell through) or "unmatched"
	// (Rule carries the raw string sing-box reported and Index is -1).
	Kind  string `json:"kind"`
	Index int    `json:"index"`
	Rule  string `json:"rule,omitempty"`
	// Exit is the outbound the rule sent traffic to — the last chain element.
	Exit        string     `json:"exit"`
	Down        float64    `json:"down"`
	Up          float64    `json:"up"`
	Connections int        `json:"connections"`
	Hosts       []HostFlow `json:"hosts"`
}

// HostFlow is one destination's share of a rule's traffic.
type HostFlow struct {
	Host string  `json:"host"`
	Down float64 `json:"down"`
}

// ExitFlow is traffic leaving through one outbound the rules named.
type ExitFlow struct {
	Tag         string  `json:"tag"`
	Down        float64 `json:"down"`
	Up          float64 `json:"up"`
	Connections int     `json:"connections"`
	// Via lists the leaf nodes a group actually dialled through. Empty for a
	// leaf outbound (the exit IS the leaf).
	Via []ViaFlow `json:"via"`
}

// ViaFlow is one leaf node's share of an exit's traffic.
type ViaFlow struct {
	Tag         string  `json:"tag"`
	Down        float64 `json:"down"`
	Connections int     `json:"connections"`
}

// Rule kinds, as written to the wire.
const (
	KindRule      = "rule"
	KindFinal     = "final"
	KindUnmatched = "unmatched"
)

// FinalRule is what sing-box writes in `rule` when no rule matched.
const FinalRule = "final"

// maxSources bounds the source picker's list. A busy tun inbound on a router
// can see every device on the LAN plus the router itself; past a few dozen the
// list is scrolled, not read, and the ones worth filtering on are the busy
// ones. The cap keeps the busiest and the picker still sorts by address.
const maxSources = 64

// maxHostsPerRule bounds the per-rule host list. Three is what fits a tooltip
// and is enough to answer "what is flowing here".
const maxHostsPerRule = 3
