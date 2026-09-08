package repo

import "time"

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
