package dualstack

// Actually sending the traffic.
//
// The dial targets a LITERAL address, never a name. That is the whole point:
// the resolve phase already established what sing-box returns for A and for
// AAAA, and re-resolving here would let the host's own resolver — or the
// happy-eyeballs logic in a name-based dial — silently choose a family and
// hide the one being tested. It is the same reason the shell script reaches
// for `curl --resolve` rather than `curl -6`.

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"time"
)

// Reachability is the outcome of testing one address family.
//
// Four states, not two. The two extra ones are the difference between a useful
// report and a misleading one: a proxied domain SHOULD have no AAAA (that is
// what an IPv6 split does), and a panel on a host with no global IPv6 cannot
// test IPv6 at all. Reporting either as "unreachable" would flag a working
// config as broken.
type Reachability string

const (
	// ReachabilityOK means the address was dialled successfully.
	ReachabilityOK Reachability = "reachable"
	// ReachabilityFailed means the dial was attempted and did not succeed.
	ReachabilityFailed Reachability = "unreachable"
	// ReachabilityNoAddress means DNS returned nothing for this family. For a
	// proxied domain's AAAA this is the CORRECT outcome, not a failure.
	ReachabilityNoAddress Reachability = "no_address"
	// ReachabilityUntested means the panel could not test it: this host has no
	// global address of that family, or dialling was switched off.
	ReachabilityUntested Reachability = "untested"
)

// defaultDialTimeout bounds one dial.
const defaultDialTimeout = 8 * time.Second

// DialRequest is one address-family test.
type DialRequest struct {
	Address netip.Addr
	Port    uint16
	// ServerName enables a TLS handshake with this SNI once TCP is up. A bare
	// TCP connect proves a path exists; a completed handshake proves the path
	// reaches the right server, which is what distinguishes real egress from
	// a transparent proxy answering on its behalf.
	ServerName string
	Timeout    time.Duration
	// Hold runs while the connection is still OPEN, and is the only way the
	// correlation step can work: sing-box's /connections lists ACTIVE
	// connections only, so a connect-and-close is gone before the next poll
	// and would correlate against nothing every time. It receives the local
	// address, which is what identifies this connection if the destination
	// alone ever proves ambiguous.
	Hold func(local string)
}

// DialResult is what happened.
type DialResult struct {
	Status    Reachability `json:"status"`
	ElapsedMS int64        `json:"elapsed_ms"`
	// Error carries the underlying failure, for ReachabilityFailed.
	Error string `json:"error,omitempty"`
	// Peer is the address actually connected to, as the kernel reports it.
	// Worth showing: an IPv4-mapped peer on an IPv6 test means the dial did
	// not go out over IPv6 at all.
	Peer string `json:"peer,omitempty"`
	// TLSHandshake reports that a TLS handshake completed, when one was asked
	// for. A TCP connect with a failed handshake is NOT a working path.
	TLSHandshake bool `json:"tls_handshake,omitempty"`
}

// Dial connects to one literal address and reports what happened.
//
// It never returns an error: every failure is a Status the caller renders. A
// diagnostic that aborts on the first unreachable address cannot report on the
// other family, which is the comparison being asked for.
func Dial(ctx context.Context, request DialRequest) DialResult {
	if !request.Address.IsValid() {
		return DialResult{Status: ReachabilityNoAddress}
	}
	timeout := request.Timeout
	if timeout <= 0 {
		timeout = defaultDialTimeout
	}

	// Pin the network to the address's family. "tcp" would let the resolver
	// stack substitute the other one for an IPv4-mapped literal, which is
	// exactly the false PASS this whole feature exists to avoid.
	network := "tcp4"
	if request.Address.Is6() && !request.Address.Is4In6() {
		network = "tcp6"
	}
	target := net.JoinHostPort(request.Address.String(), strconv.Itoa(int(request.Port)))

	dialCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	started := time.Now()
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(dialCtx, network, target)
	if err != nil {
		return DialResult{
			Status:    ReachabilityFailed,
			ElapsedMS: time.Since(started).Milliseconds(),
			Error:     err.Error(),
		}
	}
	defer conn.Close()

	result := DialResult{Status: ReachabilityOK, Peer: conn.RemoteAddr().String()}

	if request.ServerName != "" {
		if err := handshake(dialCtx, conn, request.ServerName); err != nil {
			result.Status = ReachabilityFailed
			result.Error = fmt.Sprintf("tls handshake failed: %s", err)
			result.ElapsedMS = time.Since(started).Milliseconds()
			return result
		}
		result.TLSHandshake = true
	}

	result.ElapsedMS = time.Since(started).Milliseconds()

	// Measured BEFORE the hold: the elapsed time is how long the path took to
	// come up, and folding in however long the observer polled would report a
	// fast connection as a slow one.
	if request.Hold != nil {
		request.Hold(conn.LocalAddr().String())
	}
	return result
}

// handshake completes TLS over an established connection.
//
// InsecureSkipVerify is deliberate and narrow: the question is whether bytes
// reach the intended server over this family, not whether its certificate
// chains to a root this router trusts. A device with a stale CA bundle would
// otherwise report every destination as unreachable, which is a different
// problem wearing this one's clothes. The SNI is still sent, so the server
// still has to be the one being asked for.
func handshake(ctx context.Context, conn net.Conn, serverName string) error {
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName:         serverName,
		InsecureSkipVerify: true,
	})
	return tlsConn.HandshakeContext(ctx)
}

// HasGlobalAddress reports whether this host holds a routable address of the
// given family.
//
// Without it, an IPv6 test on an IPv4-only host reports every destination as
// unreachable and reads as "your IPv6 egress is broken" — when in fact nothing
// was tested. The script calls the same distinction SKIP.
func HasGlobalAddress(v6 bool) bool {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return false
	}
	parsed := make([]netip.Addr, 0, len(addrs))
	for _, entry := range addrs {
		prefix, ok := entry.(*net.IPNet)
		if !ok {
			continue
		}
		if address, ok := netip.AddrFromSlice(prefix.IP); ok {
			parsed = append(parsed, address.Unmap())
		}
	}
	return hasGlobalAddress(parsed, v6)
}

// hasGlobalAddress is HasGlobalAddress's pure half, so the classification is
// testable without depending on the machine running the tests.
//
// "Global" here means routable off-link, which includes RFC1918 and ULA: a
// router NATs the former and may route the latter, so excluding them would
// declare every ordinary LAN host untestable.
func hasGlobalAddress(addrs []netip.Addr, v6 bool) bool {
	for _, address := range addrs {
		// Unmap first, so an IPv4-mapped ::ffff:a.b.c.d is classified as the
		// IPv4 address it is rather than counting as IPv6 connectivity — the
		// precise confusion that makes `curl -6` lie on an IPv4-only host.
		address = address.Unmap()
		if v6 != address.Is6() {
			continue
		}
		if address.IsLoopback() || address.IsLinkLocalUnicast() ||
			address.IsLinkLocalMulticast() || address.IsUnspecified() {
			continue
		}
		return true
	}
	return false
}
