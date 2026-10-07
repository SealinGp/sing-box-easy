package clashapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// The shape a sing-box 1.14.1 host actually returns: a synthetic GLOBAL group,
// groups carrying now/all, a group whose type is neither Selector nor URLTest,
// and tags with spaces, emoji and "|".
const proxiesBody = `{"proxies":{
  "GLOBAL":{"type":"Fallback","name":"GLOBAL","udp":true,"history":[],"all":["Media"],"now":"Media"},
  "Media":{"type":"Selector","name":"Media","udp":true,"history":[{"time":"2026-10-07T08:41:40Z","delay":189}],"now":"流媒体","all":["流媒体","🇭🇰香港 01 f45c32ff | sub_1"]},
  "流媒体":{"type":"URLTest","name":"流媒体","udp":true,"history":[],"now":"🇭🇰香港 01 f45c32ff | sub_1","all":["🇭🇰香港 01 f45c32ff | sub_1"]},
  "backup":{"type":"Fallback","name":"backup","udp":true,"history":[],"now":"","all":[]},
  "🇭🇰香港 01 f45c32ff | sub_1":{"type":"VLESS","name":"🇭🇰香港 01 f45c32ff | sub_1","udp":true,"history":[{"time":"2026-10-07T08:41:39Z","delay":111}]}
}}`

func TestProxiesKeepsConfigOrderAndDropsGlobal(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/proxies" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(proxiesBody))
	})

	proxies, err := client.Proxies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, proxy := range proxies {
		names = append(names, proxy.Name)
	}
	want := "Media,流媒体,backup,🇭🇰香港 01 f45c32ff | sub_1"
	if got := strings.Join(names, ","); got != want {
		t.Fatalf("order = %q, want %q", got, want)
	}
}

func TestProxiesTellsGroupsFromNodesByAll(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(proxiesBody))
	})
	proxies, err := client.Proxies(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	selector, emptyGroup, node := proxies[0], proxies[2], proxies[3]
	if !selector.IsGroup() || selector.Now != "流媒体" || len(selector.All) != 2 || selector.Delay != 189 {
		t.Errorf("selector = %+v", selector)
	}
	// A group of some other type, with no members, is
	// still a group.
	if !emptyGroup.IsGroup() || emptyGroup.Type != "Fallback" {
		t.Errorf("empty fallback group = %+v", emptyGroup)
	}
	if node.IsGroup() || node.Delay != 111 || node.TestedAt.IsZero() {
		t.Errorf("node = %+v", node)
	}
}

func TestSelectProxyEscapesTagAndSendsName(t *testing.T) {
	const group = "AI 手动 | a/b"
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s", r.Method)
		}
		if r.URL.Path != "/proxies/"+group {
			t.Errorf("decoded path = %q", r.URL.Path)
		}
		if strings.Contains(r.URL.EscapedPath(), " ") || strings.Count(r.URL.EscapedPath(), "/") != 2 {
			t.Errorf("escaped path = %q, the tag must stay one segment", r.URL.EscapedPath())
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{"name":"美国自动"}` {
			t.Errorf("body = %s", body)
		}
		w.WriteHeader(http.StatusNoContent)
	})
	if err := client.SelectProxy(context.Background(), group, "美国自动"); err != nil {
		t.Fatal(err)
	}
}

func TestSelectProxyClassifiesStatus(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   error
	}{
		{http.StatusBadRequest, `{"message":"Must be a Selector"}`, ErrSelectRejected},
		{http.StatusNotFound, `{"message":"Resource not found"}`, ErrProxyNotFound},
		{http.StatusUnauthorized, ``, ErrUnauthorized},
	}
	for _, tc := range cases {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		})
		err := client.SelectProxy(context.Background(), "g", "n")
		if !errors.Is(err, tc.want) {
			t.Errorf("status %d: err = %v, want %v", tc.status, err, tc.want)
		}
	}

	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"Must be a Selector"}`))
	})
	if err := client.SelectProxy(context.Background(), "g", "n"); !strings.Contains(err.Error(), "Must be a Selector") {
		t.Errorf("sing-box's reason was dropped: %v", err)
	}
}

func TestGroupDelayReturnsOnlyAnsweringMembers(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/group/自动选择/delay" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("timeout") != "5000" || r.URL.Query().Get("url") != DefaultDelayURL {
			t.Errorf("query = %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"香港 01":42,"日本 01":88}`))
	})
	delays, err := client.GroupDelay(context.Background(), "自动选择", DefaultDelayURL, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if len(delays) != 2 || delays["香港 01"] != 42 {
		t.Fatalf("delays = %v", delays)
	}
}

func TestGroupDelayClassifiesStatus(t *testing.T) {
	cases := map[int]error{
		http.StatusNotFound:       ErrProxyNotFound,
		http.StatusGatewayTimeout: ErrDelayFailed,
		http.StatusUnauthorized:   ErrUnauthorized,
	}
	for status, want := range cases {
		client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) })
		if _, err := client.GroupDelay(context.Background(), "g", DefaultDelayURL, time.Second); !errors.Is(err, want) {
			t.Errorf("status %d: err = %v, want %v", status, err, want)
		}
	}
}

func TestCloseConnectionAndCloseAll(t *testing.T) {
	var seen []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		seen = append(seen, r.Method+" "+r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	if err := client.CloseConnection(context.Background(), "9c9077cf"); err != nil {
		t.Fatal(err)
	}
	if err := client.CloseAllConnections(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(seen, ","); got != "DELETE /connections/9c9077cf,DELETE /connections" {
		t.Fatalf("requests = %q", got)
	}
}

func TestCloseConnectionReportsUnauthorized(t *testing.T) {
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusUnauthorized) })
	if err := client.CloseAllConnections(context.Background()); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("err = %v", err)
	}
}
