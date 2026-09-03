package dnsprobe

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/SealinGp/sing-box-easy/app/pkg/clashapi"
	"github.com/sagernet/sing-box/option"
)

// ErrClashAPIDisabled is returned when the running config has no clash_api
// block, so there is no way to ask sing-box what it resolved.
//
// An alias of the shared client's error, kept so existing callers and messages
// are unchanged.
var ErrClashAPIDisabled = clashapi.ErrDisabled

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

// ClashClient queries sing-box's Clash API for the DNS probe. The transport —
// controller URL, bearer, status classification — lives in package clashapi;
// this adds the two calls the probe needs.
type ClashClient struct {
	*clashapi.Client
}

// NewClashClient builds a client from the running sing-box config. It returns
// ErrClashAPIDisabled when the config has no external controller.
func NewClashClient(experimental *option.ExperimentalOptions) (*ClashClient, error) {
	client, err := clashapi.New(experimental)
	if err != nil {
		return nil, err
	}
	return &ClashClient{Client: client}, nil
}

// controllerURL is kept as a package-level name for the tests that pin the
// bind-address rewrite; the logic now lives in clashapi.
func controllerURL(controller string) (string, error) {
	return clashapi.ControllerURL(controller)
}

// Query asks sing-box to resolve name/qType through its own DNS router.
func (c *ClashClient) Query(name, qType string) (*LiveResult, error) {
	path := fmt.Sprintf("/dns/query?name=%s&type=%s", url.QueryEscape(name), url.QueryEscape(qType))

	var decoded clashResponse
	started := time.Now()
	if err := c.Get(context.Background(), path, &decoded); err != nil {
		return nil, err
	}
	elapsed := time.Since(started)

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
	var decoded struct {
		Mode string `json:"mode"`
	}
	if err := c.Get(context.Background(), "/configs", &decoded); err != nil {
		return "", err
	}
	return decoded.Mode, nil
}
