package subscription

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type OrphanSubscription struct {
	ID    string   `json:"id"`
	Nodes []string `json:"nodes"`
}
type OrphanPlan struct {
	Revision      string               `json:"revision"`
	Subscriptions []OrphanSubscription `json:"subscriptions"`
	NodeCount     int                  `json:"node_count"`
	UpdatedGroups []string             `json:"updated_groups"`
	RemovedGroups []string             `json:"removed_groups"`
	Blockers      []string             `json:"blockers"`
}

var subscriptionSuffix = regexp.MustCompile(`^sub_[A-Za-z0-9_-]+$`)

func orphanOwner(tag string) string {
	index := strings.LastIndex(tag, " | ")
	if index < 0 {
		return ""
	}
	id := tag[index+3:]
	if subscriptionSuffix.MatchString(id) {
		return id
	}
	return ""
}
func stringValue(raw json.RawMessage) string {
	var value string
	_ = json.Unmarshal(raw, &value)
	return value
}
func orphanPlan(sections map[string]json.RawMessage, known map[string]bool) (OrphanPlan, json.RawMessage, error) {
	plan := OrphanPlan{Subscriptions: []OrphanSubscription{}, UpdatedGroups: []string{}, RemovedGroups: []string{}, Blockers: []string{}}
	raw, err := json.Marshal(sections)
	if err != nil {
		return plan, nil, err
	}
	ids := make([]string, 0, len(known))
	for id := range known {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	encodedIDs, _ := json.Marshal(ids)
	digest := sha256.Sum256(append(raw, encodedIDs...))
	plan.Revision = hex.EncodeToString(digest[:])
	var records []map[string]json.RawMessage
	if data := sections["outbounds"]; len(data) > 0 {
		if err := json.Unmarshal(data, &records); err != nil {
			return plan, nil, err
		}
	}
	deleted := map[string]bool{}
	owners := map[string][]string{}
	for _, record := range records {
		tag := stringValue(record["tag"])
		id := orphanOwner(tag)
		if id != "" && !known[id] {
			owners[id] = append(owners[id], tag)
			deleted[tag] = true
			plan.NodeCount++
		}
	}
	ids = nil
	for id := range owners {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		plan.Subscriptions = append(plan.Subscriptions, OrphanSubscription{ID: id, Nodes: owners[id]})
	}
	// Remove only groups made empty by this cleanup, including parent groups.
	for {
		changed := false
		for _, record := range records {
			tag, kind := stringValue(record["tag"]), stringValue(record["type"])
			if deleted[tag] || (kind != "selector" && kind != "urltest") {
				continue
			}
			var members []string
			if err := json.Unmarshal(record["outbounds"], &members); err != nil {
				return plan, nil, fmt.Errorf("group %q: %w", tag, err)
			}
			remaining := 0
			for _, member := range members {
				if !deleted[member] {
					remaining++
				}
			}
			if len(members) > 0 && remaining == 0 {
				deleted[tag] = true
				plan.RemovedGroups = append(plan.RemovedGroups, tag)
				changed = true
			}
		}
		if !changed {
			break
		}
	}
	kept := make([]map[string]json.RawMessage, 0, len(records))
	for _, record := range records {
		tag, kind := stringValue(record["tag"]), stringValue(record["type"])
		if deleted[tag] {
			continue
		}
		if kind == "selector" || kind == "urltest" {
			var members []string
			_ = json.Unmarshal(record["outbounds"], &members)
			next := make([]string, 0, len(members))
			for _, member := range members {
				if !deleted[member] {
					next = append(next, member)
				}
			}
			changed := len(next) != len(members)
			if changed {
				record["outbounds"], _ = json.Marshal(next)
			}
			if deleted[stringValue(record["default"])] {
				delete(record, "default")
				changed = true
			}
			if changed {
				plan.UpdatedGroups = append(plan.UpdatedGroups, tag)
			}
		}
		kept = append(kept, record)
	}
	outbounds, err := json.Marshal(kept)
	if err != nil {
		return plan, nil, err
	}
	candidate := make(map[string]json.RawMessage, len(sections))
	for k, v := range sections {
		candidate[k] = v
	}
	candidate["outbounds"] = outbounds
	after, _ := json.Marshal(candidate)
	// Refuse unrecognised references rather than silently rewrite routing policy.
	var tree any
	_ = json.Unmarshal(after, &tree)
	findRemovedReferences(tree, "config", deleted, &plan.Blockers)
	var route struct {
		Final string `json:"final"`
	}
	_ = json.Unmarshal(sections["route"], &route)
	if route.Final == "" && len(records) > 0 && deleted[stringValue(records[0]["tag"])] {
		plan.Blockers = append(plan.Blockers, "route.final (implicit) → "+stringValue(records[0]["tag"]))
	}
	sort.Strings(plan.Blockers)
	plan.Blockers = uniqueStrings(plan.Blockers)
	return plan, outbounds, nil
}
func findRemovedReferences(value any, path string, deleted map[string]bool, result *[]string) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			findRemovedReferences(child, path+"."+key, deleted, result)
		}
	case []any:
		for index, child := range v {
			findRemovedReferences(child, fmt.Sprintf("%s[%d]", path, index), deleted, result)
		}
	case string:
		if deleted[v] {
			*result = append(*result, path+" → "+v)
		}
	}
}
func uniqueStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if len(result) == 0 || value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}
