package node

import (
	"encoding/json"
)

type SubNodeParser interface {
	TypeName() string
	Schema() string
	Parse(uri string) (*SubNode, error)
	Validate() error
}

type Options any

// SubNode represents a shadowsocks subscription node
type SubNode struct {
	Tag  string `json:"tag"`  // Node display name (e.g., "🇭🇰 香港 01")
	Type string `json:"type"` // Protocol type (e.g., "shadowsocks")
	Options
}

// MarshalJSON implements custom JSON marshaling to flatten the Options field
func (s *SubNode) MarshalJSON() ([]byte, error) {
	// First marshal the Options to get its fields
	optionsJSON, err := json.Marshal(s.Options)
	if err != nil {
		return nil, err
	}

	// Unmarshal into a map
	var optionsMap map[string]interface{}
	if err := json.Unmarshal(optionsJSON, &optionsMap); err != nil {
		return nil, err
	}

	// Add tag and type to the map
	optionsMap["tag"] = s.Tag
	optionsMap["type"] = s.Type

	// Marshal the combined map
	return json.Marshal(optionsMap)
}
