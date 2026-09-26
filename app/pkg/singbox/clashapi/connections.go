package clashapi

import (
	"context"
	"time"
)

// Connection is one tracked connection as sing-box serialises it
// (experimental/clashapi/trafficontrol/tracker.go, `MarshalJSON`).
//
// Two fields are not what a Clash-family dashboard would lead you to expect,
// because sing-box fills them differently from mihomo:
//
//   - Chains is REVERSED: index 0 is the leaf node the bytes actually went
//     through, and the LAST element is the outbound the rule named — the
//     selector or urltest group. Read it from the end to find the exit.
//   - Rule is `<rule.String()> => <action.String()>`, or the literal "final"
//     when no rule matched. There is no rule index anywhere in the payload;
//     `Rules()` below is how one is recovered. RulePayload is always "".
//
// Verified identical between sing-box 1.12.12 and 1.13.12 (only the
// process-path detail differs), so a host running either is fine.
type Connection struct {
	ID       string             `json:"id"`
	Metadata ConnectionMetadata `json:"metadata"`
	Upload   int64              `json:"upload"`
	Download int64              `json:"download"`
	Start    time.Time          `json:"start"`
	Chains   []string           `json:"chains"`
	Rule     string             `json:"rule"`
}

// ConnectionMetadata is the `metadata` object of a Connection.
type ConnectionMetadata struct {
	Network string `json:"network"`
	// Type is `<inboundType>/<inboundTag>` — "tun/tun-in", "mixed/mixed-in" —
	// or the bare type when the inbound has no tag.
	Type            string `json:"type"`
	SourceIP        string `json:"sourceIP"`
	DestinationIP   string `json:"destinationIP"`
	SourcePort      string `json:"sourcePort"`
	DestinationPort string `json:"destinationPort"`
	// Host is the sniffed domain when there is one, else the destination FQDN.
	Host        string `json:"host"`
	ProcessPath string `json:"processPath"`
}

// Snapshot is the body of `GET /connections`: every ACTIVE connection, plus the
// kernel's lifetime byte totals. Connections through a `dns` outbound are
// excluded by sing-box itself. Closed connections are kept by sing-box (the
// last 1000) but not exposed on any endpoint as of 1.13.
type Snapshot struct {
	DownloadTotal int64        `json:"downloadTotal"`
	UploadTotal   int64        `json:"uploadTotal"`
	Connections   []Connection `json:"connections"`
	Memory        uint64       `json:"memory"`
}

// Connections fetches one snapshot. sing-box serves the same data over a
// WebSocket at a fixed tick; polling this on the panel's own ticker gives the
// same cadence with no upgrade handshake.
func (c *Client) Connections(ctx context.Context) (*Snapshot, error) {
	var snapshot Snapshot
	if err := c.Get(ctx, "/connections", &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// Rule is one entry of `GET /rules`.
//
// sing-box returns `router.Rules()` — the slice itself, in config order — with
// Payload = rule.String() and Proxy = rule.Action().String(). A Connection's
// Rule field is exactly `Payload + " => " + Proxy`, which is what makes the
// index recoverable: the mapping comes from sing-box, not from reimplementing
// every matcher's String().
type Rule struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
}

// Rules fetches the running rule list, in config order.
func (c *Client) Rules(ctx context.Context) ([]Rule, error) {
	var body struct {
		Rules []Rule `json:"rules"`
	}
	if err := c.Get(ctx, "/rules", &body); err != nil {
		return nil, err
	}
	return body.Rules, nil
}
