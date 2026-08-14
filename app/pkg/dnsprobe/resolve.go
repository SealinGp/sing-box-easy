package dnsprobe

import (
	"crypto/tls"
	"fmt"
	"net"
	"sort"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/sagernet/sing-box/option"
)

// directTimeout bounds one upstream query. Kept short: a comparison across
// several servers runs them concurrently and a dead server must not stall the
// whole probe.
const directTimeout = 5 * time.Second

// ServerResult is one configured DNS server's answer, used to compare
// upstreams against each other.
type ServerResult struct {
	Tag  string `json:"tag"`
	Type string `json:"type"`
	// Address is the resolver actually dialled, empty when not applicable.
	Address string `json:"address,omitempty"`
	// Skipped explains why a server was not queried (unsupported type, or it
	// is only reachable through a proxy detour this process cannot use).
	Skipped string `json:"skipped,omitempty"`
	Error   string `json:"error,omitempty"`
	// Records holds the comparable payload: the A/AAAA/CNAME values, sorted so
	// two servers returning the same set in a different order still compare
	// equal.
	Records   []string `json:"records"`
	ElapsedMS int64    `json:"elapsed_ms"`
}

// recordTypeName maps a DNS type number to its text form, falling back to the
// numeric form for types miekg does not name.
func recordTypeName(qType int) string {
	if name, ok := dns.TypeToString[uint16(qType)]; ok {
		return name
	}
	return fmt.Sprintf("TYPE%d", qType)
}

// QueryServers asks every directly reachable configured server the same
// question, so their answers can be compared.
//
// Servers behind a `detour` are skipped rather than queried: this process
// would reach them directly, bypassing the proxy sing-box uses, which produces
// an answer sing-box would never have seen. A misleading comparison is worse
// than an absent one.
func QueryServers(servers []option.DNSServerOptions, name, qType string) []ServerResult {
	results := make([]ServerResult, len(servers))

	type job struct {
		index   int
		address string
		useTLS  bool
	}
	var jobs []job

	for i, server := range servers {
		result := ServerResult{Tag: serverTag(server), Type: server.Type, Records: []string{}}

		address, useTLS, detour, reason := dialTarget(server)
		switch {
		case reason != "":
			result.Skipped = reason
		case detour != "":
			result.Skipped = "reachable only through detour " + detour
		default:
			result.Address = address
			jobs = append(jobs, job{index: i, address: address, useTLS: useTLS})
		}
		results[i] = result
	}

	done := make(chan struct{}, len(jobs))
	for _, j := range jobs {
		go func(j job) {
			defer func() { done <- struct{}{} }()
			records, elapsed, err := exchange(j.address, j.useTLS, name, qType)
			if err != nil {
				results[j.index].Error = err.Error()
			}
			results[j.index].Records = records
			results[j.index].ElapsedMS = elapsed
		}(j)
	}
	for range jobs {
		<-done
	}

	return results
}

// dialTarget works out how to reach a configured server directly.
//
// Only plain UDP and DNS-over-TLS are dialled. Types like `hosts`, `fakeip`,
// `dhcp` and `local` have no upstream to compare against, and `https`/`quic`
// would need their own transports for little extra signal.
func dialTarget(server option.DNSServerOptions) (address string, useTLS bool, detour string, skip string) {
	switch server.Type {
	case "udp", "tcp":
		options, ok := server.Options.(*option.RemoteDNSServerOptions)
		if !ok {
			return "", false, "", "unsupported server options"
		}
		port := options.ServerPort
		if port == 0 {
			port = 53
		}
		return net.JoinHostPort(options.Server, fmt.Sprint(port)), false,
			options.Detour, ""
	case "tls":
		options, ok := server.Options.(*option.RemoteTLSDNSServerOptions)
		if !ok {
			return "", false, "", "unsupported server options"
		}
		port := options.ServerPort
		if port == 0 {
			port = 853
		}
		return net.JoinHostPort(options.Server, fmt.Sprint(port)), true,
			options.Detour, ""
	default:
		return "", false, "", "type " + server.Type + " has no comparable upstream"
	}
}

// exchange performs one query and returns the answer values, sorted.
func exchange(address string, useTLS bool, name, qType string) ([]string, int64, error) {
	question, ok := dns.StringToType[qType]
	if !ok {
		return []string{}, 0, fmt.Errorf("unsupported query type %q", qType)
	}

	message := new(dns.Msg)
	message.SetQuestion(dns.Fqdn(name), question)

	client := &dns.Client{Timeout: directTimeout}
	if useTLS {
		client.Net = "tcp-tls"
		host, _, err := net.SplitHostPort(address)
		if err == nil {
			client.TLSConfig = &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
		}
	}

	started := time.Now()
	response, _, err := client.Exchange(message, address)
	elapsed := time.Since(started).Milliseconds()
	if err != nil {
		return []string{}, elapsed, err
	}

	records := make([]string, 0, len(response.Answer))
	for _, rr := range response.Answer {
		records = append(records, recordValue(rr))
	}
	sort.Strings(records)

	return records, elapsed, nil
}

// recordValue renders the comparable part of a record — the value, without the
// TTL, so two servers answering with different TTLs still compare equal.
func recordValue(rr dns.RR) string {
	header := rr.Header()
	value := strings.TrimPrefix(rr.String(), header.String())
	return recordTypeName(int(header.Rrtype)) + " " + strings.TrimSpace(value)
}

// serverTag returns the server's tag, falling back to its type when untagged.
func serverTag(server option.DNSServerOptions) string {
	if server.Tag != "" {
		return server.Tag
	}
	return "(" + server.Type + ")"
}
