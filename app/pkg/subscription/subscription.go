package subscription

import (
	"time"
)

type SubscriptionManager interface {
	Init() error
	List() ([]*Subscription, error)
	Get(id string) (*Subscription, error)
	Add(sub Subscription) error
	Update(id string, sub Subscription) error
	Delete(id string) error
	UpdateLastUpdate(id string) error
}

const (
	DefaultSubscriptionPath = "/etc/sing-box/subscriptions.json"
)

// Subscription represents a node subscription
type Subscription struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Enabled        bool      `json:"enabled"`
	AutoUpdate     bool      `json:"auto_update"`
	UpdateInterval string    `json:"update_interval"` // e.g., "24h"
	LastUpdate     time.Time `json:"last_update"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
