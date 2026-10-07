// Package proxies is the runtime view of sing-box's outbound groups: which
// node each group is routing through RIGHT NOW, how fast each member last
// tested, and the one mutation the running process accepts — switching a
// selector.
//
// It is deliberately separate from `singbox/config/outbounds`, which edits the
// config document. The two answer different questions and must not be confused:
// that package says what the groups are DEFINED as; this one says what the
// running process is DOING with them. A selection made here never touches
// config.json — sing-box's own `experimental.cache_file` is what carries it
// across a restart, and the view reports whether that is enabled.
package proxies

import "github.com/SealinGp/sing-box-easy/app/pkg/singbox/clashapi"

// selectorType is the only group type sing-box lets a client switch
// (experimental/clashapi/proxies.go answers "Must be a Selector" otherwise).
const selectorType = "Selector"

// maxResolveDepth bounds following `now` through nested groups. sing-box
// rejects a group cycle at start, so this only guards a payload that lies.
const maxResolveDepth = 16

// Member is one entry of a group's member list.
type Member struct {
	Name string `json:"name"`
	// Type is sing-box's display name ("VLESS", "URLTest", …).
	Type string `json:"type"`
	// Delay is the last URL-test latency in ms, 0 when untested or failed. For
	// a member that is itself a group it is the delay of the node that group
	// currently resolves to.
	Delay int `json:"delay"`
	// Group marks a member that is another group.
	Group bool `json:"group"`
}

// Group is one selector/urltest-style group of the running sing-box.
type Group struct {
	Name string `json:"name"`
	Type string `json:"type"`
	// Now is the member currently in use.
	Now string `json:"now"`
	// Delay is the latency of the node Now ultimately resolves to.
	Delay int `json:"delay"`
	// Switchable is true only for selectors.
	Switchable bool     `json:"switchable"`
	Members    []Member `json:"members"`
}

// View is the body of `GET /runtime/proxies`.
type View struct {
	Groups []Group `json:"groups"`
	// SelectionPersisted reports whether `experimental.cache_file` is enabled.
	// Without it a selection lasts until the next restart — and the panel
	// restarts sing-box on every config save.
	SelectionPersisted bool `json:"selection_persisted"`
}

// BuildView shapes the raw proxy list into groups, in config order.
//
// A member the running config does not contain is still listed (with an empty
// type) rather than dropped: sing-box named it, and hiding it would make the
// member count disagree with the group's own.
func BuildView(list []clashapi.Proxy, selectionPersisted bool) View {
	byName := make(map[string]clashapi.Proxy, len(list))
	for _, proxy := range list {
		byName[proxy.Name] = proxy
	}

	groups := make([]Group, 0)
	for _, proxy := range list {
		if !proxy.IsGroup() {
			continue
		}
		members := make([]Member, 0, len(proxy.All))
		for _, name := range proxy.All {
			member := byName[name]
			members = append(members, Member{
				Name:  name,
				Type:  member.Type,
				Delay: resolveDelay(byName, name),
				Group: member.IsGroup(),
			})
		}
		groups = append(groups, Group{
			Name:       proxy.Name,
			Type:       proxy.Type,
			Now:        proxy.Now,
			Delay:      resolveDelay(byName, proxy.Now),
			Switchable: proxy.Type == selectorType,
			Members:    members,
		})
	}
	return View{Groups: groups, SelectionPersisted: selectionPersisted}
}

// resolveDelay follows `now` through nested groups to the node actually dialled
// and returns that node's delay. A group's own history is not used: for a
// selector it is whatever the last probe through it happened to record, which
// can describe a member that is no longer selected.
func resolveDelay(byName map[string]clashapi.Proxy, name string) int {
	for depth := 0; depth < maxResolveDepth; depth++ {
		proxy, ok := byName[name]
		if !ok {
			return 0
		}
		if !proxy.IsGroup() {
			return proxy.Delay
		}
		if proxy.Now == "" {
			return 0
		}
		name = proxy.Now
	}
	return 0
}
