package trafficflow

import (
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Aggregate folds live connections into one Frame.
//
// Every accumulator here is keyed by something the expected-flow diagram also
// has a node for — an inbound tag, a rule index, an outbound tag — which is
// what lets the browser overlay the frame on the drawing without a second
// layout. The one thing the drawing does not have a node for is the leaf a
// group dialled through; that rides along as an exit's `via`.
func Aggregate(at time.Time, live []Live, index *RuleIndex, filter Filter) *Frame {
	frame := &Frame{
		At:       at.UnixMilli(),
		Filtered: !filter.Empty(),
		Inbounds: []InboundFlow{},
		Rules:    []RuleFlow{},
		Exits:    []ExitFlow{},
		Sources:  []SourceFlow{},
	}
	frame.Totals.All = len(live)

	inbounds := map[string]*InboundFlow{}
	sources := map[string]*SourceFlow{}
	rules := map[string]*ruleAccumulator{}
	exits := map[string]*exitAccumulator{}

	for i := range live {
		conn := &live[i]

		// Before the filter, deliberately: the picker this feeds must keep
		// offering every client while narrowed to one of them.
		if ip := conn.Metadata.SourceIP; ip != "" {
			src := sources[ip]
			if src == nil {
				src = &SourceFlow{IP: ip}
				sources[ip] = src
			}
			src.Down += conn.DownRate
			src.Up += conn.UpRate
			src.Connections++
		}

		if !filter.admits(conn) {
			continue
		}

		frame.Totals.Connections++
		frame.Totals.Down += conn.DownRate
		frame.Totals.Up += conn.UpRate

		inTag := inboundTag(conn.Metadata.Type)
		in := inbounds[inTag]
		if in == nil {
			in = &InboundFlow{Tag: inTag}
			inbounds[inTag] = in
		}
		in.Down += conn.DownRate
		in.Up += conn.UpRate
		in.Connections++

		exit, via := exitOf(conn.Chains)

		key, flow := classifyRule(conn.Rule, index, exit)
		acc := rules[key]
		if acc == nil {
			acc = &ruleAccumulator{RuleFlow: flow, hosts: map[string]float64{}}
			rules[key] = acc
		}
		acc.Down += conn.DownRate
		acc.Up += conn.UpRate
		acc.Connections++
		acc.hosts[hostOf(conn)] += conn.DownRate
		if flow.Kind == KindUnmatched {
			frame.Unmatched++
		}

		if exit == "" {
			continue
		}
		ex := exits[exit]
		if ex == nil {
			ex = &exitAccumulator{ExitFlow: ExitFlow{Tag: exit}, via: map[string]*ViaFlow{}}
			exits[exit] = ex
		}
		ex.Down += conn.DownRate
		ex.Up += conn.UpRate
		ex.Connections++
		if via != "" {
			v := ex.via[via]
			if v == nil {
				v = &ViaFlow{Tag: via}
				ex.via[via] = v
			}
			v.Down += conn.DownRate
			v.Connections++
		}
	}

	frame.Sources = finishSources(sources)

	for _, in := range inbounds {
		frame.Inbounds = append(frame.Inbounds, *in)
	}
	sort.Slice(frame.Inbounds, func(i, j int) bool { return frame.Inbounds[i].Tag < frame.Inbounds[j].Tag })

	for _, acc := range rules {
		frame.Rules = append(frame.Rules, acc.finish())
	}
	sortByDown(len(frame.Rules), func(i int) float64 { return frame.Rules[i].Down }, func(i, j int) {
		frame.Rules[i], frame.Rules[j] = frame.Rules[j], frame.Rules[i]
	})

	for _, ex := range exits {
		frame.Exits = append(frame.Exits, ex.finish())
	}
	sortByDown(len(frame.Exits), func(i int) float64 { return frame.Exits[i].Down }, func(i, j int) {
		frame.Exits[i], frame.Exits[j] = frame.Exits[j], frame.Exits[i]
	})

	return frame
}

// finishSources caps the list to the busiest addresses and then orders it by
// address, not by rate: a picker whose entries reshuffle every second is one
// an operator cannot click reliably.
func finishSources(sources map[string]*SourceFlow) []SourceFlow {
	list := make([]SourceFlow, 0, len(sources))
	for _, src := range sources {
		list = append(list, *src)
	}
	if len(list) > maxSources {
		sort.SliceStable(list, func(i, j int) bool { return list[i].Connections > list[j].Connections })
		list = list[:maxSources]
	}
	sort.SliceStable(list, func(i, j int) bool { return lessSourceIP(list[i].IP, list[j].IP) })
	return list
}

// lessSourceIP orders addresses numerically — "192.168.1.9" before
// "192.168.1.10", which a string comparison reverses — and puts anything that
// is not an address last, in string order.
func lessSourceIP(a, b string) bool {
	addrA, errA := netip.ParseAddr(a)
	addrB, errB := netip.ParseAddr(b)
	switch {
	case errA == nil && errB == nil:
		return addrA.Compare(addrB) < 0
	case errA == nil:
		return true
	case errB == nil:
		return false
	default:
		return a < b
	}
}

type ruleAccumulator struct {
	RuleFlow
	hosts map[string]float64
}

func (a *ruleAccumulator) finish() RuleFlow {
	flow := a.RuleFlow
	flow.Hosts = make([]HostFlow, 0, len(a.hosts))
	for host, down := range a.hosts {
		flow.Hosts = append(flow.Hosts, HostFlow{Host: host, Down: down})
	}
	sortByDown(len(flow.Hosts), func(i int) float64 { return flow.Hosts[i].Down }, func(i, j int) {
		flow.Hosts[i], flow.Hosts[j] = flow.Hosts[j], flow.Hosts[i]
	})
	if len(flow.Hosts) > maxHostsPerRule {
		flow.Hosts = flow.Hosts[:maxHostsPerRule]
	}
	return flow
}

type exitAccumulator struct {
	ExitFlow
	via map[string]*ViaFlow
}

func (a *exitAccumulator) finish() ExitFlow {
	flow := a.ExitFlow
	flow.Via = make([]ViaFlow, 0, len(a.via))
	for _, v := range a.via {
		flow.Via = append(flow.Via, *v)
	}
	sortByDown(len(flow.Via), func(i int) float64 { return flow.Via[i].Down }, func(i, j int) {
		flow.Via[i], flow.Via[j] = flow.Via[j], flow.Via[i]
	})
	return flow
}

// sortByDown orders highest rate first, stably, so equal rates keep the order
// they arrived in.
func sortByDown(n int, down func(int) float64, swap func(int, int)) {
	sort.Stable(&byDown{n: n, down: down, swap: swap})
}

type byDown struct {
	n    int
	down func(int) float64
	swap func(int, int)
}

func (b *byDown) Len() int           { return b.n }
func (b *byDown) Less(i, j int) bool { return b.down(i) > b.down(j) }
func (b *byDown) Swap(i, j int)      { b.swap(i, j) }

// classifyRule decides which rule row a connection lights, and returns the key
// the accumulator groups on.
func classifyRule(rule string, index *RuleIndex, exit string) (string, RuleFlow) {
	if rule == FinalRule {
		return "final", RuleFlow{Kind: KindFinal, Index: -1, Exit: exit}
	}
	if i, ok := index.Lookup(rule); ok {
		return "rule:" + strconv.Itoa(i), RuleFlow{Kind: KindRule, Index: i, Exit: exit}
	}
	return "unmatched:" + rule, RuleFlow{Kind: KindUnmatched, Index: -1, Rule: rule, Exit: exit}
}

// exitOf reads a reversed chain: the last element is the outbound the rule
// named, the first is the leaf actually used. They coincide for a leaf
// outbound, in which case there is no `via`.
func exitOf(chains []string) (exit, via string) {
	if len(chains) == 0 {
		return "", ""
	}
	exit = chains[len(chains)-1]
	if len(chains) > 1 {
		via = chains[0]
	}
	return exit, via
}

// inboundTag reads sing-box's `<type>/<tag>`; an untagged inbound reports the
// bare type, which is then the best label available.
func inboundTag(metadataType string) string {
	if i := strings.IndexByte(metadataType, '/'); i >= 0 && i+1 < len(metadataType) {
		return metadataType[i+1:]
	}
	return metadataType
}

// hostOf is the sniffed host, else the destination address.
func hostOf(conn *Live) string {
	if conn.Metadata.Host != "" {
		return conn.Metadata.Host
	}
	return conn.Metadata.DestinationIP
}

func (f Filter) admits(conn *Live) bool {
	if f.SourceIP != "" && conn.Metadata.SourceIP != f.SourceIP {
		return false
	}
	if f.Host != "" {
		needle := strings.ToLower(f.Host)
		if !strings.Contains(strings.ToLower(conn.Metadata.Host), needle) &&
			!strings.Contains(strings.ToLower(conn.Metadata.DestinationIP), needle) {
			return false
		}
	}
	return true
}
