package subscription

import "testing"

func TestNormalizeOfficialURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"https url", "https://www.paofu.cloud", "https://www.paofu.cloud"},
		{"http url", "http://example.com/user", "http://example.com/user"},
		{"path and fragment kept", "https://example.com/#/register", "https://example.com/#/register"},
		{"path case preserved", "https://example.com/User/Login", "https://example.com/User/Login"},
		{"bare domain promoted", "www.example.com", "https://www.example.com"},
		{"bare domain with path", "example.com/buy", "https://example.com/buy"},
		{"scheme relative", "//example.com/buy", "https://example.com/buy"},
		{"uppercase scheme", "HTTPS://example.com", "https://example.com"},
		{"surrounding brackets", "[https://example.com]", "https://example.com"},
		{"trailing chinese period", "https://example.com。", "https://example.com"},
		{"whitespace trimmed", "  https://example.com  ", "https://example.com"},

		// Rejected
		{"empty", "", ""},
		{"blank", "   ", ""},
		{"javascript scheme", "javascript:alert(1)", ""},
		{"data scheme", "data:text/html;base64,PHN2Zz4=", ""},
		{"file scheme", "file:///etc/passwd", ""},
		{"ftp scheme", "ftp://example.com", ""},
		{"telegram deep link", "tg://resolve?domain=x", ""},
		{"plain label", "剩余流量", ""},
		{"quota figure", "4.7", ""},
		{"version-like", "1.2.3", ""},
		{"sentence with a url", "visit https://example.com now", ""},
		{"no tld", "localhost", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeOfficialURL(tt.in); got != tt.want {
				t.Errorf("NormalizeOfficialURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// The provider-supplied value lands in an href, so a scheme that can execute
// must never survive normalization in any casing or padding.
func TestNormalizeOfficialURLRejectsExecutableSchemes(t *testing.T) {
	hostile := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"  javascript:alert(1)  ",
		"jAvAsCrIpT:alert(document.domain)",
		"data:text/html,<script>alert(1)</script>",
		"vbscript:msgbox(1)",
	}
	for _, raw := range hostile {
		if got := NormalizeOfficialURL(raw); got != "" {
			t.Errorf("NormalizeOfficialURL(%q) = %q, want \"\"", raw, got)
		}
	}
}

func TestDetectOfficialURL(t *testing.T) {
	tests := []struct {
		name   string
		header string
		info   []SubInfo
		want   string
	}{
		{
			name:   "header wins",
			header: "https://www.paofu.cloud",
			info:   []SubInfo{{Key: "官网", Value: "https://other.example.com"}},
			want:   "https://www.paofu.cloud",
		},
		{
			name: "chinese label",
			info: []SubInfo{
				{Key: "剩余流量", Value: "4.7 TB"},
				{Key: "官网", Value: "https://example.com"},
			},
			want: "https://example.com",
		},
		{
			name: "english label, bare domain",
			info: []SubInfo{{Key: "Official Website", Value: "example.com"}},
			want: "https://example.com",
		},
		{
			// A support/chat entry is metadata and often holds a URL, but it is
			// not where "top up" should go.
			name: "support link ignored",
			info: []SubInfo{{Key: "客服", Value: "https://t.me/support"}},
			want: "",
		},
		{
			name:   "unusable header falls through to info",
			header: "javascript:alert(1)",
			info:   []SubInfo{{Key: "网址", Value: "https://example.com"}},
			want:   "https://example.com",
		},
		{
			name: "quota entry never mistaken for a site",
			info: []SubInfo{{Key: "剩余流量", Value: "4.7"}},
			want: "",
		},
		{"nothing", "", nil, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectOfficialURL(tt.header, tt.info); got != tt.want {
				t.Errorf("DetectOfficialURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOfficialURLToPersist(t *testing.T) {
	info := []SubInfo{{Key: "官网", Value: "https://from-info.example.com"}}

	tests := []struct {
		name    string
		current string
		header  string
		info    []SubInfo
		want    string
	}{
		{
			name:   "fills an empty field from the header",
			header: "https://example.com",
			want:   "https://example.com",
		},
		{
			name: "fills an empty field from an info entry",
			info: info,
			want: "https://from-info.example.com",
		},
		{
			// The operator corrected a moved domain; a mirror still reports the
			// old one. Their edit has to survive every future refresh.
			name:    "never overwrites an operator's own link",
			current: "https://operator-set.example.com",
			header:  "https://provider-says.example.com",
			info:    info,
			want:    "",
		},
		{
			name:    "whitespace-only counts as empty",
			current: "   ",
			header:  "https://example.com",
			want:    "https://example.com",
		},
		{
			name: "writes nothing when the feed says nothing",
			want: "",
		},
		{
			name:   "writes nothing when the feed's value is unusable",
			header: "javascript:alert(1)",
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := officialURLToPersist(tt.current, tt.header, tt.info); got != tt.want {
				t.Errorf("officialURLToPersist() = %q, want %q", got, tt.want)
			}
		})
	}
}
