package dnsprobe

import (
	"testing"

	"github.com/sagernet/sing-box/option"
)

func udpServer(tag, host string, port uint16, detour string) option.DNSServerOptions {
	options := &option.RemoteDNSServerOptions{}
	options.Server = host
	options.ServerPort = port
	options.Detour = detour
	return option.DNSServerOptions{Type: "udp", Tag: tag, Options: options}
}

func TestDescribeServer(t *testing.T) {
	servers := []option.DNSServerOptions{
		udpServer("dns_router", "192.168.9.2", 53, ""),
		// Port omitted: the transport default must be applied.
		udpServer("dns_plain", "1.1.1.1", 0, "proxy-out"),
		{Type: "hosts", Tag: "dns_lan", Options: &option.HostsDNSServerOptions{}},
	}

	t.Run("address and port", func(t *testing.T) {
		got := DescribeServer(servers, "dns_router")
		if got == nil || got.Address != "192.168.9.2:53" || !got.Found {
			t.Fatalf("got %+v, want 192.168.9.2:53", got)
		}
	})

	t.Run("default port and detour", func(t *testing.T) {
		got := DescribeServer(servers, "dns_plain")
		if got.Address != "1.1.1.1:53" {
			t.Errorf("Address = %q, want 1.1.1.1:53", got.Address)
		}
		if got.Detour != "proxy-out" {
			t.Errorf("Detour = %q, want proxy-out", got.Detour)
		}
	})

	t.Run("local server has no address", func(t *testing.T) {
		got := DescribeServer(servers, "dns_lan")
		if !got.Found || got.Address != "" {
			t.Errorf("got %+v, want found with no address", got)
		}
	})

	t.Run("unknown tag is reported, not blanked", func(t *testing.T) {
		got := DescribeServer(servers, "nope")
		if got == nil || got.Found {
			t.Fatalf("got %+v, want Found=false", got)
		}
	})

	t.Run("empty tag yields nothing", func(t *testing.T) {
		if got := DescribeServer(servers, ""); got != nil {
			t.Errorf("got %+v, want nil", got)
		}
	})
}
