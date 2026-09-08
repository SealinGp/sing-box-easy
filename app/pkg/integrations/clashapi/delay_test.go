package clashapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"
)

// TestDelayStatusMapping pins the classification the whole feature rests on:
// which status codes mean "this node is down" and which mean "we could not
// test it". Conflating them would blame a provider for an unapplied config.
func TestDelayStatusMapping(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    int
		wantErr error
	}{
		{name: "ok", status: http.StatusOK, body: `{"delay":312}`, want: 312},
		{name: "not found is untestable", status: http.StatusNotFound, body: `{}`, wantErr: ErrProxyNotFound},
		{name: "service unavailable is down", status: http.StatusServiceUnavailable, body: `{}`, wantErr: ErrDelayFailed},
		{name: "gateway timeout is down", status: http.StatusGatewayTimeout, body: `{}`, wantErr: ErrDelayFailed},
		{name: "unauthorized aborts", status: http.StatusUnauthorized, body: `{}`, wantErr: ErrUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			})

			got, err := client.Delay(context.Background(), "node", DefaultDelayURL, time.Second)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("delay = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestDelayEscapesTag is the difference between probing a node and 404ing on
// it. Real subscription tags carry spaces, emoji flags and the " | subID"
// ownership suffix, none of which survive an unescaped path segment.
func TestDelayEscapesTag(t *testing.T) {
	const tag = "🇭🇰 香港 01 a1b2c3d4 | sub_1778669339"

	var gotPath string
	var gotQuery url.Values
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path // already decoded by net/http
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(`{"delay":42}`))
	})

	if _, err := client.Delay(context.Background(), tag, DefaultDelayURL, 3*time.Second); err != nil {
		t.Fatalf("delay: %v", err)
	}

	if want := "/proxies/" + tag + "/delay"; gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if got := gotQuery.Get("url"); got != DefaultDelayURL {
		t.Errorf("url param = %q, want %q", got, DefaultDelayURL)
	}
	// sing-box parses `timeout` as milliseconds with ParseInt(..., 16), so it
	// must be an integer count of ms and must fit in an int16 — a value it
	// cannot parse makes the endpoint answer 400 for every node.
	if got := gotQuery.Get("timeout"); got != "3000" {
		t.Errorf("timeout param = %q, want %q", got, "3000")
	}
}

// TestDelaySendsBearer — an unauthenticated probe would report every node as
// untestable on any deployment that set a secret.
func TestDelaySendsBearer(t *testing.T) {
	var gotAuth string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = w.Write([]byte(`{"delay":1}`))
	})
	if _, err := client.Delay(context.Background(), "n", DefaultDelayURL, time.Second); err != nil {
		t.Fatalf("delay: %v", err)
	}
	if gotAuth != "Bearer s3cret" {
		t.Errorf("Authorization = %q", gotAuth)
	}
}
