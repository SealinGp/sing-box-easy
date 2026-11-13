package trafficpolicy

import (
	"fmt"
	"net/netip"
	"strings"
)

func validateGroup(g *Group, d Rule, s State) error {
	g.Name = strings.TrimSpace(g.Name)
	if g.Name == "" || len(g.Name) > 120 {
		return fmt.Errorf("group name is required (maximum 120 characters)")
	}
	for _, other := range s.Groups {
		if other.ID != g.ID && strings.EqualFold(other.Name, g.Name) {
			return fmt.Errorf("group name already exists")
		}
	}
	if len(g.Servers) < 2 || len(g.Servers) > 16 {
		return fmt.Errorf("choose between 2 and 16 DNS servers")
	}
	seen := map[string]bool{}
	available := tags(section(d, "dns")["servers"])
	for _, server := range g.Servers {
		if seen[server] || !has(available, server) {
			return fmt.Errorf("duplicate or missing DNS server: %s", server)
		}
		seen[server] = true
	}
	if g.Timeout < 1 || g.Timeout > 30 {
		return fmt.Errorf("timeout must be between 1 and 30 seconds")
	}
	if g.Fallback != "strict" && g.Fallback != "general" {
		return fmt.Errorf("invalid DNS fallback behavior")
	}
	return nil
}
func generate(p *Policy, g Group, d Rule) error {
	if p.DNSServer != "" {
		if p.GroupID != "" {
			return fmt.Errorf("choose either a DNS group or a single server")
		}
		if !has(tags(section(d, "dns")["servers"]), p.DNSServer) {
			return fmt.Errorf("DNS server does not exist")
		}
		if p.Timeout < 1 || p.Timeout > 30 {
			return fmt.Errorf("timeout must be between 1 and 30 seconds")
		}
		if p.Fallback != "strict" && p.Fallback != "general" {
			return fmt.Errorf("invalid DNS fallback behavior")
		}
		g = Group{Servers: []string{p.DNSServer}, Timeout: p.Timeout, Fallback: p.Fallback}
	}

	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" || len(p.Name) > 120 {
		return fmt.Errorf("policy name is required (maximum 120 characters)")
	}
	if !has(tags(d["outbounds"]), p.Outbound) {
		return fmt.Errorf("outbound does not exist: %s", p.Outbound)
	}
	if p.Mode != "or" && p.Mode != "and" {
		return fmt.Errorf("invalid condition logic")
	}
	if len(p.Conditions) == 0 || len(p.Conditions) > 32 {
		return fmt.Errorf("choose between 1 and 32 match conditions")
	}
	branches := []Rule{}
	merged := map[string]int{}
	hasIP := false
	for i, c := range p.Conditions {
		if !has([]string{"domain", "domain_suffix", "rule_set", "ip_cidr"}, c.Kind) {
			return fmt.Errorf("unsupported matcher: %s", c.Kind)
		}
		values := []string{}
		for _, v := range c.Values {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			if len(v) > 253 || strings.ContainsAny(v, "\r\n\t ") {
				return fmt.Errorf("invalid match value: %s", v)
			}
			if c.Kind == "ip_cidr" {
				if _, err := netip.ParsePrefix(v); err != nil {
					return fmt.Errorf("invalid IP range: %s", v)
				}
				hasIP = true
			}
			if c.Kind == "rule_set" && !has(tags(section(d, "route")["rule_set"]), v) {
				return fmt.Errorf("rule set does not exist: %s", v)
			}
			if !has(values, v) {
				values = append(values, v)
			}
		}
		if len(values) == 0 || len(values) > 1000 {
			return fmt.Errorf("each condition needs 1–1000 values")
		}
		p.Conditions[i].Values = values
		if index, ok := merged[c.Kind]; ok && p.Mode == "or" {
			previous := branches[index][c.Kind].([]string)
			for _, v := range values {
				if !has(previous, v) {
					previous = append(previous, v)
				}
			}
			branches[index][c.Kind] = previous
		} else {
			merged[c.Kind] = len(branches)
			branches = append(branches, Rule{c.Kind: values})
		}
	}
	if hasIP && (p.GroupID != "" || p.DNSServer != "") {
		return fmt.Errorf("IP conditions cannot select DNS before resolution; leave DNS unchanged")
	}
	match := Rule{}
	if len(branches) == 1 {
		match = branches[0]
	} else {
		match = Rule{"type": "logical", "mode": p.Mode, "rules": branches}
	}
	makeRule := func(fields Rule) Rule {
		r := Rule{}
		for k, v := range match {
			r[k] = v
		}
		for k, v := range fields {
			r[k] = v
		}
		return r
	}
	p.Route = []Rule{makeRule(Rule{"action": "route", "outbound": p.Outbound})}
	p.DNS = []Rule{}
	if p.GroupID == "" && p.DNSServer == "" {
		return nil
	}
	for i, server := range g.Servers {
		p.DNS = append(p.DNS, makeRule(Rule{"action": "evaluate", "server": server, "tag": fmt.Sprintf("%s_%d", p.Prefix, i+1), "timeout": fmt.Sprintf("%ds", g.Timeout)}))
	}
	for i := range g.Servers {
		r := Rule{"match_response": fmt.Sprintf("%s_%d", p.Prefix, i+1), "response_rcode": "NOERROR", "action": "respond"}
		if len(g.Servers) > 1 {
			r["race"] = true
		}
		p.DNS = append(p.DNS, r)
	}
	if g.Fallback == "strict" {
		p.DNS = append(p.DNS, makeRule(Rule{"action": "reject", "method": "default", "no_drop": true}))
	}
	return nil
}

// Locate an owned contiguous block by its complete canonical content. Duplicate
// matches, partial edits and reordered members are ambiguous and fail closed.
func locate(all, owned []Rule) int {
	if len(owned) == 0 {
		return 0
	}
	found := -1
	for i := 0; i+len(owned) <= len(all); i++ {
		if hash(all[i:i+len(owned)]) == hash(owned) {
			if found >= 0 {
				return -1
			}
			found = i
		}
	}
	return found
}
func policyStatus(p Policy, d Rule) string {
	r, dns := rules(d, "route"), rules(d, "dns")
	if locate(r, p.Route) >= 0 && locate(dns, p.DNS) >= 0 {
		return "synced"
	}
	for _, members := range []struct{ all, owned []Rule }{{r, p.Route}, {dns, p.DNS}} {
		for _, a := range members.all {
			for _, b := range members.owned {
				if hash(a) == hash(b) {
					return "modified"
				}
			}
		}
	}
	// Tags are only evidence of drift, never proof of ownership for removal.
	for _, r := range dns {
		tag, _ := r["tag"].(string)
		ref, _ := r["match_response"].(string)
		if strings.HasPrefix(tag, p.Prefix+"_") || strings.HasPrefix(ref, p.Prefix+"_") {
			return "modified"
		}
	}
	return "missing"
}
func externalReferences(p Policy, d Rule) bool {
	ownedTags := []string{}
	for _, r := range p.DNS {
		if tag, ok := r["tag"].(string); ok {
			ownedTags = append(ownedTags, tag)
		}
	}
	start := locate(rules(d, "dns"), p.DNS)
	var references func(any) bool
	references = func(v any) bool {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				if k == "match_response" {
					if tag, ok := v.(string); ok && has(ownedTags, tag) {
						return true
					}
					for _, a := range array(v) {
						if tag, ok := a.(string); ok && has(ownedTags, tag) {
							return true
						}
					}
				}
				if references(v) {
					return true
				}
			}
		case []any:
			for _, a := range x {
				if references(a) {
					return true
				}
			}
		}
		return false
	}
	for i, r := range rules(d, "dns") {
		if i >= start && i < start+len(p.DNS) {
			continue
		}
		if references(r) {
			return true
		}
	}
	return false
}
func removeOwned(d Rule, p Policy) error {
	if policyStatus(p, d) != "synced" {
		return fmt.Errorf("policy %s has configuration drift; review or stop managing it first", p.Name)
	}
	if externalReferences(p, d) {
		return fmt.Errorf("other DNS rules reference responses owned by %s", p.Name)
	}
	for _, x := range []struct {
		name  string
		owned []Rule
	}{{"route", p.Route}, {"dns", p.DNS}} {
		if len(x.owned) == 0 {
			continue
		}
		all := rules(d, x.name)
		i := locate(all, x.owned)
		putRules(d, x.name, append(all[:i:i], all[i+len(x.owned):]...))
	}
	return nil
}
func insert(d Rule, p Policy) error {
	for _, x := range []struct {
		name    string
		members []Rule
		index   int
	}{{"route", p.Route, p.RouteIndex}, {"dns", p.DNS, p.DNSIndex}} {
		if len(x.members) == 0 {
			continue
		}
		all := rules(d, x.name)
		if x.index < 0 || x.index > len(all) {
			return fmt.Errorf("%s insertion position is out of range", x.name)
		}
		out := append([]Rule{}, all[:x.index]...)
		out = append(out, x.members...)
		out = append(out, all[x.index:]...)
		putRules(d, x.name, out)
	}
	return nil
}

// Ownership may be intact while a manually removed dependency makes a policy
// unusable. Report that separately so the UI can offer an ordinary repair edit.
func reportedStatus(p Policy, state State, d Rule) string {
	status := policyStatus(p, d)
	if status != "synced" {
		return status
	}
	if !has(tags(d["outbounds"]), p.Outbound) {
		return "missing_dependency"
	}
	if p.GroupID != "" {
		g, err := groupByID(state, p.GroupID)
		if err != nil {
			return "missing_dependency"
		}
		for _, server := range g.Servers {
			if !has(tags(section(d, "dns")["servers"]), server) {
				return "missing_dependency"
			}
		}
	}
	if p.DNSServer != "" && !has(tags(section(d, "dns")["servers"]), p.DNSServer) {
		return "missing_dependency"
	}
	for _, c := range p.Conditions {
		if c.Kind == "rule_set" {
			for _, tag := range c.Values {
				if !has(tags(section(d, "route")["rule_set"]), tag) {
					return "missing_dependency"
				}
			}
		}
	}
	return status
}
