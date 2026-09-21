package dualstack

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"
)

func TestHasGlobalAddressIgnoresLoopbackAndLinkLocal(t *testing.T) {
	// The reason this exists: `curl -6` on a host with no global IPv6 quietly
	// returns an IPv4-mapped address and succeeds, so a naive test reports a
	// false PASS for a direct domain and a false FAIL for a proxied one. The
	// honest outcome on such a host is "untested", and that needs this check.
	cases := []struct {
		name  string
		addrs []string
		v6    bool
		want  bool
	}{
		{"only loopback v6", []string{"::1"}, true, false},
		{"only link-local v6", []string{"fe80::1"}, true, false},
		{"global v6", []string{"fe80::1", "2001:db8::1"}, true, true},
		{"unique local counts", []string{"fd00::1"}, true, true},
		{"v4 does not satisfy v6", []string{"192.168.1.5"}, true, false},
		{"private v4 counts", []string{"192.168.1.5"}, false, true},
		{"only loopback v4", []string{"127.0.0.1"}, false, false},
		{"v6 does not satisfy v4", []string{"2001:db8::1"}, false, false},
		{"nothing at all", nil, true, false},
	}

	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			addrs := make([]netip.Addr, 0, len(test.addrs))
			for _, raw := range test.addrs {
				addrs = append(addrs, netip.MustParseAddr(raw))
			}
			if got := hasGlobalAddress(addrs, test.v6); got != test.want {
				t.Fatalf("hasGlobalAddress(%v, v6=%v) = %v, want %v",
					test.addrs, test.v6, got, test.want)
			}
		})
	}
}

func TestDialReportsAReachableAddress(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen locally: %v", err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addrPort := netip.MustParseAddrPort(listener.Addr().String())
	result := Dial(context.Background(), DialRequest{
		Address: addrPort.Addr(),
		Port:    addrPort.Port(),
		Timeout: 2 * time.Second,
	})

	if result.Status != ReachabilityOK {
		t.Fatalf("status = %q, error = %q", result.Status, result.Error)
	}
	if result.Peer == "" {
		t.Fatal("expected the peer address to be reported")
	}
}

func TestDialReportsAnUnreachableAddress(t *testing.T) {
	// Port 1 on loopback with nothing bound: refused immediately, so this
	// does not depend on a timeout elapsing.
	result := Dial(context.Background(), DialRequest{
		Address: netip.MustParseAddr("127.0.0.1"),
		Port:    1,
		Timeout: 2 * time.Second,
	})

	if result.Status != ReachabilityFailed {
		t.Fatalf("status = %q, want %q", result.Status, ReachabilityFailed)
	}
	if result.Error == "" {
		t.Fatal("a failure must carry its reason")
	}
}

func TestDialRejectsAnInvalidAddress(t *testing.T) {
	result := Dial(context.Background(), DialRequest{Port: 443, Timeout: time.Second})
	if result.Status != ReachabilityNoAddress {
		t.Fatalf("status = %q, want %q", result.Status, ReachabilityNoAddress)
	}
}

func TestDialHonoursACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := Dial(ctx, DialRequest{
		Address: netip.MustParseAddr("192.0.2.1"), // TEST-NET-1, never routable
		Port:    443,
		Timeout: 30 * time.Second,
	})
	if result.Status != ReachabilityFailed {
		t.Fatalf("status = %q, want %q", result.Status, ReachabilityFailed)
	}
}

func TestDialRunsHoldWhileTheConnectionIsOpen(t *testing.T) {
	// The correlation step depends on this: sing-box lists only ACTIVE
	// connections, so the probe's own connection has to still be open when
	// /connections is read, or every dual-stack report would show "no
	// connection observed" regardless of what actually happened.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("cannot listen locally: %v", err)
	}
	defer listener.Close()

	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		accepted <- conn
	}()

	addrPort := netip.MustParseAddrPort(listener.Addr().String())
	openDuringHold := false
	result := Dial(context.Background(), DialRequest{
		Address: addrPort.Addr(),
		Port:    addrPort.Port(),
		Timeout: 2 * time.Second,
		Hold: func(local string) {
			if local == "" {
				t.Error("hold should receive the local address")
			}
			select {
			case conn := <-accepted:
				// Still readable from the server side => still open.
				conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				buf := make([]byte, 1)
				_, err := conn.Read(buf)
				openDuringHold = err == nil || !isClosed(err)
				conn.Close()
			case <-time.After(time.Second):
				t.Error("server never accepted the probe's connection")
			}
		},
	})

	if result.Status != ReachabilityOK {
		t.Fatalf("status = %q", result.Status)
	}
	if !openDuringHold {
		t.Fatal("the connection was already closed when Hold ran")
	}
}

// isClosed reports whether the error means the peer went away, as opposed to
// a read timing out on a connection that is perfectly healthy and just idle.
func isClosed(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return false
	}
	return true
}
