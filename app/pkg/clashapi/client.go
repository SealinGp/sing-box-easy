// Package clashapi is the panel's client for sing-box's Clash-compatible API.
//
// It exists so the secret and the controller address are resolved in ONE place.
// The API is the only channel to a running sing-box's state — what it resolved,
// which connections are open, which rule each one matched — and two callers
// already need it (the DNS probe, the route probe) with a third (live traffic)
// on the way. Each growing its own copy of "turn `external_controller` into a
// URL, attach the bearer, classify the status code" is how one of them ends up
// rewriting `0.0.0.0` and another does not.
//
// Everything here is plain HTTP. sing-box serves its streaming endpoints over
// WebSocket for browsers, but every one of them also answers a plain GET —
// `/connections` returns a snapshot, `/traffic` writes chunked JSON — so the
// panel needs no WebSocket dependency and no upgrade handshake.
package clashapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sagernet/sing-box/option"
)

// defaultTimeout bounds one request. sing-box answers these from memory, so
// anything slower is an unreachable controller, not a slow one.
const defaultTimeout = 10 * time.Second

// ErrDisabled is returned when the config has no clash_api block, so there is
// no way to ask the running sing-box anything.
var ErrDisabled = errors.New("clash_api is not enabled in the sing-box config")

// ErrUnauthorized is returned when sing-box rejects the secret.
var ErrUnauthorized = errors.New("clash api rejected the request: check experimental.clash_api.secret")

// Client talks to one sing-box Clash API.
type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

// New builds a client from the sing-box config's experimental block. It returns
// ErrDisabled when no external controller is configured.
func New(experimental *option.ExperimentalOptions) (*Client, error) {
	if experimental == nil || experimental.ClashAPI == nil {
		return nil, ErrDisabled
	}
	controller := strings.TrimSpace(experimental.ClashAPI.ExternalController)
	if controller == "" {
		return nil, ErrDisabled
	}

	base, err := ControllerURL(controller)
	if err != nil {
		return nil, err
	}

	return &Client{
		baseURL: base,
		secret:  strings.TrimSpace(experimental.ClashAPI.Secret),
		http:    &http.Client{Timeout: defaultTimeout},
	}, nil
}

// BaseURL is the resolved controller URL, for messages that name it.
func (c *Client) BaseURL() string {
	return c.baseURL
}

// ControllerURL turns an `external_controller` value into a base URL.
//
// The value is a LISTEN address, not a URL: it may be "127.0.0.1:9090",
// ":9090" or "0.0.0.0:9090". A wildcard or empty host is a bind address, not a
// reachable one, so it is rewritten to loopback — the panel and sing-box run on
// the same host.
func ControllerURL(controller string) (string, error) {
	if strings.Contains(controller, "://") {
		parsed, err := url.Parse(controller)
		if err != nil {
			return "", fmt.Errorf("invalid external_controller %q: %w", controller, err)
		}
		return strings.TrimSuffix(parsed.String(), "/"), nil
	}

	host, port, err := net.SplitHostPort(controller)
	if err != nil {
		return "", fmt.Errorf("invalid external_controller %q: %w", controller, err)
	}
	switch host {
	case "", "0.0.0.0", "::", "[::]":
		host = "127.0.0.1"
	}

	return "http://" + net.JoinHostPort(host, port), nil
}

// Get performs one GET against `path` (which may carry a query string) and
// decodes the JSON body into `into`.
//
// Status handling is deliberately narrow: 200 decodes, 401 is ErrUnauthorized
// so callers can say "check the secret" instead of "status 401", and anything
// else is reported with its code. sing-box's error bodies are one-line JSON
// and add nothing a status code does not.
func (c *Client) Get(ctx context.Context, path string, into any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to build clash api request: %w", err)
	}
	if c.secret != "" {
		request.Header.Set("Authorization", "Bearer "+c.secret)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return fmt.Errorf("clash api unreachable at %s: %w", c.baseURL, err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return ErrUnauthorized
	default:
		return fmt.Errorf("clash api returned status %d", response.StatusCode)
	}

	if into == nil {
		return nil
	}
	if err := json.NewDecoder(response.Body).Decode(into); err != nil {
		return fmt.Errorf("failed to decode clash api response: %w", err)
	}
	return nil
}
