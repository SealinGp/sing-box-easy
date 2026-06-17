package repo

import "time"

// Subscription represents a node subscription table
type Subscription struct {
	ID             string    `xorm:"'id' pk varchar(255)" json:"id"`
	Name           string    `xorm:"'name' notnull index" json:"name"`
	URL            string    `xorm:"'url' notnull text" json:"url"`
	AutoUpdate     bool      `xorm:"'auto_update' notnull default(0)" json:"auto_update"`
	UpdateInterval string    `xorm:"'update_interval' default('24h')" json:"update_interval"`
	LastUpdate     time.Time `xorm:"'last_update' null" json:"last_update,omitempty"`
	// Info is a JSON-encoded []subscription.SubInfo of generic account metadata
	// (traffic/expiry/reset) extracted from the feed's loopback "info nodes".
	Info string `xorm:"'info' text" json:"info"`
	// FetchMode / ProxyURL control how the URL is fetched (direct / clean_dns /
	// proxy) on censored networks. See subscription.FetchMode* constants.
	FetchMode string    `xorm:"'fetch_mode' notnull default('')" json:"fetch_mode"`
	ProxyURL  string    `xorm:"'proxy_url' notnull default('')" json:"proxy_url"`
	CreatedAt time.Time `xorm:"created" json:"created_at"`
	UpdatedAt      time.Time `xorm:"updated" json:"updated_at"`
}

// TableName specifies the table name for Subscription
func (Subscription) TableName() string {
	return "subscriptions"
}