package lanclients

import (
	"errors"
	"os"
	"reflect"
	"testing"
)

// Fixtures are verbatim shapes from an iStoreOS 24.10 router.
const leasesFixture = `1790521368 e4:fe:43:d4:eb:42 192.168.31.194 xiaomi-blanket-mj3_mibtEB42 01:e4:fe:43:d4:eb:42
1790539697 94:8c:d7:1c:0f:5e 192.168.31.208 * *
garbage line
duid 00:01:00:01:aa:bb:cc:dd:ee:ff
`

const arpFixture = `IP address       HW type     Flags       HW address            Mask     Device
192.168.31.208   0x1         0x0         94:8c:d7:1c:0f:5e     *        br-lan
192.168.31.209   0x1         0x2         aa:21:e7:a7:28:e4     *        br-lan
192.168.31.1     0x1         0x0         00:00:00:00:00:00     *        br-lan
`

const uciFixture = `dhcp.@dnsmasq[0]=dnsmasq
dhcp.@dnsmasq[0].leasefile='/tmp/custom.leases'
dhcp.@host[0]=host
dhcp.@host[0].enabled='1'
dhcp.@host[0].mac='00:19:0F:34:84:73'
dhcp.@host[0].name='LeozhangFnOS1'
dhcp.@host[0].ip='192.168.31.240'
dhcp.@host[1]=host
dhcp.@host[1].mac='AA:21:E7:A7:28:E4' '11:22:33:44:55:66'
dhcp.@host[1].name='nas'
dhcp.@host[2]=host
dhcp.@host[2].enabled='0'
dhcp.@host[2].mac='de:ad:be:ef:00:01'
`

func TestParseLeases(t *testing.T) {
	got := parseLeases([]byte(leasesFixture))
	want := []Client{
		{MAC: "e4:fe:43:d4:eb:42", IP: "192.168.31.194", Hostname: "xiaomi-blanket-mj3_mibtEB42"},
		{MAC: "94:8c:d7:1c:0f:5e", IP: "192.168.31.208"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestParseARP(t *testing.T) {
	got := parseARP([]byte(arpFixture))
	want := []Client{
		{MAC: "94:8c:d7:1c:0f:5e", IP: "192.168.31.208", Device: "br-lan"},
		{MAC: "aa:21:e7:a7:28:e4", IP: "192.168.31.209", Device: "br-lan", Online: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestParseUCI(t *testing.T) {
	hosts, leasefile := parseUCI(uciFixture)
	if leasefile != "/tmp/custom.leases" {
		t.Fatalf("leasefile = %q", leasefile)
	}
	want := []Client{
		{MAC: "00:19:0f:34:84:73", IP: "192.168.31.240", Hostname: "LeozhangFnOS1"},
		{MAC: "aa:21:e7:a7:28:e4", Hostname: "nas"},
		{MAC: "11:22:33:44:55:66", Hostname: "nas"},
	}
	if !reflect.DeepEqual(hosts, want) {
		t.Fatalf("got %#v\nwant %#v", hosts, want)
	}
}

func TestCollectMergesSources(t *testing.T) {
	files := map[string]string{"/tmp/custom.leases": leasesFixture, arpPath: arpFixture}
	c := &Collector{
		enabled: true,
		readFile: func(p string) ([]byte, error) {
			if s, ok := files[p]; ok {
				return []byte(s), nil
			}
			return nil, os.ErrNotExist
		},
		run: func(string, ...string) (string, error) { return uciFixture, nil },
	}
	res := c.Collect()
	if !res.Supported || len(res.Warnings) != 0 {
		t.Fatalf("supported=%v warnings=%v", res.Supported, res.Warnings)
	}
	byMAC := map[string]Client{}
	for _, cl := range res.Clients {
		byMAC[cl.MAC] = cl
	}
	if len(byMAC) != 5 {
		t.Fatalf("want 5 unique clients, got %d: %#v", len(res.Clients), res.Clients)
	}
	// Static name wins over the neighbour table, which contributes liveness.
	nas := byMAC["aa:21:e7:a7:28:e4"]
	if nas.Hostname != "nas" || nas.IP != "192.168.31.209" || !nas.Online ||
		!reflect.DeepEqual(nas.Sources, []string{SourceStatic, SourceNeighbor}) {
		t.Fatalf("nas merged wrong: %#v", nas)
	}
	washer := byMAC["94:8c:d7:1c:0f:5e"]
	if washer.Online || !reflect.DeepEqual(washer.Sources, []string{SourceLease, SourceNeighbor}) {
		t.Fatalf("washer merged wrong: %#v", washer)
	}
	// Online clients sort first.
	if !res.Clients[0].Online {
		t.Fatalf("online client should sort first: %#v", res.Clients[0])
	}
}

func TestCollectDegrades(t *testing.T) {
	off := &Collector{enabled: false}
	if res := off.Collect(); res.Supported || len(res.Clients) != 0 {
		t.Fatalf("disabled collector must report unsupported and empty: %#v", res)
	}

	broken := &Collector{
		enabled:  true,
		readFile: func(string) ([]byte, error) { return nil, os.ErrPermission },
		run:      func(string, ...string) (string, error) { return "", errors.New("uci: not found") },
	}
	res := broken.Collect()
	if !res.Supported || len(res.Clients) != 0 || len(res.Warnings) != 3 {
		t.Fatalf("every source failing must be reported, not fatal: %#v", res)
	}
}
