package v1_13_0

import (
	"strings"
	"testing"
)

func TestFormatSSEFrameShape(t *testing.T) {
	got := formatSSEFrame("lines", []byte(`{"code":0}`))
	want := "event: lines\ndata: {\"code\":0}\n\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// The regression this framing exists to prevent.
//
// sing-box logs multi-line messages (a config decode error prints the offending
// fragment), and the line reaches us inside a JSON string. A single `data:` line
// carrying a raw newline ends the frame at the newline, so the client parses
// half an object, fails, and drops the event — a fatal message vanishing exactly
// when it matters most.
func TestFormatSSEFrameSplitsEmbeddedNewlines(t *testing.T) {
	got := formatSSEFrame("lines", []byte("line one\nline two\nline three"))

	want := "event: lines\ndata: line one\ndata: line two\ndata: line three\n\n"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}

	// Every payload line must be prefixed; none may ride bare.
	for _, line := range strings.Split(strings.TrimSuffix(got, "\n\n"), "\n")[1:] {
		if !strings.HasPrefix(line, "data: ") {
			t.Fatalf("unprefixed line %q would terminate the frame early", line)
		}
	}
}

// A frame always ends with a blank line; without it the client's parser waits
// for a boundary that never comes and the event is never dispatched.
func TestFormatSSEFrameTerminates(t *testing.T) {
	for _, payload := range []string{"", "{}", "a\nb"} {
		got := formatSSEFrame("done", []byte(payload))
		if !strings.HasSuffix(got, "\n\n") {
			t.Fatalf("payload %q produced an unterminated frame: %q", payload, got)
		}
	}
}
