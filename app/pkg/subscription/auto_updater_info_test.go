package subscription

import (
	"reflect"
	"testing"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
)

// TestIsInfoNode verifies loopback-server detection (the provider/language-
// agnostic signal) and that real nodes are never misclassified.
func TestIsInfoNode(t *testing.T) {
	cases := []struct {
		name   string
		server string
		want   bool
	}{
		{"ipv4 loopback", "127.0.0.1", true},
		{"ipv6 loopback", "::1", true},
		{"localhost", "localhost", true},
		{"unspecified", "0.0.0.0", true},
		{"localhost mixed case", "LocalHost", true},
		{"real host", "jp-lx.777076.xyz", false},
		{"real ip", "1.2.3.4", false},
		{"empty server", "", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			n := sn("剩余流量：4.59 TB", c.server, 443)
			if got := isInfoNode(n); got != c.want {
				t.Errorf("isInfoNode(server=%q) = %v, want %v", c.server, got, c.want)
			}
		})
	}
	if isInfoNode(nil) {
		t.Error("isInfoNode(nil) = true, want false")
	}
}

// TestParseInfoEntry verifies generic key/value splitting on the first colon,
// covering the fullwidth "：", the ASCII ":", and the no-colon fallback.
func TestParseInfoEntry(t *testing.T) {
	cases := []struct {
		in   string
		want SubInfo
	}{
		{"剩余流量：4.59 TB", SubInfo{Key: "剩余流量", Value: "4.59 TB"}},
		{"套餐到期：2026-10-19", SubInfo{Key: "套餐到期", Value: "2026-10-19"}},
		{"距离下次重置剩余：3 天", SubInfo{Key: "距离下次重置剩余", Value: "3 天"}},
		{"Expire: 2026-10-19", SubInfo{Key: "Expire", Value: "2026-10-19"}},
		// Only the FIRST colon splits; later colons stay in the value.
		{"Reset：12:00 next day", SubInfo{Key: "Reset", Value: "12:00 next day"}},
		{"  spaced ： value ", SubInfo{Key: "spaced", Value: "value"}},
		{"NoColonHere", SubInfo{Key: "NoColonHere", Value: ""}},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			if got := parseInfoEntry(c.in); got != c.want {
				t.Errorf("parseInfoEntry(%q) = %+v, want %+v", c.in, got, c.want)
			}
		})
	}
}

// TestPartitionNodes verifies a mixed feed is split into real proxy nodes and
// parsed info entries, preserving order within each group.
func TestPartitionNodes(t *testing.T) {
	feed := []*node.SubNode{
		sn("🇭🇰 香港 01", "hk.example.com", 443),
		sn("剩余流量：4.59 TB", "127.0.0.1", 443),
		sn("🇯🇵 日本 02", "jp.example.com", 443),
		sn("套餐到期：2026-10-19", "127.0.0.1", 443),
	}

	real, info := partitionNodes(feed)

	if len(real) != 2 || real[0].Tag != "🇭🇰 香港 01" || real[1].Tag != "🇯🇵 日本 02" {
		t.Fatalf("real = %v, want the two non-loopback nodes in order", collectTagsFromNodes(real))
	}
	wantInfo := []SubInfo{
		{Key: "剩余流量", Value: "4.59 TB"},
		{Key: "套餐到期", Value: "2026-10-19"},
	}
	if !reflect.DeepEqual(info, wantInfo) {
		t.Errorf("info = %+v, want %+v", info, wantInfo)
	}
}

// TestMarshalUnmarshalInfo verifies the JSON column round-trips, and that
// empty/invalid encodings decode to nil rather than erroring.
func TestMarshalUnmarshalInfo(t *testing.T) {
	in := []SubInfo{{Key: "剩余流量", Value: "4.59 TB"}, {Key: "套餐到期", Value: "2026-10-19"}}
	encoded, err := marshalInfo(in)
	if err != nil {
		t.Fatalf("marshalInfo error: %v", err)
	}
	if got := unmarshalInfo(encoded); !reflect.DeepEqual(got, in) {
		t.Errorf("round-trip = %+v, want %+v", got, in)
	}

	if got := unmarshalInfo(""); got != nil {
		t.Errorf("unmarshalInfo(\"\") = %+v, want nil", got)
	}
	if got := unmarshalInfo("not json"); got != nil {
		t.Errorf("unmarshalInfo(invalid) = %+v, want nil", got)
	}

	// nil marshals to an empty JSON array (clears stale info).
	if encoded, _ := marshalInfo(nil); encoded != "[]" {
		t.Errorf("marshalInfo(nil) = %q, want %q", encoded, "[]")
	}
}

// TestParseUserinfo verifies the standard Subscription-Userinfo header is turned
// into human-readable entries, using the real header observed from 白月光.
func TestParseUserinfo(t *testing.T) {
	got := parseUserinfo("upload=7532082534; download=88520110292; total=805306368000; expire=1788925517")
	want := []SubInfo{
		{Key: "Used", Value: "89.46 GB"},     // (7532082534+88520110292)/1024^3
		{Key: "Total", Value: "750.00 GB"},   // 805306368000/1024^3
		{Key: "Remaining", Value: "660.54 GB"},
		{Key: "Expires", Value: "2026-09-09"}, // time.Unix(1788925517) in local tz
	}
	// Compare everything except the date, which is timezone-dependent.
	if len(got) != len(want) {
		t.Fatalf("parseUserinfo len = %d (%+v), want %d", len(got), got, len(want))
	}
	for i := 0; i < 3; i++ {
		if got[i] != want[i] {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if got[3].Key != "Expires" || got[3].Value == "" {
		t.Errorf("expires entry = %+v, want a non-empty date", got[3])
	}

	// total=0 means unlimited: only Used (and Expires) should appear.
	unl := parseUserinfo("upload=1; download=1073741823; total=0; expire=0")
	if len(unl) != 1 || unl[0].Key != "Used" {
		t.Errorf("unlimited plan = %+v, want only [Used]", unl)
	}

	if parseUserinfo("") != nil {
		t.Error("parseUserinfo(\"\") should be nil")
	}
	if parseUserinfo("garbage") != nil {
		t.Error("parseUserinfo(garbage) should be nil")
	}
}

// TestHumanizeBytes verifies binary-unit formatting.
func TestHumanizeBytes(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.00 KB"},
		{805306368000, "750.00 GB"},
		{5 * 1024 * 1024 * 1024 * 1024, "5.00 TB"},
	}
	for _, c := range cases {
		if got := humanizeBytes(c.in); got != c.want {
			t.Errorf("humanizeBytes(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func collectTagsFromNodes(nodes []*node.SubNode) []string {
	tags := make([]string, len(nodes))
	for i, n := range nodes {
		tags[i] = n.Tag
	}
	return tags
}
