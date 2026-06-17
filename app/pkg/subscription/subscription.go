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
	UpdateInfo(id string, info []SubInfo) error
}

const (
	DefaultSubscriptionPath = "/etc/sing-box/subscriptions.json"
)

// SubInfo is one generic account-metadata entry extracted from a subscription's
// "info nodes" (loopback-server pseudo-nodes whose name is a "key：value" pair,
// e.g. "剩余流量：4.59 TB", "套餐到期：2026-10-19"). Keys are provider-defined and
// language-specific, so they are stored verbatim rather than mapped to fixed
// fields.
type SubInfo struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// Subscription represents a node subscription
type Subscription struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	Enabled        bool      `json:"enabled"`
	AutoUpdate     bool      `json:"auto_update"`
	UpdateInterval string    `json:"update_interval"` // e.g., "24h"
	LastUpdate     time.Time `json:"last_update"`
	Info           []SubInfo `json:"info,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
