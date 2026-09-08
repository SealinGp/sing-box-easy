package service

import (
	"reflect"
	"testing"
)

func TestClampLines(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, defaultLogLines},
		{-5, defaultLogLines},
		{50, 50},
		{maxLogLines, maxLogLines},
		{maxLogLines + 1, maxLogLines},
	}
	for _, tc := range cases {
		if got := clampLines(tc.in); got != tc.want {
			t.Errorf("clampLines(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestSplitJournalOutput(t *testing.T) {
	out := "-- Logs begin at Sun 2026-05-31 --\n" +
		"2026-05-31T16:48:16+08:00 host sing-box[1607]: line one\n" +
		"2026-05-31T16:48:17+08:00 host sing-box[1607]: line two\n" +
		"-- cursor: s=abc123;i=1;b=2\n"

	lines, cursor := splitJournalOutput(out)
	if cursor != "s=abc123;i=1;b=2" {
		t.Fatalf("cursor = %q, want s=abc123;i=1;b=2", cursor)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 log lines, got %d: %v", len(lines), lines)
	}
	if lines[0] != "2026-05-31T16:48:16+08:00 host sing-box[1607]: line one" {
		t.Errorf("unexpected first line: %q", lines[0])
	}
}

func TestSplitJournalOutputNoEntries(t *testing.T) {
	lines, cursor := splitJournalOutput("-- No entries --\n")
	if len(lines) != 0 {
		t.Errorf("expected no lines, got %v", lines)
	}
	if cursor != "" {
		t.Errorf("expected empty cursor, got %q", cursor)
	}
}

func TestRingBuffer(t *testing.T) {
	r := newRingBuffer(3)
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		r.push(s)
	}
	got := r.slice()
	want := []string{"c", "d", "e"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ring buffer = %v, want %v", got, want)
	}

	// Fewer pushes than capacity preserves order without padding.
	r2 := newRingBuffer(5)
	r2.push("x")
	r2.push("y")
	if got := r2.slice(); !reflect.DeepEqual(got, []string{"x", "y"}) {
		t.Errorf("partial ring buffer = %v, want [x y]", got)
	}
}
