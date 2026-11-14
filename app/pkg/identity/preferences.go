package identity

import (
	"encoding/json"
	"errors"

	"github.com/SealinGp/sing-box-easy/app/pkg/identity/repo"
)

var ErrInvalidPreferences = errors.New("overview_order and overview_hidden must contain unique, known card IDs")

var overviewCards = []string{"route-topology", "service-status", "connections", "latency", "network-info", "subscriptions", "dns-probe", "route-probe", "api-endpoints"}

type Preferences struct {
	OverviewHidden []string `json:"overview_hidden"`
	OverviewOrder  []string `json:"overview_order"`
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
	hidden := []string{}
	if u.OverviewHidden != "" {
		if err := json.Unmarshal([]byte(u.OverviewHidden), &hidden); err != nil {
			return nil, err
		}
	}
	return &Preferences{OverviewOrder: normalizeOverviewOrder(order), OverviewHidden: hidden}, nil
}

// Updates only the caller's layout column, preserving profile and credentials.
// An empty order resets the layout to defaults.
func (m *ManagerXORM) UpdatePreferences(id int64, preferences Preferences) (*Preferences, error) {
	for _, ids := range [][]string{preferences.OverviewOrder, preferences.OverviewHidden} {
		seen := make(map[string]bool)
		for _, id := range ids {
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
	}
	order := normalizeOverviewOrder(preferences.OverviewOrder)
	encoded, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	update := &repo.User{OverviewOrder: string(encoded)}
	columns := []string{"overview_order"}
	if preferences.OverviewHidden != nil {
		hidden, err := json.Marshal(preferences.OverviewHidden)
		if err != nil {
			return nil, err
		}
		update.OverviewHidden = string(hidden)
		columns = append(columns, "overview_hidden")
	}
	affected, err := m.e.ID(id).Cols(columns...).Update(update)
	if err != nil {
		return nil, err
	}
	if affected == 0 {
		return nil, ErrUserNotFound
	}
	return m.GetPreferences(id)
}
