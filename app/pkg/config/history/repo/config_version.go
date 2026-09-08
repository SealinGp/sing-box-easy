package repo

import "time"

// ConfigVersion is a historical snapshot of the sing-box config, stored in the
// database so the active config directory only ever holds the live config.json.
type ConfigVersion struct {
	ID        int64     `xorm:"pk autoincr 'id'" json:"id"`
	Content   string    `xorm:"text notnull 'content'" json:"-"` // full config JSON; never listed in bulk
	Size      int       `xorm:"notnull default(0) 'size'" json:"size"`
	CreatedAt time.Time `xorm:"created 'created_at'" json:"created_at"`
}

// TableName specifies the table name for ConfigVersion.
func (ConfigVersion) TableName() string {
	return "config_versions"
}
