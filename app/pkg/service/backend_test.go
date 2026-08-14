package service

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseProcStatStartTicks(t *testing.T) {
	tests := []struct {
		name  string
		stat  string
		want  int64
		wantK bool
	}{
		{
			name: "typical stat line",
			stat: "1234 (sing-box) S 1 1234 1234 0 -1 4194560 1290 0 0 0 12 34 0 0 20 0 8 0 567890 1146880 512 18446744073709551615 1 1 0 0 0 0 0 0 0 0 0 0 17 3 0 0 0 0 0",
			want: 567890, wantK: true,
		},
		{
			name: "comm with spaces and parens",
			stat: "42 (my (weird) proc) R 1 42 42 0 -1 4194560 0 0 0 0 0 0 0 0 20 0 1 0 100 0 0 0 1 1 0 0 0 0 0 0 0 0 0 0 17 0 0 0 0 0 0",
			want: 100, wantK: true,
		},
		{name: "empty", stat: "", want: 0, wantK: false},
		{name: "no closing paren", stat: "1234 sing-box S 1", want: 0, wantK: false},
		{name: "too few fields", stat: "1234 (sing-box) S 1 2 3", want: 0, wantK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseProcStatStartTicks(tt.stat)
			if got != tt.want || ok != tt.wantK {
				t.Errorf("parseProcStatStartTicks() = (%d, %v), want (%d, %v)",
					got, ok, tt.want, tt.wantK)
			}
		})
	}
}

func TestParseBootTimeUnix(t *testing.T) {
	sysStat := "cpu  123 0 456 789 0 0 0 0 0 0\n" +
		"intr 12345\n" +
		"btime 1723600000\n" +
		"processes 999\n"

	got, ok := parseBootTimeUnix(sysStat)
	if !ok || got != 1723600000 {
		t.Errorf("parseBootTimeUnix() = (%d, %v), want (1723600000, true)", got, ok)
	}

	if _, ok := parseBootTimeUnix("cpu 1 2 3\n"); ok {
		t.Error("parseBootTimeUnix() without btime line should return ok=false")
	}
	if _, ok := parseBootTimeUnix("btime notanumber\n"); ok {
		t.Error("parseBootTimeUnix() with invalid btime should return ok=false")
	}
}

func TestFilterSyslogLines(t *testing.T) {
	var input string
	for i := 0; i < 5; i++ {
		input += fmt.Sprintf("Thu Aug 14 10:00:0%d 2026 daemon.info sing-box[123]: line %d\n", i, i)
	}
	input += "Thu Aug 14 10:00:06 2026 daemon.info dnsmasq[45]: unrelated\n"
	input += "\n  \n"

	got := filterSyslogLines(input, 3)
	if len(got) != 3 {
		t.Fatalf("filterSyslogLines() returned %d lines, want 3", len(got))
	}
	// Should keep the most recent sing-box lines (2, 3, 4) and drop dnsmasq.
	for i, want := range []string{"line 2", "line 3", "line 4"} {
		if !strings.Contains(got[i], want) {
			t.Errorf("line %d = %q, want it to contain %q", i, got[i], want)
		}
	}
}

func TestHasProcdInitScriptNonOpenwrt(t *testing.T) {
	// On non-OpenWrt system types the procd backend must never be selected,
	// regardless of what exists on disk.
	for _, st := range []SystemType{SystemDebian, SystemUnknown} {
		if hasProcdInitScript(st) {
			t.Errorf("hasProcdInitScript(%s) = true, want false", st)
		}
	}
}
