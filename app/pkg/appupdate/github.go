package appupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// Repo is the GitHub repository releases are pulled from.
	Repo = "SealinGp/sing-box-easy"

	// releasesAPI lists releases newest-first.
	releasesAPI = "https://api.github.com/repos/" + Repo + "/releases"

	// releaseDownloadBase is the prefix for release asset downloads.
	releaseDownloadBase = "https://github.com/" + Repo + "/releases/download"

	// releasesPageSize caps how many releases a single listing returns.
	releasesPageSize = 30

	// cacheTTL keeps GitHub's low unauthenticated rate limit at bay. Release
	// listings change rarely, so a few minutes of staleness is harmless.
	cacheTTL = 5 * time.Minute

	// errorCacheTTL debounces retries after a failed lookup without pinning
	// the failure for the full cacheTTL.
	errorCacheTTL = 30 * time.Second

	// apiTimeout bounds metadata requests (not downloads).
	apiTimeout = 20 * time.Second
)

// Release is the subset of the GitHub release payload the UI needs.
type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Prerelease  bool      `json:"prerelease"`
	Draft       bool      `json:"draft"`
	PublishedAt time.Time `json:"published_at"`
	HTMLURL     string    `json:"html_url"`
	Body        string    `json:"body"`
}

// AssetName is the release asset for the running platform. The release
// workflow publishes a single stable name per platform, so it can be derived
// from the tag alone.
func AssetName() string {
	return fmt.Sprintf("sing-box-easy-%s-%s.tar.gz", runtime.GOOS, runtime.GOARCH)
}

// AssetURL builds the download URL for a given release tag.
func AssetURL(tag string) string {
	return fmt.Sprintf("%s/%s/%s", releaseDownloadBase, tag, AssetName())
}

// ChecksumURL builds the sha256 sidecar URL for a given release tag.
func ChecksumURL(tag string) string {
	return AssetURL(tag) + ".sha256"
}

// IpkAssetName is the OpenWrt package asset for a release tag and opkg
// architecture, e.g. "sing-box-easy_1.2.4_x86_64.ipk".
//
// The arch is opkg's ("x86_64", "aarch64_generic", "arm_cortex-a7"), not Go's,
// and must come from the installed package's own Architecture field rather
// than runtime.GOARCH — one GOARCH maps to several opkg arches. The release
// workflow strips the leading "v" from the tag for this filename.
func IpkAssetName(tag, arch string) string {
	return fmt.Sprintf("sing-box-easy_%s_%s.ipk", strings.TrimPrefix(tag, "v"), arch)
}

// IpkAssetURL builds the ipk download URL for a release tag and opkg arch.
func IpkAssetURL(tag, arch string) string {
	return fmt.Sprintf("%s/%s/%s", releaseDownloadBase, tag, IpkAssetName(tag, arch))
}

// IpkChecksumURL builds the sha256 sidecar URL for the ipk asset.
func IpkChecksumURL(tag, arch string) string {
	return IpkAssetURL(tag, arch) + ".sha256"
}

// TokenFunc resolves the GitHub token to authenticate API calls with. It is a
// function rather than a plain string so a token saved in settings takes effect
// without rebuilding the client (which would drop the release cache). Returning
// an empty string means "call GitHub anonymously".
type TokenFunc func() string

// ReleaseClient fetches release metadata from GitHub with a short-lived cache.
type ReleaseClient struct {
	httpClient *http.Client

	// apiURL is the releases endpoint. It is a field rather than a constant
	// so tests can point it at a local server.
	apiURL string

	// token resolves the credential for each request; may be nil.
	token TokenFunc

	mu        sync.Mutex
	cached    []Release
	cachedAt  time.Time
	cachedErr error

	// etag is the last successful response's ETag. Replaying it as
	// If-None-Match lets GitHub answer 304, which does NOT count against the
	// REST rate limit — the single biggest win for anonymous callers.
	etag string

	// rate mirrors the last seen X-RateLimit-* headers, for diagnostics.
	rate RateLimit
}

// RateLimit is the rate-limit state reported by GitHub's response headers.
type RateLimit struct {
	Limit         int       `json:"limit"`
	Remaining     int       `json:"remaining"`
	ResetAt       time.Time `json:"reset_at"`
	Authenticated bool      `json:"authenticated"`
}

// NewReleaseClient creates a release client. proxy, when non-empty, must be a
// valid proxy URL (e.g. "http://127.0.0.1:7890") and is used for GitHub calls.
// token may be nil, in which case GitHub is called anonymously (60 req/h/IP).
func NewReleaseClient(proxy string, token TokenFunc) (*ReleaseClient, error) {
	client := &http.Client{Timeout: apiTimeout}

	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", proxy, err)
		}
		client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	return &ReleaseClient{httpClient: client, apiURL: releasesAPI, token: token}, nil
}

// RateLimitState returns the last observed rate-limit headers.
func (c *ReleaseClient) RateLimitState() RateLimit {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rate
}

// currentToken resolves and trims the configured token.
func (c *ReleaseClient) currentToken() string {
	if c.token == nil {
		return ""
	}
	return strings.TrimSpace(c.token())
}

// ListReleases returns published releases newest-first, excluding drafts.
// Results are cached for cacheTTL; pass force to bypass the cache.
func (c *ReleaseClient) ListReleases(force bool) ([]Release, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// A failed lookup is cached far more briefly than a successful one: a
	// transient GitHub outage must not pin an error for the full TTL.
	ttl := cacheTTL
	if c.cachedErr != nil {
		ttl = errorCacheTTL
	}

	if !force && !c.cachedAt.IsZero() && time.Since(c.cachedAt) < ttl {
		if c.cachedErr != nil {
			return nil, c.cachedErr
		}
		// Return a copy so callers cannot mutate the cache.
		return append([]Release(nil), c.cached...), nil
	}

	releases, err := c.fetchReleases()
	c.cachedAt = time.Now()

	// 304: GitHub confirmed the cached listing is still current (and charged
	// nothing against the rate limit). Keep the body, refresh the timestamp.
	if errors.Is(err, errNotModified) {
		c.cachedErr = nil
		return append([]Release(nil), c.cached...), nil
	}

	c.cached, c.cachedErr = releases, err

	if err != nil {
		return nil, err
	}
	return append([]Release(nil), releases...), nil
}

// LatestRelease returns the newest non-draft, non-prerelease release. If the
// repository only has prereleases, the newest prerelease is returned instead.
func (c *ReleaseClient) LatestRelease(force bool) (*Release, error) {
	releases, err := c.ListReleases(force)
	if err != nil {
		return nil, err
	}
	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases published for %s", Repo)
	}

	for i := range releases {
		if !releases[i].Prerelease {
			return &releases[i], nil
		}
	}
	return &releases[0], nil
}

// FindRelease returns the release matching tag, or an error when absent.
func (c *ReleaseClient) FindRelease(tag string) (*Release, error) {
	releases, err := c.ListReleases(false)
	if err != nil {
		return nil, err
	}
	for i := range releases {
		if releases[i].TagName == tag {
			return &releases[i], nil
		}
	}
	return nil, fmt.Errorf("release %q not found in the latest %d releases of %s", tag, releasesPageSize, Repo)
}

// errNotModified signals a 304: the cached listing is still current.
var errNotModified = errors.New("not modified")

func (c *ReleaseClient) fetchReleases() ([]Release, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s?per_page=%d", c.apiURL, releasesPageSize), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build releases request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "sing-box-easy")

	token := c.currentToken()
	if token != "" {
		// Bearer works for both classic PATs and fine-grained tokens.
		req.Header.Set("Authorization", "Bearer "+token)
	}
	// Only replay the ETag when a previous body is still cached to fall back on.
	if c.etag != "" && c.cached != nil {
		req.Header.Set("If-None-Match", c.etag)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	c.rate = parseRateLimit(resp.Header, token != "")

	if resp.StatusCode == http.StatusNotModified {
		return nil, errNotModified
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("GitHub rejected the stored access token (HTTP 401); sign in again under Settings")
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
		return nil, c.rateLimitError(resp.StatusCode, token != "")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	// Cap the body so a malformed/huge response cannot exhaust memory.
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read GitHub response: %w", err)
	}

	var all []Release
	if err := json.Unmarshal(body, &all); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub response: %w", err)
	}

	published := make([]Release, 0, len(all))
	for _, r := range all {
		if r.Draft || r.TagName == "" {
			continue
		}
		published = append(published, r)
	}

	// Record the ETag only alongside a body we can serve on a later 304.
	c.etag = resp.Header.Get("ETag")

	return published, nil
}

// rateLimitError builds an actionable message: anonymous callers get 60
// requests/hour per IP, so the fix is almost always "add a token".
func (c *ReleaseClient) rateLimitError(status int, authenticated bool) error {
	wait := ""
	if !c.rate.ResetAt.IsZero() {
		if s := formatResetWait(time.Until(c.rate.ResetAt)); s != "" {
			wait = " Resets in " + s + "."
		}
	}
	if authenticated {
		return fmt.Errorf("GitHub API rate limit reached (HTTP %d) for the signed-in GitHub account.%s", status, wait)
	}
	return fmt.Errorf("GitHub API rate limit reached (HTTP %d). Anonymous requests are capped at 60/hour per IP — sign in with GitHub under Settings to raise it to 5000/hour.%s", status, wait)
}

// formatResetWait renders a wait as human prose. Go's default Duration string
// would give "4m0s"/"1h0m0s", which reads like a stopwatch, not an ETA.
// Returns "" when the deadline has already passed.
func formatResetWait(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	if d < time.Minute {
		return "less than a minute"
	}

	mins := int(d.Round(time.Minute) / time.Minute)
	if mins < 60 {
		return fmt.Sprintf("%d %s", mins, plural(mins, "minute"))
	}

	hours, rem := mins/60, mins%60
	if rem == 0 {
		return fmt.Sprintf("%d %s", hours, plural(hours, "hour"))
	}
	return fmt.Sprintf("%d %s %d %s", hours, plural(hours, "hour"), rem, plural(rem, "minute"))
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

// parseRateLimit reads the X-RateLimit-* headers; missing headers yield zeros.
func parseRateLimit(h http.Header, authenticated bool) RateLimit {
	rl := RateLimit{Authenticated: authenticated}
	if v, err := strconv.Atoi(h.Get("X-RateLimit-Limit")); err == nil {
		rl.Limit = v
	}
	if v, err := strconv.Atoi(h.Get("X-RateLimit-Remaining")); err == nil {
		rl.Remaining = v
	}
	if v, err := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64); err == nil && v > 0 {
		rl.ResetAt = time.Unix(v, 0)
	}
	return rl
}
