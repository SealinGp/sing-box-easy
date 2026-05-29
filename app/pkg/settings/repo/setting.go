package repo

import "time"

// Setting is a single key-value application setting. A generic KV table keeps
// the schema stable as new settings are added.
type Setting struct {
	Key       string    `xorm:"pk 'key'" json:"key"`
	Value     string    `xorm:"text notnull 'value'" json:"value"`
	UpdatedAt time.Time `xorm:"updated 'updated_at'" json:"updated_at"`
}

// TableName specifies the table name for Setting.
func (Setting) TableName() string {
	return "settings"
}
