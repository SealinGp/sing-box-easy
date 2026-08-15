package openwrtnet

import "testing"

// Real `uci show firewall` output from an ImmortalWrt 23.05.1 box, trimmed.
const uciShowFirewall = `firewall.@defaults[0]=defaults
firewall.@defaults[0].input='ACCEPT'
firewall.@zone[0]=zone
firewall.@zone[0].name='lan'
firewall.@zone[0].network='lan'
firewall.@zone[1]=zone
firewall.@zone[1].name='wan'
firewall.@zone[1].masq='1'
firewall.@zone[2]=zone
firewall.@zone[2].name='sing-box-easy'
firewall.@zone[2].device='tun0'
firewall.@forwarding[0]=forwarding
firewall.@forwarding[0].src='lan'
firewall.@forwarding[0].dest='wan'
firewall.@forwarding[1]=forwarding
firewall.@forwarding[1].src='lan'
firewall.@forwarding[1].dest='sing-box-easy'
`

func TestFindAnonSection(t *testing.T) {
	tests := []struct {
		name              string
		section, key, val string
		want              string
		wantFound         bool
	}{
		{
			name: "our zone", section: "zone", key: "name", val: "sing-box-easy",
			want: "firewall.@zone[2]", wantFound: true,
		},
		{
			name: "our forwarding", section: "forwarding", key: "dest", val: "sing-box-easy",
			want: "firewall.@forwarding[1]", wantFound: true,
		},
		{
			// Must not confuse the zone named "wan" with a forwarding to it.
			name: "wan zone not the wan forwarding", section: "zone", key: "name", val: "wan",
			want: "firewall.@zone[1]", wantFound: true,
		},
		{
			name: "absent", section: "zone", key: "name", val: "nope", wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, found := findAnonSection(uciShowFirewall, "firewall", tt.section, tt.key, tt.val)
			if found != tt.wantFound {
				t.Fatalf("found = %v, want %v", found, tt.wantFound)
			}
			if found && got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseUCIList(t *testing.T) {
	tests := []struct {
		name string
		out  string
		want []string
	}{
		{name: "single value", out: "127.0.0.1#7874\n", want: []string{"127.0.0.1#7874"}},
		{
			// `uci -q get` prints a list space-separated on one line.
			name: "space separated list",
			out:  "223.5.5.5 119.29.29.29\n",
			want: []string{"223.5.5.5", "119.29.29.29"},
		},
		{name: "empty", out: "\n", want: nil},
		{name: "whitespace only", out: "   \n", want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseUCIList(tt.out)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
