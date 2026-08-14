package dnsprobe

import "testing"

// The mapping below was confirmed against a running sing-box 1.13.11: config
// rules 0/1/2 were logged as match[1]/match[3]/match[5]. sing-box computes the
// displayed value as `d += d + 1`, so a naive reading of the number as a rule
// index would point at the wrong rule.
func TestDecodeLoggedIndex(t *testing.T) {
	cases := []struct {
		logged int
		want   int
	}{
		{1, 0},
		{3, 1},
		{5, 2},
		{7, 3},
		{0, -1},  // even values are not produced by 2i+1
		{2, -1},  //
		{-1, -1}, // out of range
	}

	for _, c := range cases {
		if got := decodeLoggedIndex(c.logged); got != c.want {
			t.Errorf("decodeLoggedIndex(%d) = %d, want %d", c.logged, got, c.want)
		}
	}
}

// Real lines captured from sing-box 1.13.11 running with log.level=debug,
// including the ANSI colour codes it emits to a terminal.
func TestParseMatchLines(t *testing.T) {
	lines := []string{
		"+0800 2026-08-14 14:47:33 \x1b[37mDEBUG\x1b[0m dns: match[1] domain=a.example.com => predefined(NOERROR,a.example.com.\t300\tIN\tA\t1.2.3.4)",
		"+0800 2026-08-14 14:47:33 \x1b[37mDEBUG\x1b[0m dns: exchange a.example.com IN A",
		"+0800 2026-08-14 14:47:33 \x1b[37mDEBUG\x1b[0m dns: match[3] domain_suffix=b.example.com => route(s2)",
		// A routing (non-DNS) decision must not be picked up.
		"+0800 2026-08-14 14:47:33 \x1b[37mDEBUG\x1b[0m router: match[1] domain=x.com => route(direct)",
		"+0800 2026-08-14 14:47:33 \x1b[37mINFO\x1b[0m unrelated line",
	}

	got := ParseMatchLines(lines)

	if len(got) != 2 {
		t.Fatalf("got %d matches, want 2: %+v", len(got), got)
	}
	if got[0].LoggedIndex != 1 || got[0].ConfigIndex != 0 {
		t.Errorf("first: logged=%d config=%d, want 1/0", got[0].LoggedIndex, got[0].ConfigIndex)
	}
	if got[0].Description != "domain=a.example.com" {
		t.Errorf("first description = %q", got[0].Description)
	}
	if got[0].Action == "" {
		t.Error("first action is empty")
	}
	if got[1].LoggedIndex != 3 || got[1].ConfigIndex != 1 {
		t.Errorf("second: logged=%d config=%d, want 3/1", got[1].LoggedIndex, got[1].ConfigIndex)
	}
}

// sing-box omits the description for rules with no printable conditions.
func TestParseMatchLinesWithoutDescription(t *testing.T) {
	got := ParseMatchLines([]string{"DEBUG dns: match[5] => route(dns_router)"})

	if len(got) != 1 {
		t.Fatalf("got %d matches, want 1", len(got))
	}
	if got[0].ConfigIndex != 2 {
		t.Errorf("ConfigIndex = %d, want 2", got[0].ConfigIndex)
	}
	if got[0].Description != "" {
		t.Errorf("Description = %q, want empty", got[0].Description)
	}
	if got[0].Action != "route(dns_router)" {
		t.Errorf("Action = %q", got[0].Action)
	}
}

func TestVerifyMatches(t *testing.T) {
	attribution := Attribution{Rules: []RuleEvaluation{
		{Index: 0, Summary: "domain=a.example.com"},
		{Index: 1, Summary: "domain_suffix=b.example.com"},
	}}

	t.Run("agreeing decode is verified", func(t *testing.T) {
		got := VerifyMatches([]LoggedMatch{
			{LoggedIndex: 1, ConfigIndex: 0, Description: "domain=a.example.com"},
		}, attribution)
		if !got[0].Verified {
			t.Error("Verified = false, want true")
		}
	})

	t.Run("disagreeing decode stays unverified", func(t *testing.T) {
		got := VerifyMatches([]LoggedMatch{
			{LoggedIndex: 1, ConfigIndex: 0, Description: "rule_set=geosite-cn"},
		}, attribution)
		if got[0].Verified {
			t.Error("Verified = true, want false — description names a different condition")
		}
	})

	t.Run("out of range index is discarded", func(t *testing.T) {
		got := VerifyMatches([]LoggedMatch{
			{LoggedIndex: 99, ConfigIndex: 49, Description: "domain=x"},
		}, attribution)
		if got[0].ConfigIndex != -1 {
			t.Errorf("ConfigIndex = %d, want -1", got[0].ConfigIndex)
		}
	})
}

func TestHasDisagreement(t *testing.T) {
	cases := []struct {
		name    string
		results []ServerResult
		want    bool
	}{
		{
			name: "same records agree",
			results: []ServerResult{
				{Tag: "a", Records: []string{"A 1.1.1.1"}},
				{Tag: "b", Records: []string{"A 1.1.1.1"}},
			},
			want: false,
		},
		{
			name: "different records disagree",
			results: []ServerResult{
				{Tag: "a", Records: []string{"A 1.1.1.1"}},
				{Tag: "b", Records: []string{"A 9.9.9.9"}},
			},
			want: true,
		},
		{
			name: "errors and skips are not disagreement",
			results: []ServerResult{
				{Tag: "a", Records: []string{"A 1.1.1.1"}},
				{Tag: "b", Error: "timeout"},
				{Tag: "c", SkipReason: SkipReasonDetour},
				{Tag: "d", Records: []string{}},
			},
			want: false,
		},
		{
			name:    "single answer cannot disagree",
			results: []ServerResult{{Tag: "a", Records: []string{"A 1.1.1.1"}}},
			want:    false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := hasDisagreement(c.results); got != c.want {
				t.Errorf("hasDisagreement() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestControllerURL(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"127.0.0.1:9090", "http://127.0.0.1:9090", false},
		{"192.168.9.253:9095", "http://192.168.9.253:9095", false},
		// A wildcard bind address is not reachable; it must become loopback.
		{"0.0.0.0:9090", "http://127.0.0.1:9090", false},
		{":9090", "http://127.0.0.1:9090", false},
		{"::;bad", "", true},
	}

	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := controllerURL(c.in)
			if c.wantErr {
				if err == nil {
					t.Errorf("controllerURL(%q) = %q, want error", c.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

// The log window is established by diffing snapshots, because only the systemd
// backend honours the tailer's cursor. Without this, a probe that matched no
// rule inherited the previous probe's decision and displayed it as confirmed.
func TestNewLines(t *testing.T) {
	cases := []struct {
		name   string
		before []string
		after  []string
		want   []string
	}{
		{
			name:   "appended lines only",
			before: []string{"a", "b"},
			after:  []string{"a", "b", "c", "d"},
			want:   []string{"c", "d"},
		},
		{
			name:   "nothing appended",
			before: []string{"a", "b"},
			after:  []string{"a", "b"},
			want:   []string{},
		},
		{
			name:   "window scrolled past the anchor",
			before: []string{"a", "b"},
			after:  []string{"c", "d"},
			want:   []string{"c", "d"},
		},
		{
			name:   "empty snapshot means everything is new",
			before: []string{},
			after:  []string{"a", "b"},
			want:   []string{"a", "b"},
		},
		{
			name:   "repeated anchor uses the last occurrence",
			before: []string{"x", "dup"},
			after:  []string{"dup", "y", "dup", "z"},
			want:   []string{"z"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NewLines(c.before, c.after)
			if len(got) != len(c.want) {
				t.Fatalf("NewLines() = %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("NewLines()[%d] = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}
