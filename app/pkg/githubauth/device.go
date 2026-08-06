// Package githubauth implements GitHub sign-in via the OAuth Device
// Authorization Grant (RFC 8628), the flow the `gh` CLI uses.
//
// Device flow is the only OAuth variant that fits a self-hosted app:
//   - It needs no client secret, so nothing confidential ships in the binary.
//   - It needs no redirect/callback URL, so it works when the dashboard is
//     reached at an arbitrary LAN IP, over plain HTTP, or behind NAT — none of
//     which a registered callback URL could cover.
//
// The resulting user token raises the GitHub API rate limit from 60 requests
// per hour per IP to 5000 per hour. No scopes are requested: reading public
// release metadata requires none.
package githubauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GitHub device-flow endpoints.
const (
	deviceCodeURL  = "https://github.com/login/device/code"
	accessTokenURL = "https://github.com/login/oauth/access_token"
	userAPIURL     = "https://api.github.com/user"
)

// requestTimeout bounds a single call to GitHub.
const requestTimeout = 20 * time.Second

// maxBodyBytes caps a response body so a malformed reply cannot exhaust memory.
const maxBodyBytes = 1 << 20

// Poll error codes returned by GitHub's token endpoint (RFC 8628 §3.5).
const (
	// errAuthorizationPending means the user has not finished approving yet.
	// It is the expected steady state while waiting, not a failure.
	errAuthorizationPending = "authorization_pending"

	// errSlowDown means we polled too fast; GitHub requires adding 5 seconds
	// to the interval each time this is returned.
	errSlowDown = "slow_down"

	errExpiredToken       = "expired_token"
	errAccessDenied       = "access_denied"
	errDeviceFlowDisabled = "device_flow_disabled"
	errBadClientID        = "incorrect_client_credentials"
)

// slowDownPenalty is the interval increase GitHub mandates on a slow_down.
const slowDownPenalty = 5 * time.Second

// DeviceCode is GitHub's response to a device authorization request.
type DeviceCode struct {
	// DeviceCode is the secret this server polls with. It must never be
	// shown to the user or returned to the browser.
	DeviceCode string `json:"device_code"`

	// UserCode is the short code the user types into GitHub.
	UserCode string `json:"user_code"`

	// VerificationURI is where the user enters UserCode.
	VerificationURI string `json:"verification_uri"`

	// VerificationURIComplete, when present, embeds the code so the user can
	// approve with one click. GitHub does not currently send it, but it is
	// part of RFC 8628 and costs nothing to support.
	VerificationURIComplete string `json:"verification_uri_complete"`

	ExpiresIn int `json:"expires_in"`
	Interval  int `json:"interval"`
}

// Expiry converts ExpiresIn to an absolute deadline from now.
func (d DeviceCode) Expiry(now time.Time) time.Time {
	if d.ExpiresIn <= 0 {
		// GitHub documents 900s; assume that rather than expiring instantly.
		return now.Add(15 * time.Minute)
	}
	return now.Add(time.Duration(d.ExpiresIn) * time.Second)
}

// PollInterval returns the minimum delay between token polls.
func (d DeviceCode) PollInterval() time.Duration {
	if d.Interval <= 0 {
		return 5 * time.Second
	}
	return time.Duration(d.Interval) * time.Second
}

// Client talks to GitHub's device-flow endpoints.
type Client struct {
	httpClient *http.Client
	clientID   string

	// Endpoints are fields rather than constants so tests can redirect them.
	deviceCodeURL  string
	accessTokenURL string
	userAPIURL     string
}

// NewClient builds a device-flow client for the given OAuth App client ID.
// proxy may be empty. The client ID is public by design — device flow has no
// confidential credential to protect.
func NewClient(clientID, proxy string) (*Client, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, ErrNotConfigured
	}

	httpClient := &http.Client{Timeout: requestTimeout}
	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", proxy, err)
		}
		httpClient.Transport = &http.Transport{Proxy: http.ProxyURL(proxyURL)}
	}

	return &Client{
		httpClient:     httpClient,
		clientID:       clientID,
		deviceCodeURL:  deviceCodeURL,
		accessTokenURL: accessTokenURL,
		userAPIURL:     userAPIURL,
	}, nil
}

// RequestDeviceCode starts a device authorization. No scope is requested:
// public release metadata needs none, and a zero-scope token is the least
// dangerous thing to hold on a user's behalf.
func (c *Client) RequestDeviceCode(ctx context.Context) (*DeviceCode, error) {
	form := url.Values{"client_id": {c.clientID}}

	var out struct {
		DeviceCode
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	status, err := c.postForm(ctx, c.deviceCodeURL, form, &out)
	if err != nil {
		// GitHub answers a bare 404 (no JSON body) when the client ID does not
		// resolve to an app, which is the likeliest misconfiguration here.
		if status == http.StatusNotFound {
			return nil, describeAuthError(errBadClientID, "")
		}
		return nil, err
	}

	if out.Error != "" {
		return nil, describeAuthError(out.Error, out.ErrorDescription)
	}
	if out.DeviceCode.DeviceCode == "" || out.UserCode == "" {
		return nil, fmt.Errorf("GitHub returned an incomplete device authorization response")
	}
	if out.VerificationURI == "" {
		out.VerificationURI = "https://github.com/login/device"
	}

	return &out.DeviceCode, nil
}

// PollResult is the outcome of one token-endpoint poll.
type PollResult struct {
	// Token is set only when Status is StatusAuthorized.
	Token string

	Status PollStatus

	// SlowDown asks the caller to widen its polling interval by
	// slowDownPenalty before the next attempt.
	SlowDown bool

	// Err carries the terminal failure when Status is StatusFailed.
	Err error
}

// PollStatus classifies a poll outcome.
type PollStatus string

const (
	StatusPending    PollStatus = "pending"
	StatusAuthorized PollStatus = "authorized"
	StatusDenied     PollStatus = "denied"
	StatusExpired    PollStatus = "expired"
	StatusFailed     PollStatus = "failed"
)

// SlowDownPenalty is the amount to add to the poll interval on a slow_down.
func SlowDownPenalty() time.Duration { return slowDownPenalty }

// PollToken performs exactly one exchange attempt. Callers drive the retry
// loop so they control cancellation and the interval.
func (c *Client) PollToken(ctx context.Context, deviceCode string) PollResult {
	form := url.Values{
		"client_id":   {c.clientID},
		"device_code": {deviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}

	var out struct {
		AccessToken      string `json:"access_token"`
		TokenType        string `json:"token_type"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if _, err := c.postForm(ctx, c.accessTokenURL, form, &out); err != nil {
		// A transport hiccup is not terminal — the user may still be
		// approving, so stay pending and let the caller retry.
		return PollResult{Status: StatusPending}
	}

	switch out.Error {
	case "":
		if out.AccessToken == "" {
			return PollResult{Status: StatusFailed, Err: fmt.Errorf("GitHub returned an empty access token")}
		}
		return PollResult{Status: StatusAuthorized, Token: out.AccessToken}
	case errAuthorizationPending:
		return PollResult{Status: StatusPending}
	case errSlowDown:
		return PollResult{Status: StatusPending, SlowDown: true}
	case errAccessDenied:
		return PollResult{Status: StatusDenied, Err: fmt.Errorf("authorization was denied on GitHub")}
	case errExpiredToken:
		return PollResult{Status: StatusExpired, Err: fmt.Errorf("the login code expired before it was approved")}
	default:
		return PollResult{Status: StatusFailed, Err: describeAuthError(out.Error, out.ErrorDescription)}
	}
}

// AccountLogin returns the username the token belongs to, for display. A
// failure here is not fatal to sign-in, so callers may ignore the error.
func (c *Client) AccountLogin(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.userAPIURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to build the user request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "sing-box-easy")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub returned HTTP %d when reading the account", resp.StatusCode)
	}

	var out struct {
		Login string `json:"login"`
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return "", fmt.Errorf("failed to read the GitHub account response: %w", err)
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("failed to parse the GitHub account response: %w", err)
	}
	return out.Login, nil
}

// postForm submits a form-encoded body and decodes a JSON reply. It returns
// the HTTP status alongside the error so callers can map transport-level
// statuses (GitHub does not always use a JSON error body) to better messages.
func (c *Client) postForm(ctx context.Context, endpoint string, form url.Values, out any) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return 0, fmt.Errorf("failed to build the GitHub request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Without this GitHub replies url-encoded rather than as JSON.
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sing-box-easy")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to reach GitHub: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return resp.StatusCode, fmt.Errorf("failed to read the GitHub response: %w", err)
	}

	// GitHub reports device-flow errors in the JSON body with a 200, so a
	// non-2xx status is an infrastructure problem worth naming separately.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("GitHub returned HTTP %d", resp.StatusCode)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return resp.StatusCode, fmt.Errorf("failed to parse the GitHub response: %w", err)
	}
	return resp.StatusCode, nil
}

// describeAuthError turns a GitHub error code into an actionable message.
// The generic descriptions GitHub returns ("the device flow has not been
// enabled") do not tell an operator which checkbox to tick.
func describeAuthError(code, description string) error {
	switch code {
	case errDeviceFlowDisabled:
		return fmt.Errorf("device flow is not enabled for this GitHub OAuth App; " +
			"enable it under Settings -> Developer settings -> OAuth Apps -> your app -> \"Enable Device Flow\"")
	case errBadClientID:
		return fmt.Errorf("GitHub rejected the configured OAuth client ID; " +
			"check github.oauth_client_id in app.yml")
	}
	if description != "" {
		return fmt.Errorf("GitHub returned %s: %s", code, description)
	}
	return fmt.Errorf("GitHub returned %s", code)
}
