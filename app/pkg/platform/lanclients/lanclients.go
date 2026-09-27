// Package lanclients lists the devices on the router's LAN, for the route
// rule pickers behind `source_mac_address` and `source_hostname`.
//
// Three sources, merged by MAC:
//
//   - dnsmasq's lease file — MAC, IP and the hostname the client announced
//     over DHCP. The same name sing-box matches `source_hostname` against.
//   - static DHCP hosts (`uci show dhcp`, `@host` sections) — operator-named
//     devices, including ones not online right now.
//   - the kernel neighbour table (/proc/net/arp) — clients with a hand-set
//     IP that never took a lease. It is where sing-box itself resolves
//     `source_mac_address`, so anything here is matchable.
//
// Everything is read, nothing is written, and only the uci call spawns a
// process. Each source fails independently: a missing lease file is a
// warning next to whatever the other two found, never an empty answer.
// Only enabled on OpenWrt, where all three exist.
package lanclients

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"sort"
	"strings"
)

const (
	SourceLease    = "lease"
	SourceStatic   = "static"
	SourceNeighbor = "neighbor"

	arpPath          = "/proc/net/arp"
	defaultLeaseFile = "/tmp/dhcp.leases"
	// ATF_COM: the neighbour entry is complete, i.e. the host answered
	// recently. Entries without it keep the last-known MAC (FAILED/STALE).
	arpFlagComplete = 0x2
)

// Client is one LAN device. MAC is lower-case, colon-separated.
type Client struct {
	MAC      string   `json:"mac"`
	IP       string   `json:"ip,omitempty"`
	Hostname string   `json:"hostname,omitempty"`
	Device   string   `json:"device,omitempty"`
	Online   bool     `json:"online"`
	Sources  []string `json:"sources"`
}

// Result is the payload of GET /system/lan-clients.
type Result struct {
	// Supported is false off OpenWrt; the UI then falls back to free text.
	Supported bool     `json:"supported"`
	Clients   []Client `json:"clients"`
	// Warnings names each source that could not be read.
	Warnings []string `json:"warnings,omitempty"`
}

// Runner executes a host command; swappable for tests.
type Runner func(name string, args ...string) (string, error)

type Collector struct {
	enabled  bool
	readFile func(string) ([]byte, error)
	run      Runner
}

// New returns a collector; enabled should be true only on OpenWrt.
func New(enabled bool) *Collector {
	return &Collector{enabled: enabled, readFile: os.ReadFile, run: execRun}
}

func execRun(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "", fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return string(out), nil
}

// Collect reads every source and merges them. Never fails as a whole.
func (c *Collector) Collect() Result {
	if !c.enabled {
		return Result{Supported: false, Clients: []Client{}}
	}
	var warnings []string
	merged := newMerger()

	var static []Client
	leaseFile := defaultLeaseFile
	if out, err := c.run("uci", "-q", "show", "dhcp"); err != nil {
		warnings = append(warnings, "static DHCP hosts: "+err.Error())
	} else {
		var configured string
		static, configured = parseUCI(out)
		if configured != "" {
			leaseFile = configured
		}
	}
	merged.add(static, SourceStatic)

	if data, err := c.readFile(leaseFile); err != nil {
		warnings = append(warnings, "DHCP leases: "+err.Error())
	} else {
		merged.add(parseLeases(data), SourceLease)
	}

	if data, err := c.readFile(arpPath); err != nil {
		warnings = append(warnings, "neighbour table: "+err.Error())
	} else {
		merged.add(parseARP(data), SourceNeighbor)
	}

	return Result{Supported: true, Clients: merged.sorted(), Warnings: warnings}
}

// normalizeMAC returns the canonical lower-case form, or "" when s is not a
// usable 48-bit MAC (including the all-zero placeholder of an incomplete entry).
func normalizeMAC(s string) string {
	hw, err := net.ParseMAC(strings.TrimSpace(s))
	if err != nil || len(hw) != 6 {
		return ""
	}
	mac := hw.String()
	if mac == "00:00:00:00:00:00" {
		return ""
	}
	return mac
}

// parseLeases reads dnsmasq's lease file:
//
//	<expiry> <mac> <ip> <hostname|*> <client-id|*>
//
// The `duid` line (DHCPv6 server id) and anything malformed are skipped.
func parseLeases(data []byte) []Client {
	var clients []Client
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		mac := normalizeMAC(fields[1])
		if mac == "" {
			continue
		}
		hostname := fields[3]
		if hostname == "*" {
			hostname = ""
		}
		clients = append(clients, Client{MAC: mac, IP: fields[2], Hostname: hostname})
	}
	return clients
}

// parseARP reads /proc/net/arp. The header row and incomplete entries (no
// MAC) are skipped; entries that kept a MAC but are not complete are kept as
// offline — the device exists, it just has not answered lately.
func parseARP(data []byte) []Client {
	var clients []Client
	for i, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if i == 0 || len(fields) < 6 {
			continue
		}
		mac := normalizeMAC(fields[3])
		if mac == "" {
			continue
		}
		var flags int
		fmt.Sscanf(fields[2], "0x%x", &flags)
		clients = append(clients, Client{
			MAC:    mac,
			IP:     fields[0],
			Device: fields[5],
			Online: flags&arpFlagComplete != 0,
		})
	}
	return clients
}

// parseUCI extracts enabled static hosts and the dnsmasq lease file path
// from `uci -q show dhcp`. A host's `mac` may list several addresses, each
// quoted: `mac='AA:..' 'BB:..'`.
func parseUCI(out string) (hosts []Client, leaseFile string) {
	type host struct {
		macs     []string
		name, ip string
		disabled bool
	}
	byID := map[string]*host{}
	var order []string

	for _, line := range strings.Split(out, "\n") {
		key, value, ok := strings.Cut(strings.TrimSpace(line), "=")
		if !ok {
			continue
		}
		if strings.HasPrefix(key, "dhcp.@dnsmasq[0].leasefile") && leaseFile == "" {
			leaseFile = unquote(value)
			continue
		}
		if !strings.HasPrefix(key, "dhcp.@host[") {
			continue
		}
		id, option, _ := strings.Cut(key, "].")
		h, seen := byID[id]
		if !seen {
			h = &host{}
			byID[id] = h
			order = append(order, id)
		}
		switch option {
		case "mac":
			for _, part := range strings.Fields(value) {
				if mac := normalizeMAC(unquote(part)); mac != "" {
					h.macs = append(h.macs, mac)
				}
			}
		case "name":
			h.name = unquote(value)
		case "ip":
			h.ip = unquote(value)
		case "enabled":
			h.disabled = unquote(value) == "0"
		}
	}

	for _, id := range order {
		h := byID[id]
		if h.disabled {
			continue
		}
		for _, mac := range h.macs {
			hosts = append(hosts, Client{MAC: mac, IP: h.ip, Hostname: h.name})
		}
	}
	return hosts, leaseFile
}

func unquote(s string) string { return strings.Trim(strings.TrimSpace(s), "'") }

// merger folds entries by MAC. Sources are added in priority order (static,
// lease, neighbour), so the first non-empty hostname — the operator's own
// name for a device — wins; IP and liveness prefer the neighbour table,
// which is the only source describing the present.
type merger struct {
	byMAC map[string]*Client
	order []string
}

func newMerger() *merger { return &merger{byMAC: map[string]*Client{}} }

func (m *merger) add(clients []Client, source string) {
	for _, in := range clients {
		cur, ok := m.byMAC[in.MAC]
		if !ok {
			next := in
			next.Sources = []string{source}
			m.byMAC[in.MAC] = &next
			m.order = append(m.order, in.MAC)
			continue
		}
		if cur.Hostname == "" {
			cur.Hostname = in.Hostname
		}
		if in.IP != "" && (cur.IP == "" || source == SourceNeighbor) {
			cur.IP = in.IP
		}
		if in.Device != "" {
			cur.Device = in.Device
		}
		cur.Online = cur.Online || in.Online
		if !contains(cur.Sources, source) {
			cur.Sources = append(cur.Sources, source)
		}
	}
}

// sorted puts online devices first, then named before unnamed, then by name/MAC.
func (m *merger) sorted() []Client {
	out := make([]Client, 0, len(m.order))
	for _, mac := range m.order {
		out = append(out, *m.byMAC[mac])
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Online != b.Online {
			return a.Online
		}
		if (a.Hostname == "") != (b.Hostname == "") {
			return a.Hostname != ""
		}
		if a.Hostname != b.Hostname {
			return strings.ToLower(a.Hostname) < strings.ToLower(b.Hostname)
		}
		return a.MAC < b.MAC
	})
	return out
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}
