package dualstack

// Finding the probe's own connection in sing-box's connection table.
//
// This is what turns a prediction into an observation: the route probe says
// where a destination WOULD go, and this says where it DID. Both are needed,
// because the interesting case is when they disagree.
//
// The matching is deliberately stricter than the shell script's, which selects
// the first connection whose destination is of the right family. On a router
// that is also carrying a browser, a media box and an update daemon, that
// picks up somebody else's connection and reports its rule as the probe's —
// a wrong answer that looks exactly like a right one.

import (
	"net/netip"
	"strconv"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/integrations/clashapi"
)

// startSkew is how far BEFORE the dial a connection may be stamped and still
// count as this dial's.
//
// sing-box stamps Start from its own clock, and the panel reads its own; the
// two are the same machine but not the same instant. Without a window, a
// connection created microseconds "before" the dial began is discarded, which
// throws away the exact connection being looked for. It is kept short because
// its only other job is to exclude connections that were genuinely already
// open, and those are typically seconds or minutes old.
const startSkew = 500 * time.Millisecond

// Observed is what sing-box actually did with the probe's connection.
type Observed struct {
	// Rule is sing-box's own rule string, verbatim:
	// "<rule.String()> => <action.String()>", or the literal "final".
	// There is no rule index in the payload.
	Rule string `json:"rule"`
	// Outbound is the outbound the RULE named — the last element of `chains`,
	// which sing-box reverses.
	Outbound string `json:"outbound"`
	// Via is the leaf actually dialled (chains[0]). For a selector or urltest
	// it is the member that was elected, which is the thing an operator
	// usually wants and which Outbound alone does not say.
	Via string `json:"via,omitempty"`
	// Inbound is the tag the connection arrived on, parsed out of
	// metadata.type ("tun/tun-in").
	Inbound string `json:"inbound,omitempty"`
	// Host is the sniffed name, when sing-box sniffed one.
	Host string `json:"host,omitempty"`
}

// Correlate finds the connection this probe opened, in one snapshot.
//
// Returns nil when nothing matches, which is a real outcome and not an error:
// a short-lived dial can be closed again before the next poll, and sing-box
// exposes only ACTIVE connections.
func Correlate(
	snapshot *clashapi.Snapshot, address netip.Addr, port uint16, dialedAt time.Time,
) *Observed {
	if snapshot == nil || !address.IsValid() {
		return nil
	}

	wantPort := strconv.Itoa(int(port))
	floor := dialedAt.Add(-startSkew)

	var best *clashapi.Connection
	for i := range snapshot.Connections {
		candidate := &snapshot.Connections[i]

		if candidate.Metadata.DestinationPort != wantPort {
			continue
		}
		// Compare parsed addresses, not text: sing-box prints the address it
		// parsed, and "2001:0db8::1" and "2001:db8::1" are the same address
		// spelled two ways.
		parsed, err := netip.ParseAddr(candidate.Metadata.DestinationIP)
		if err != nil || parsed.Unmap() != address.Unmap() {
			continue
		}
		if candidate.Start.Before(floor) {
			// Already open before the probe ran, so its rule says nothing
			// about this dial.
			continue
		}
		// Several may qualify if the probe is re-run quickly; the newest is
		// the one just opened.
		if best == nil || candidate.Start.After(best.Start) {
			best = candidate
		}
	}

	if best == nil {
		return nil
	}

	return &Observed{
		Rule:     best.Rule,
		Outbound: lastChain(best.Chains),
		Via:      firstChain(best.Chains),
		Inbound:  inboundTag(best.Metadata.Type),
		Host:     best.Metadata.Host,
	}
}

// lastChain is the outbound the rule named. sing-box reverses `chains`, so
// this is the END of the slice — reading index 0 gives the leaf instead, which
// on a urltest group is a different tag every few minutes.
func lastChain(chains []string) string {
	if len(chains) == 0 {
		return ""
	}
	return chains[len(chains)-1]
}

// firstChain is the leaf actually dialled. Reported separately from the
// outbound, and omitted when they are the same node.
func firstChain(chains []string) string {
	if len(chains) < 2 {
		return ""
	}
	return chains[0]
}

// inboundTag pulls the tag out of metadata.type, which sing-box formats as
// "<inboundType>/<inboundTag>" — or as a bare type when the inbound is
// untagged, in which case there is no tag to report.
func inboundTag(metadataType string) string {
	for i := len(metadataType) - 1; i >= 0; i-- {
		if metadataType[i] == '/' {
			return metadataType[i+1:]
		}
	}
	return ""
}
