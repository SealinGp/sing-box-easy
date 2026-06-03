package repo

import "time"

// FilterRule is a persisted "Filter" (a.k.a. the user's "Region Node"): a named,
// keyword/tag-matched bucket of endpoint outbounds. Geography is just one use —
// a Filter named "YouTube" matching {Hong Kong, Streaming} is equally valid.
//
// Matchers is a JSON-encoded ordered array of {type,value} objects (see the
// noderules.Matcher domain type). FilterIDs/membership is computed at apply
// time from the live endpoint set, not stored here.
type FilterRule struct {
	ID           string `xorm:"'id' pk varchar(255)" json:"id"`
	Name         string `xorm:"'name' notnull unique index" json:"name"`
	Matchers     string `xorm:"'matchers' text" json:"matchers"`
	OutboundType string `xorm:"'outbound_type' notnull default('urltest')" json:"outbound_type"`
	Priority     int    `xorm:"'priority' notnull default(0) index" json:"priority"`
	IsFallback   bool   `xorm:"'is_fallback' notnull default(0)" json:"is_fallback"`
	// urltest health-check settings (only used when OutboundType is "urltest").
	TestURL       string    `xorm:"'test_url' notnull default('')" json:"test_url"`
	TestInterval  string    `xorm:"'test_interval' notnull default('')" json:"test_interval"`
	TestTolerance int       `xorm:"'test_tolerance' notnull default(0)" json:"test_tolerance"`
	CreatedAt     time.Time `xorm:"created" json:"created_at"`
	UpdatedAt     time.Time `xorm:"updated" json:"updated_at"`
}

// TableName specifies the table name for FilterRule.
func (FilterRule) TableName() string {
	return "filter_rules"
}

// GroupRule is a persisted "Group" (a.k.a. the user's "Group Node"): a named,
// ordered set of Filters. It materializes as a sing-box selector whose members
// are the Filter outbound tags.
//
// FilterIDs is a JSON-encoded ordered array of FilterRule.ID.
type GroupRule struct {
	ID        string    `xorm:"'id' pk varchar(255)" json:"id"`
	Name      string    `xorm:"'name' notnull unique index" json:"name"`
	FilterIDs string    `xorm:"'filter_ids' text" json:"filter_ids"`
	Priority  int       `xorm:"'priority' notnull default(0) index" json:"priority"`
	CreatedAt time.Time `xorm:"created" json:"created_at"`
	UpdatedAt time.Time `xorm:"updated" json:"updated_at"`
}

// TableName specifies the table name for GroupRule.
func (GroupRule) TableName() string {
	return "group_rules"
}
