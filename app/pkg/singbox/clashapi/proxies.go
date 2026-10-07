package clashapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// ErrSelectRejected is returned when sing-box refuses a node switch: the group
// is not a selector, or the requested member is not one of its outbounds. Both
// are the caller's mistake, not a transport failure, so they must not be
// reported as "sing-box unreachable".
var ErrSelectRejected = errors.New("sing-box rejected the selection")

// globalGroup is the synthetic entry sing-box adds to `/proxies` "to fix clash
// dashboard" (experimental/clashapi/proxies.go). It is not an outbound in the
// config and selecting through it does nothing, so it is never reported.
const globalGroup = "GLOBAL"

// Proxy is one entry of `GET /proxies`: an outbound of the RUNNING sing-box.
//
// A group is told apart by the presence of `all`, not by its type string. The
// keying on the name of a type would silently render the next new group type
// as a plain node.
type Proxy struct {
	Name string
	// Type is sing-box's display name: "Selector", "URLTest", "VLESS", …
	Type string
	UDP  bool
	// Now is the member a group currently routes through. Empty for a node.
	Now string
	// All lists a group's members in config order. Nil for a node.
	All []string
	// Delay is the most recent URL-test result in milliseconds, 0 when the
	// outbound has never been tested or its last test failed (sing-box deletes
	// the history entry on failure).
	Delay    int
	TestedAt time.Time
}

// IsGroup reports whether the entry is a selector/urltest-style group.
func (p Proxy) IsGroup() bool {
	return p.All != nil
}

type proxyWire struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	UDP     bool   `json:"udp"`
	History []struct {
		Time  time.Time `json:"time"`
		Delay int       `json:"delay"`
	} `json:"history"`
	Now string   `json:"now"`
	All []string `json:"all"`
}

// Proxies lists the running outbounds IN CONFIG ORDER.
//
// sing-box writes the `proxies` object in outbound order, and that order is the
// only one an operator recognises — it is the order of their config. Decoding
// into a Go map would shuffle it on every call, so the object is walked token
// by token instead.
func (c *Client) Proxies(ctx context.Context) ([]Proxy, error) {
	var body struct {
		Proxies json.RawMessage `json:"proxies"`
	}
	if err := c.Get(ctx, "/proxies", &body); err != nil {
		return nil, err
	}
	return decodeProxies(body.Proxies)
}

func decodeProxies(raw json.RawMessage) ([]Proxy, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if _, err := decoder.Token(); err != nil {
		return nil, fmt.Errorf("failed to decode clash api proxies: %w", err)
	}

	var proxies []Proxy
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return nil, fmt.Errorf("failed to decode clash api proxies: %w", err)
		}
		var wire proxyWire
		if err := decoder.Decode(&wire); err != nil {
			return nil, fmt.Errorf("failed to decode clash api proxy %v: %w", key, err)
		}
		name, _ := key.(string)
		if name == globalGroup {
			continue
		}

		proxy := Proxy{Name: name, Type: wire.Type, UDP: wire.UDP, Now: wire.Now, All: wire.All}
		if last := len(wire.History) - 1; last >= 0 {
			proxy.Delay = wire.History[last].Delay
			proxy.TestedAt = wire.History[last].Time
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
}

// SelectProxy switches a selector group to one of its members.
//
// This changes the RUNNING process only. Whether it survives a restart is
// sing-box's `experimental.cache_file` at work, not anything written here —
// the config document is deliberately left alone.
//
//	204 → switched
//	400 → ErrSelectRejected (not a selector, or not a member)
//	404 → ErrProxyNotFound  (no such outbound in the running config)
func (c *Client) SelectProxy(ctx context.Context, group, name string) error {
	payload, err := json.Marshal(map[string]string{"name": name})
	if err != nil {
		return fmt.Errorf("failed to encode selection: %w", err)
	}
	response, err := c.send(ctx, http.MethodPut, "/proxies/"+url.PathEscape(group), payload)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusNoContent, http.StatusOK:
		return nil
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrSelectRejected, errorMessage(response.Body))
	case http.StatusNotFound:
		return ErrProxyNotFound
	case http.StatusUnauthorized:
		return ErrUnauthorized
	default:
		return fmt.Errorf("clash api returned status %d for selection", response.StatusCode)
	}
}

// GroupDelay URL-tests every member of a group and returns the latency of the
// ones that answered, keyed by tag. A member that failed is ABSENT from the
// map, not present with a zero.
//
// `timeout` bounds the whole group, not each member: sing-box derives one
// context from it and shares it across a batch of ten, so on a large group the
// members queued behind dead ones are never dialled and are simply absent.
// That is why the Proxies page tests members one by one instead (see
// proxies.Service.TestGroup); this stays for small groups and for callers that
// want sing-box's own urltest election to run.
func (c *Client) GroupDelay(ctx context.Context, group, testURL string, timeout time.Duration) (map[string]int, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	query := url.Values{}
	query.Set("url", testURL)
	query.Set("timeout", strconv.FormatInt(timeout.Milliseconds(), 10))

	response, err := c.send(ctx, http.MethodGet, "/group/"+url.PathEscape(group)+"/delay?"+query.Encode(), nil)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return nil, ErrProxyNotFound
	case http.StatusUnauthorized:
		return nil, ErrUnauthorized
	case http.StatusGatewayTimeout, http.StatusServiceUnavailable:
		return nil, fmt.Errorf("%w: %s", ErrDelayFailed, errorMessage(response.Body))
	default:
		return nil, fmt.Errorf("clash api returned status %d for group delay test", response.StatusCode)
	}

	delays := map[string]int{}
	if err := json.NewDecoder(response.Body).Decode(&delays); err != nil {
		return nil, fmt.Errorf("failed to decode group delay response: %w", err)
	}
	return delays, nil
}

// send performs one request and returns the raw response for the caller to
// classify. It exists for the calls whose status code IS the result; `Get`
// stays the path for plain reads.
func (c *Client) send(ctx context.Context, method, path string, body []byte) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("failed to build clash api request: %w", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if c.secret != "" {
		request.Header.Set("Authorization", "Bearer "+c.secret)
	}
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("clash api unreachable at %s: %w", c.baseURL, err)
	}
	return response, nil
}

// errorMessage reads sing-box's `{"message": "..."}` error body, falling back
// to a fixed phrase so a caller never formats an empty reason.
func errorMessage(body io.Reader) string {
	var payload struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(io.LimitReader(body, 4096)).Decode(&payload); err != nil || payload.Message == "" {
		return "no reason given"
	}
	return payload.Message
}
