// Package trafficpolicy manages reusable DNS settings and ownership of generated
// rules. Arbitrary existing rules remain unowned and are never silently merged.
package trafficpolicy

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type Rule = map[string]any
type Group struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Servers  []string `json:"servers"`
	Timeout  int      `json:"timeout"`
	Fallback string   `json:"fallback"`
}
type Condition struct {
	Kind   string   `json:"kind"`
	Values []string `json:"values"`
}
type Policy struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	GroupID    string      `json:"group_id"`
	DNSServer  string      `json:"dns_server,omitempty"`
	Timeout    int         `json:"timeout,omitempty"`
	Fallback   string      `json:"fallback,omitempty"`
	Outbound   string      `json:"outbound"`
	Mode       string      `json:"mode"`
	Conditions []Condition `json:"conditions"`
	// Explicit insertion positions: priority must be reviewed, never guessed.
	RouteIndex int    `json:"route_index"`
	DNSIndex   int    `json:"dns_index"`
	Prefix     string `json:"prefix"`
	Route      []Rule `json:"route_members"`
	DNS        []Rule `json:"dns_members"`
	Status     string `json:"status,omitempty"`
}
type State struct {
	Groups   []Group  `json:"groups"`
	Policies []Policy `json:"policies"`
}
type Snapshot struct {
	Recovery bool `json:"recovery"`
	State
	Revision   string   `json:"revision"`
	Servers    []string `json:"servers"`
	Outbounds  []string `json:"outbounds"`
	RuleSets   []string `json:"rule_sets"`
	RouteRules []Rule   `json:"route_rules"`
	DNSRules   []Rule   `json:"dns_rules"`
}
type Command struct {
	Action      string  `json:"action"`
	ID          string  `json:"id"`
	Replacement string  `json:"replacement"`
	Group       *Group  `json:"group,omitempty"`
	Policy      *Policy `json:"policy,omitempty"`
	Revision    string  `json:"revision"`
}
type Plan struct {
	Revision string   `json:"revision"`
	Command  Command  `json:"command"`
	Before   Rule     `json:"before"`
	After    Rule     `json:"after"`
	Affected []string `json:"affected"`
	// Internal candidate; only rule arrays are exposed, never unrelated secrets.
	state    State
	document []byte
}

func newID() (string, error) {
	b := make([]byte, 9)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func encoded(v any) []byte { b, _ := json.Marshal(v); return b }
func hash(v any) string    { b := sha256.Sum256(encoded(v)); return hex.EncodeToString(b[:]) }
func decode(raw []byte) (Rule, error) {
	var d Rule
	err := json.Unmarshal(raw, &d)
	if err == nil && d == nil {
		err = fmt.Errorf("configuration must be an object")
	}
	return d, err
}
func section(d Rule, name string) Rule {
	s, _ := d[name].(map[string]any)
	if s == nil {
		s = Rule{}
	}
	return s
}
func rules(d Rule, name string) []Rule {
	result := []Rule{}
	for _, r := range array(section(d, name)["rules"]) {
		if x, ok := r.(map[string]any); ok {
			result = append(result, x)
		}
	}
	return result
}
func array(v any) []any {
	switch a := v.(type) {
	case []any:
		return a
	case []Rule:
		out := make([]any, len(a))
		for i, x := range a {
			out[i] = x
		}
		return out
	case []string:
		out := make([]any, len(a))
		for i, x := range a {
			out[i] = x
		}
		return out
	}
	return nil
}
func tags(v any) []string {
	result := []string{}
	for _, x := range array(v) {
		if m, ok := x.(map[string]any); ok {
			if tag, ok := m["tag"].(string); ok {
				result = append(result, tag)
			}
		}
	}
	return result
}
func has(a []string, v string) bool {
	for _, x := range a {
		if x == v {
			return true
		}
	}
	return false
}
func putRules(d Rule, name string, r []Rule) { s := section(d, name); s["rules"] = r; d[name] = s }
func cloneState(s State) State               { var c State; _ = json.Unmarshal(encoded(s), &c); return c }
func ruleView(d Rule) Rule {
	return Rule{"route": Rule{"rules": rules(d, "route")}, "dns": Rule{"rules": rules(d, "dns")}}
}
