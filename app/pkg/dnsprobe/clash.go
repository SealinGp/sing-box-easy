package dnsprobe

import (
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

// clashTimeout bounds a single /dns/query call. sing-box's own DNS timeout is
// shorter, so this only catches an unreachable controller.
const clashTimeout = 10 * time.Second

// ErrClashAPIDisabled is returned when the running config has no clash_api
// block, so there is no way to ask sing-box what it resolved.
var ErrClashAPIDisabled = errors.New("clash_api is not enabled in the sing-box config")

// Answer is one resource record from a DNS response.
type Answer struct {
	Name string `json:"name"`
	Type string `json:"type"`
	TTL  uint32 `json:"ttl"`
	Data string `json:"data"`
}

// LiveResult is what sing-box itself returned for a query. This is ground
// truth: the request goes through sing-box's DNS router, so hosts entries,
// predefined answers, rule routing and FakeIP are all applied.
type LiveResult struct {
	Status  int      `json:"status"`
	Answers []Answer `json:"answers"`
	// ElapsedMS is the round trip as measured by this process.
	ElapsedMS int64 `json:"elapsed_ms"`
}

// clashResponse mirrors the subset of sing-box's /dns/query payload we use.
// Field names follow the DoH JSON convention sing-box emits (capitalised).
type clashResponse struct {
	Status int `json:"Status"`
	Answer []struct {
		Name string `json:"name"`
		Type int    `json:"type"`
		TTL  uint32 `json:"TTL"`
		Data string `json:"data"`
	} `json:"Answer"`
}

// ClashClient queries sing-box's Clash API.
type ClashClient struct {
	baseURL string
	secret  string
	http    *http.Client
}

// NewClashClient builds a client from the running sing-box config. It returns
// ErrClashAPIDisabled when the config has no external controller.
func NewClashClient(experimental *option.ExperimentalOptions) (*ClashClient, error) {
	if experimental == nil || experimental.ClashAPI == nil {
		return nil, ErrClashAPIDisabled
	}
	controller := strings.TrimSpace(experimental.ClashAPI.ExternalController)
	if controller == "" {
		return nil, ErrClashAPIDisabled
	}

	host, err := controllerURL(controller)
	if err != nil {
		return nil, err
	}

	return &ClashClient{
		baseURL: host,
		secret:  strings.TrimSpace(experimental.ClashAPI.Secret),
		http:    &http.Client{Timeout: clashTimeout},
	}, nil
}

// controllerURL turns an external_controller value into a base URL.
//
// The value is a listen address, not a URL: it may be "127.0.0.1:9090",
// ":9090" or "0.0.0.0:9090". A wildcard or empty host is a bind address, not a
// reachable one, so it is rewritten to loopback — the panel and sing-box run on
// the same host.
func controllerURL(controller string) (string, error) {
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

// Query asks sing-box to resolve name/qType through its own DNS router.
func (c *ClashClient) Query(name, qType string) (*LiveResult, error) {
	endpoint := fmt.Sprintf("%s/dns/query?name=%s&type=%s",
		c.baseURL, url.QueryEscape(name), url.QueryEscape(qType))

	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build clash api request: %w", err)
	}
	if c.secret != "" {
		request.Header.Set("Authorization", "Bearer "+c.secret)
	}

	started := time.Now()
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("clash api unreachable at %s: %w", c.baseURL, err)
	}
	defer response.Body.Close()
	elapsed := time.Since(started)

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized:
		return nil, errors.New("clash api rejected the request: check experimental.clash_api.secret")
	default:
		return nil, fmt.Errorf("clash api returned status %d", response.StatusCode)
	}

	var decoded clashResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("failed to decode clash api response: %w", err)
	}

	result := &LiveResult{
		Status:    decoded.Status,
		Answers:   make([]Answer, 0, len(decoded.Answer)),
		ElapsedMS: elapsed.Milliseconds(),
	}
	for _, answer := range decoded.Answer {
		result.Answers = append(result.Answers, Answer{
			Name: strings.TrimSuffix(answer.Name, "."),
			Type: recordTypeName(answer.Type),
			TTL:  answer.TTL,
			Data: answer.Data,
		})
	}

	return result, nil
}

// Mode returns the running instance's Clash mode (rule / global / direct).
//
// Configs commonly carry `clash_mode` escape-hatch rules — "when I flip the
// dashboard to global, send everything through the proxy" — and those rules sit
// near the top, ahead of everything else. Without the current mode they are
// undecidable, which makes every prediction below them a guess. It is one HTTP
// call to turn the most common source of uncertainty into a fact.
func (c *ClashClient) Mode() (string, error) {
	request, err := http.NewRequest(http.MethodGet, c.baseURL+"/configs", nil)
	if err != nil {
		return "", fmt.Errorf("failed to build clash api request: %w", err)
	}
	if c.secret != "" {
		request.Header.Set("Authorization", "Bearer "+c.secret)
	}

	response, err := c.http.Do(request)
	if err != nil {
		return "", fmt.Errorf("clash api unreachable at %s: %w", c.baseURL, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("clash api returned status %d", response.StatusCode)
	}

	var decoded struct {
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return "", fmt.Errorf("failed to decode clash api response: %w", err)
	}
	return decoded.Mode, nil
}
