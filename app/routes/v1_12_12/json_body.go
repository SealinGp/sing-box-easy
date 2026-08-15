package v1_13_0

import (
	"bytes"
	"encoding/json"
)

// dropEmptyJSONFields removes the named top-level keys from a JSON object body
// when their value is an empty string.
//
// sing-box types optional durations as badoption.Duration, whose UnmarshalJSON
// rejects "" outright:
//
//	time: invalid duration ""
//
// A web form naturally submits an empty string for "leave this unset", so a
// blank optional duration would fail the entire request with an error that
// names neither the field nor anything the operator can act on. Every such
// field is `omitempty` on the Go side, so removing the key is exactly
// equivalent to the client not sending it.
//
// Only an empty string is treated as "unset". null, 0 and false are values the
// caller chose and are passed through untouched, as is any body that is not a
// JSON object — a malformed payload must reach the binder so the caller sees
// the real parse error instead of a silent rewrite.
func dropEmptyJSONFields(body []byte, keys ...string) []byte {
	if len(keys) == 0 || len(body) == 0 {
		return body
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return body
	}

	changed := false
	for _, key := range keys {
		raw, ok := fields[key]
		if !ok {
			continue
		}
		// Require an actual JSON string literal. Unmarshalling `null` into a
		// string succeeds and leaves it empty, which would make an explicit
		// null indistinguishable from "" and silently drop it.
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 || trimmed[0] != '"' {
			continue
		}

		var value string
		if err := json.Unmarshal(trimmed, &value); err == nil && value == "" {
			delete(fields, key)
			changed = true
		}
	}
	if !changed {
		return body
	}

	rewritten, err := json.Marshal(fields)
	if err != nil {
		return body
	}
	return rewritten
}
