package clashapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ErrProxyNotFound is returned when the running sing-box has no outbound with
// the requested tag. That is NOT a failed node: it means the config on disk and
// the config in memory disagree — usually because an edit has not been applied
// — so callers must report it as "could not test" rather than "down".
var ErrProxyNotFound = errors.New("outbound not found in the running sing-box")

// ErrDelayFailed is returned when the outbound exists but the URL test did not
// complete: a timeout, a refused connection, a TLS failure. This IS a node
// being unavailable.
var ErrDelayFailed = errors.New("url test failed")

// DefaultDelayURL is the target sing-box itself defaults to.
//
// HTTPS is not a preference. sing-box DISCARDS any `url` beginning with
// "http://" (experimental/clashapi/proxies.go: `if strings.HasPrefix(url,
// "http://") { url = "" }`) and silently substitutes this address — so a plain
// -HTTP target would produce numbers describing a different endpoint than the
// one the operator configured, with nothing anywhere saying so. Callers must
// reject http:// before it gets here; this constant documents what they would
// have silently received.
const DefaultDelayURL = "https://www.gstatic.com/generate_204"

// Delay runs one URL test through a single outbound and returns its latency.
//
// This is sing-box's own `GET /proxies/{name}/delay`, which dials the target
// through that exact outbound — same TLS settings, same transport, same
// multiplexing as real traffic — and times a HEAD request. Measuring from the
// panel instead would mean re-implementing every protocol to produce a number
// describing a connection nobody actually makes.
//
// The status codes are the result, not an error channel:
//
//	200 → the delay, in milliseconds
//	404 → ErrProxyNotFound (not in the running config; untestable)
//	503 → ErrDelayFailed  (sing-box's "an error occurred in the delay test")
//	504 → ErrDelayFailed  (the test outran `timeout`)
//	401 → ErrUnauthorized (the secret is wrong; every other call will fail too)
func (c *Client) Delay(ctx context.Context, tag, testURL string, timeout time.Duration) (int, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	query := url.Values{}
	query.Set("url", testURL)
	query.Set("timeout", strconv.FormatInt(timeout.Milliseconds(), 10))
	// The tag is user-controlled text that lands in a path segment — node tags
	// routinely carry spaces, emoji flags and "|". PathEscape, not QueryEscape:
	// the latter encodes a space as "+", which a path parser reads literally.
	path := "/proxies/" + url.PathEscape(tag) + "/delay?" + query.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to build delay request: %w", err)
	}
	if c.secret != "" {
		request.Header.Set("Authorization", "Bearer "+c.secret)
	}

	response, err := c.http.Do(request)
	if err != nil {
		// The controller itself is unreachable. Distinct from a node failing:
		// the caller must abandon the whole run rather than record every node
		// as down because the panel could not reach sing-box.
		return 0, fmt.Errorf("clash api unreachable at %s: %w", c.baseURL, err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return 0, ErrProxyNotFound
	case http.StatusUnauthorized:
		return 0, ErrUnauthorized
	case http.StatusServiceUnavailable:
		return 0, ErrDelayFailed
	case http.StatusGatewayTimeout:
		return 0, fmt.Errorf("%w: timed out after %s", ErrDelayFailed, timeout)
	default:
		return 0, fmt.Errorf("clash api returned status %d for delay test", response.StatusCode)
	}

	var body struct {
		Delay int `json:"delay"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("failed to decode delay response: %w", err)
	}
	return body.Delay, nil
}
