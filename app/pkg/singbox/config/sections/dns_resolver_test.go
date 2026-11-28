package sections

import (
	"encoding/json"
	"testing"
)

func TestPreserveImplicitDNSResolver(t *testing.T) {
	google := json.RawMessage(`{"type":"tls","tag":"google","server":"8.8.8.8"}`)
	added := json.RawMessage(`{"type":"udp","tag":"us-1","server":"7.7.7.7"}`)
	for _, tc := range []struct {
		name, route, want string
		before, after     []json.RawMessage
	}{
		{"second server", `{"rules":[{"port":53,"action":"hijack-dns"}],"future":true}`, `"google"`, []json.RawMessage{google}, []json.RawMessage{google, added}},
		{"explicit object", `{"default_domain_resolver":{"server":"us-1","strategy":"ipv4_only"}}`, `{"server":"us-1","strategy":"ipv4_only"}`, []json.RawMessage{google}, []json.RawMessage{google, added}},
		{"explicit string", `{"default_domain_resolver":"us-1"}`, `"us-1"`, []json.RawMessage{google}, []json.RawMessage{google, added}},
		{"already ambiguous", `{}`, ``, []json.RawMessage{google, added}, []json.RawMessage{google, added, added}},
		{"removed original", `{}`, ``, []json.RawMessage{google}, []json.RawMessage{added, added}},
		{"no tagged old server", `{}`, ``, []json.RawMessage{json.RawMessage(`{"type":"local"}`)}, []json.RawMessage{google, added}},
		{"only one", `{}`, ``, []json.RawMessage{google}, []json.RawMessage{google}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sections := map[string]json.RawMessage{"route": json.RawMessage(tc.route), "inbounds": json.RawMessage(`[{"future":true}]`)}
			if err := preserveImplicitDNSResolver(sections, tc.before, tc.after); err != nil {
				t.Fatal(err)
			}
			var route map[string]json.RawMessage
			if err := json.Unmarshal(sections["route"], &route); err != nil {
				t.Fatal(err)
			}
			if string(route["default_domain_resolver"]) != tc.want {
				t.Fatalf("resolver %s, want %s", route["default_domain_resolver"], tc.want)
			}
			if tc.name == "second server" && (string(route["future"]) != "true" || len(route["rules"]) == 0) {
				t.Fatal("route siblings lost")
			}
			if string(sections["inbounds"]) != `[{"future":true}]` {
				t.Fatal("unrelated section changed")
			}
		})
	}
}
