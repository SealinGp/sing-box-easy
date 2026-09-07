package config

import "testing"

func TestParseCoreVersion(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   CoreVersion
	}{
		{
			name:   "stable",
			output: "sing-box version 1.14.0\nEnvironment: go1.25 linux/amd64\n",
			want:   CoreVersion{Major: 1, Minor: 14, Patch: 0, Raw: "1.14.0"},
		},
		{
			name:   "prerelease and vendor suffix",
			output: "sing-box version 1.14.0-beta.3-openwrt\n",
			want:   CoreVersion{Major: 1, Minor: 14, Patch: 0, Raw: "1.14.0-beta.3-openwrt"},
		},
		{
			name:   "leading v",
			output: "sing-box version v1.13.2\n",
			want:   CoreVersion{Major: 1, Minor: 13, Patch: 2, Raw: "v1.13.2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCoreVersion(tt.output)
			if err != nil {
				t.Fatalf("ParseCoreVersion() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("ParseCoreVersion() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestCapabilitiesForCoreVersion(t *testing.T) {
	for _, tt := range []struct {
		version     CoreVersion
		supported   bool
		dnsEvaluate bool
	}{
		{version: CoreVersion{Major: 1, Minor: 12, Patch: 12}, supported: true, dnsEvaluate: false},
		{version: CoreVersion{Major: 1, Minor: 13, Patch: 9}, supported: true, dnsEvaluate: false},
		{version: CoreVersion{Major: 1, Minor: 14, Patch: 0}, supported: true, dnsEvaluate: true},
		{version: CoreVersion{Major: 2, Minor: 0, Patch: 0}, supported: false, dnsEvaluate: true},
	} {
		got := CapabilitiesForCoreVersion(tt.version)
		if got.Supported != tt.supported {
			t.Errorf("CapabilitiesForCoreVersion(%+v).Supported = %v, want %v", tt.version, got.Supported, tt.supported)
		}
		if got.DNSEvaluate != tt.dnsEvaluate {
			t.Errorf("CapabilitiesForCoreVersion(%+v).DNSEvaluate = %v, want %v", tt.version, got.DNSEvaluate, tt.dnsEvaluate)
		}
		if got.DNSRespond != tt.dnsEvaluate || got.DNSRace != tt.dnsEvaluate || got.DNSMatchResponse != tt.dnsEvaluate {
			t.Errorf("1.14 DNS capabilities are inconsistent for %+v: %+v", tt.version, got)
		}
	}
}
