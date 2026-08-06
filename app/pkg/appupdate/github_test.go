package appupdate

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const releasesFixture = `[
  {"tag_name":"v1.3.0","name":"1.3.0","draft":false,"prerelease":false,
   "published_at":"2026-01-05T10:00:00Z","html_url":"https://example.test/v1.3.0","body":"stable"},
  {"tag_name":"v1.4.0-rc.1","name":"1.4.0 RC1","draft":false,"prerelease":true,
   "published_at":"2026-02-01T10:00:00Z","html_url":"https://example.test/v1.4.0-rc.1","body":"rc"},
  {"tag_name":"v9.9.9","name":"draft","draft":true,"prerelease":false,
   "published_at":"2026-03-01T10:00:00Z","html_url":"https://example.test/draft","body":"draft"}
]`

// newTestClient points a ReleaseClient at a local server and reports how many
// upstream requests it actually made.
func newTestClient(t *testing.T, handler http.HandlerFunc) (*ReleaseClient, *int32) {
	t.Helper()

	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)

	client, err := NewReleaseClient("", nil)
	if err != nil {
		t.Fatalf("NewReleaseClient: %v", err)
	}
	client.apiURL = srv.URL

	return client, &calls
}

func okFixture(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(releasesFixture))
}

func TestListReleasesSkipsDrafts(t *testing.T) {
	client, _ := newTestClient(t, okFixture)

	releases, err := client.ListReleases(false)
	if err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	if len(releases) != 2 {
		t.Fatalf("got %d releases, want 2 (the draft must be filtered out)", len(releases))
	}
	for _, r := range releases {
		if r.Draft {
			t.Errorf("draft release %q leaked into the list", r.TagName)
		}
	}
}

// The cache is the whole reason GitHub's unauthenticated rate limit is
// survivable, so a repeat call must not hit the network again.
func TestListReleasesCachesBetweenCalls(t *testing.T) {
	client, calls := newTestClient(t, okFixture)

	for i := 0; i < 3; i++ {
		if _, err := client.ListReleases(false); err != nil {
			t.Fatalf("call %d: %v", i, err)
		}
	}

	if got := atomic.LoadInt32(calls); got != 1 {
		t.Errorf("upstream was called %d times, want 1 (cache should absorb the rest)", got)
	}
}

func TestListReleasesForceBypassesCache(t *testing.T) {
	client, calls := newTestClient(t, okFixture)

	if _, err := client.ListReleases(false); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := client.ListReleases(true); err != nil {
		t.Fatalf("forced call: %v", err)
	}

	if got := atomic.LoadInt32(calls); got != 2 {
		t.Errorf("upstream was called %d times, want 2 (force must refetch)", got)
	}
}

func TestListReleasesDoesNotShareMutableState(t *testing.T) {
	client, _ := newTestClient(t, okFixture)

	first, err := client.ListReleases(false)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	first[0].TagName = "mutated"

	second, err := client.ListReleases(false)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if second[0].TagName == "mutated" {
		t.Error("mutating a returned slice corrupted the cache; want a defensive copy")
	}
}

func TestLatestReleasePrefersStableOverPrerelease(t *testing.T) {
	client, _ := newTestClient(t, okFixture)

	latest, err := client.LatestRelease(false)
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	// v1.4.0-rc.1 is newer by date but is a prerelease.
	if latest.TagName != "v1.3.0" {
		t.Errorf("LatestRelease = %q, want v1.3.0 (the newest stable)", latest.TagName)
	}
}

func TestFindRelease(t *testing.T) {
	client, _ := newTestClient(t, okFixture)

	found, err := client.FindRelease("v1.4.0-rc.1")
	if err != nil {
		t.Fatalf("FindRelease: %v", err)
	}
	if found.TagName != "v1.4.0-rc.1" {
		t.Errorf("FindRelease returned %q", found.TagName)
	}

	if _, err := client.FindRelease("v0.0.1"); err == nil {
		t.Error("FindRelease on a missing tag succeeded, want an error")
	}
}

func TestListReleasesSurfacesRateLimit(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})

	_, err := client.ListReleases(false)
	if err == nil {
		t.Fatal("ListReleases succeeded on HTTP 403, want an error")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error = %v, want it to mention the rate limit", err)
	}
}

func TestNewReleaseClientRejectsBadProxy(t *testing.T) {
	if _, err := NewReleaseClient("://not-a-url", nil); err == nil {
		t.Error("NewReleaseClient accepted a malformed proxy URL, want an error")
	}
}

func TestListReleasesSendsAuthorizationWhenTokenSet(t *testing.T) {
	var gotAuth string
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		okFixture(w, r)
	})
	client.token = func() string { return "  ghp_secret  " }

	if _, err := client.ListReleases(false); err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	if gotAuth != "Bearer ghp_secret" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer ghp_secret")
	}
}

func TestListReleasesOmitsAuthorizationWhenTokenEmpty(t *testing.T) {
	var hasAuth bool
	client, _ := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, hasAuth = r.Header["Authorization"]
		okFixture(w, r)
	})
	client.token = func() string { return "   " }

	if _, err := client.ListReleases(false); err != nil {
		t.Fatalf("ListReleases: %v", err)
	}
	if hasAuth {
		t.Error("Authorization header sent with an empty token, want none")
	}
}

// A 304 must serve the cached body rather than surfacing an error — that is
// the whole point of the conditional request (it costs no rate-limit quota).
func TestListReleasesServesCacheOn304(t *testing.T) {
	const etag = `W/"abc123"`
	var sawIfNoneMatch string

	client, calls := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if inm := r.Header.Get("If-None-Match"); inm != "" {
			sawIfNoneMatch = inm
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", etag)
		okFixture(w, r)
	})

	first, err := client.ListReleases(true)
	if err != nil {
		t.Fatalf("first ListReleases: %v", err)
	}

	second, err := client.ListReleases(true) // force: must hit the network
	if err != nil {
		t.Fatalf("second ListReleases after 304: %v", err)
	}

	if sawIfNoneMatch != etag {
		t.Errorf("If-None-Match = %q, want %q", sawIfNoneMatch, etag)
	}
	if got := atomic.LoadInt32(calls); got != 2 {
		t.Errorf("upstream calls = %d, want 2", got)
	}
	if len(second) != len(first) || len(second) == 0 {
		t.Errorf("304 returned %d releases, want the %d cached ones", len(second), len(first))
	}
}

func TestListReleasesRejectsBadToken(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	client.token = func() string { return "ghp_bad" }

	_, err := client.ListReleases(false)
	if err == nil {
		t.Fatal("ListReleases succeeded on HTTP 401, want an error")
	}
	if !strings.Contains(err.Error(), "sign in") {
		t.Errorf("error = %v, want it to tell the user to sign in again", err)
	}
}

// An anonymous 403 must tell the user how to fix it, not just say "try later".
func TestRateLimitErrorSuggestsTokenWhenAnonymous(t *testing.T) {
	client, _ := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	})

	_, err := client.ListReleases(false)
	if err == nil {
		t.Fatal("ListReleases succeeded on HTTP 403, want an error")
	}
	if !strings.Contains(err.Error(), "sign in") {
		t.Errorf("anonymous rate-limit error = %v, want it to suggest signing in", err)
	}
	if rl := client.RateLimitState(); rl.Limit != 60 || rl.Remaining != 0 || rl.Authenticated {
		t.Errorf("RateLimitState = %+v, want limit=60 remaining=0 authenticated=false", rl)
	}
}

func TestFormatResetWait(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"already passed", -1 * time.Minute, ""},
		{"zero", 0, ""},
		{"seconds", 30 * time.Second, "less than a minute"},
		{"one minute", time.Minute, "1 minute"},
		{"minutes", 4 * time.Minute, "4 minutes"},
		{"rounds to minutes", 4*time.Minute + 40*time.Second, "5 minutes"},
		{"exactly an hour", time.Hour, "1 hour"},
		{"hour and minutes", 90 * time.Minute, "1 hour 30 minutes"},
		{"hours", 2 * time.Hour, "2 hours"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatResetWait(tt.d); got != tt.want {
				t.Errorf("formatResetWait(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}
