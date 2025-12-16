package database

import (
	"time"
)

// InitState represents the initialization state table
type InitState struct {
	ID                 int       `xorm:"pk autoincr 'id'" json:"id"`
	Initialized        bool      `xorm:"notnull default(0)" json:"initialized"`
	SingBoxInstalled   bool      `xorm:"'sing_box_installed' notnull default(0)" json:"sing_box_installed"`
	ConfigGenerated    bool      `xorm:"'config_generated' notnull default(0)" json:"config_generated"`
	DashboardInstalled bool      `xorm:"'dashboard_installed' notnull default(0)" json:"dashboard_installed"`
	SingBoxVersion     string    `xorm:"'sing_box_version' default('')" json:"sing_box_version"`
	InitTime           time.Time `xorm:"'init_time' null" json:"init_time,omitempty"`
	CreatedAt          time.Time `xorm:"created" json:"created_at"`
	UpdatedAt          time.Time `xorm:"updated" json:"updated_at"`
}

// TableName specifies the table name for InitState
func (InitState) TableName() string {
	return "init_state"
}

// Subscription represents a node subscription table
type Subscription struct {
	ID             string    `xorm:"'id' pk varchar(255)" json:"id"`
	Name           string    `xorm:"'name' notnull index" json:"name"`
	URL            string    `xorm:"'url' notnull text" json:"url"`
	Enabled        bool      `xorm:"'enabled' notnull default(1)" json:"enabled"`
	AutoUpdate     bool      `xorm:"'auto_update' notnull default(0)" json:"auto_update"`
	UpdateInterval string    `xorm:"'update_interval' default('24h')" json:"update_interval"`
	LastUpdate     time.Time `xorm:"'last_update' null" json:"last_update,omitempty"`
	CreatedAt      time.Time `xorm:"created" json:"created_at"`
	UpdatedAt      time.Time `xorm:"updated" json:"updated_at"`
}

// TableName specifies the table name for Subscription
func (Subscription) TableName() string {
	return "subscriptions"
}
