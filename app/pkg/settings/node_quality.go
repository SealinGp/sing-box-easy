package settings

import (
	"encoding/json"
	"fmt"
)

const KeyNodeQuality = "node_quality"

// NodeQuality defines how recent observations become a node's quality badge.
type NodeQuality struct {
	Samples     int `json:"samples"`
	MaxDelayMs  int `json:"max_delay_ms"`
	GoodPercent int `json:"good_percent"`
	FairPercent int `json:"fair_percent"`
}

func DefaultNodeQuality() NodeQuality {
	return NodeQuality{Samples: 10, MaxDelayMs: 500, GoodPercent: 80, FairPercent: 50}
}

func (q NodeQuality) Validate() error {
	if q.Samples < 1 || q.Samples > 100 {
		return fmt.Errorf("samples must be between 1 and 100")
	}
	if q.MaxDelayMs < 1 || q.MaxDelayMs > 60000 {
		return fmt.Errorf("max_delay_ms must be between 1 and 60000")
	}
	if q.FairPercent < 1 || q.GoodPercent > 100 || q.GoodPercent <= q.FairPercent {
		return fmt.Errorf("percentages must satisfy 1 <= fair_percent < good_percent <= 100")
	}
	return nil
}

func (m *ManagerXORM) GetNodeQuality() NodeQuality {
	var q NodeQuality
	raw, err := m.Get(KeyNodeQuality)
	if err != nil || json.Unmarshal([]byte(raw), &q) != nil || q.Validate() != nil {
		return DefaultNodeQuality()
	}
	return q
}
