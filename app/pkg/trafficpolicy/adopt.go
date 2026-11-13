package trafficpolicy

import (
	"fmt"
	"time"
)

// Compatible adoption requires the entire regenerated rule block to match the
// current configuration. Tags locate candidates, but never authorize mutation.
func acceptSettings(old Policy, state State, d Rule) (Policy, Group, error) {
	g, err := groupByID(state, old.GroupID)
	if err != nil {
		return old, g, err
	}
	if g.ID == "" {
		return old, g, fmt.Errorf("this policy has no managed DNS settings to adopt")
	}
	servers := []string{}
	timeout := 0
	for i := range g.Servers {
		tag := fmt.Sprintf("%s_%d", old.Prefix, i+1)
		var match Rule
		count := 0
		for _, r := range rules(d, "dns") {
			if r["tag"] == tag {
				match = r
				count++
			}
		}
		if count != 1 {
			return old, g, fmt.Errorf("DNS rules are missing or ambiguous; stop managing to preserve arbitrary edits")
		}
		server, _ := match["server"].(string)
		duration, _ := match["timeout"].(string)
		parsed, err := time.ParseDuration(duration)
		if err != nil || parsed%time.Second != 0 {
			return old, g, fmt.Errorf("manual timeout cannot be represented by group settings")
		}
		seconds := int(parsed / time.Second)
		if i > 0 && seconds != timeout {
			return old, g, fmt.Errorf("all group members must have the same timeout")
		}
		timeout = seconds
		servers = append(servers, server)
	}
	g.Servers = servers
	g.Timeout = timeout
	if err = validateGroup(&g, d, state); err != nil {
		return old, g, err
	}
	next := old
	if err = generate(&next, g, d); err != nil {
		return old, g, err
	}
	if policyStatus(next, d) != "synced" {
		return old, g, fmt.Errorf("only compatible DNS server and timeout edits can be adopted; stop managing to keep arbitrary edits")
	}
	return next, g, nil
}
