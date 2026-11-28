package sections

import (
	"encoding/json"
	"fmt"
)

// With one DNS transport, sing-box implicitly uses it for outbound resolution.
// Adding a second removes that inference (fatal since 1.14). Preserve the old
// choice explicitly in the same validated transaction; never choose a new DNS
// provider or overwrite an operator's explicit resolver.
func preserveImplicitDNSResolver(sections map[string]json.RawMessage, before, after []json.RawMessage) error {
	if len(before) != 1 || len(after) < 2 {
		return nil
	}
	tag := rawStringField(before[0], "tag")
	if tag == "" {
		return nil
	}
	retained := false
	for _, server := range after {
		if rawStringField(server, "tag") == tag {
			retained = true
		}
	}
	if !retained {
		return nil
	}
	route := map[string]json.RawMessage{}
	if raw := sections["route"]; len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &route); err != nil {
			return fmt.Errorf("decode route configuration: %w", err)
		}
	}
	if raw, ok := route["default_domain_resolver"]; ok && string(raw) != "null" && string(raw) != `""` {
		return nil
	}
	route["default_domain_resolver"], _ = json.Marshal(tag)
	encoded, err := json.Marshal(route)
	if err != nil {
		return err
	}
	sections["route"] = encoded
	return nil
}
