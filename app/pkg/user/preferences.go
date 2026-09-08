package user

import (
	"encoding/json"
	"errors"

	"github.com/SealinGp/sing-box-easy/app/pkg/user/repo"
)

var ErrInvalidPreferences = errors.New("overview_order must contain unique, known card IDs")

var overviewCards = []string{"route-topology", "service-status", "subscriptions", "dns-probe", "route-probe", "api-endpoints"}

type Preferences struct {
	OverviewOrder []string `json:"overview_order"`
}

// Complete older layouts when new cards are introduced; ignore retired IDs.
func normalizeOverviewOrder(order []string) []string {
	result := make([]string, 0, len(overviewCards))
	seen := make(map[string]bool)
	for _, id := range append(append([]string{}, order...), overviewCards...) {
		for _, known := range overviewCards {
			if id == known && !seen[id] {
				result = append(result, id)
				seen[id] = true
			}
		}
	}
	return result
}

func (m *ManagerXORM) GetPreferences(id int64) (*Preferences, error) {
	u, err := m.GetUserByID(id)
	if err != nil {
		return nil, err
	}
	var order []string
	if u.OverviewOrder != "" {
		if err := json.Unmarshal([]byte(u.OverviewOrder), &order); err != nil {
			return nil, err
		}
	}
	return &Preferences{OverviewOrder: normalizeOverviewOrder(order)}, nil
}

// Updates only the caller's layout column, preserving profile and credentials.
// An empty order resets the layout to defaults.
func (m *ManagerXORM) UpdatePreferences(id int64, preferences Preferences) (*Preferences, error) {
	seen := make(map[string]bool)
	for _, id := range preferences.OverviewOrder {
		known := false
		for _, card := range overviewCards {
			if id == card {
				known = true
				break
			}
		}
		if !known || seen[id] {
			return nil, ErrInvalidPreferences
		}
		seen[id] = true
	}
	order := normalizeOverviewOrder(preferences.OverviewOrder)
	encoded, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	affected, err := m.e.ID(id).Cols("overview_order").Update(&repo.User{OverviewOrder: string(encoded)})
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrUserNotFound
	}
	return &Preferences{OverviewOrder: order}, nil
}
