package repo

// ProbeSample is ONE subscription's aggregate result for ONE probe run: how
// many of its nodes answered a URL test, and how fast the ones that did were.
//
// Deliberately one row per (subscription, run) rather than one per node. The
// question this feature answers — "is this provider any good?" — is a property
// of the feed, and the per-node breakdown that would justify ~60x the rows is
// only ever asked about the LATEST run (which the runner keeps in memory). On
// the config this was sized against, 4 subscriptions at the default 10-minute
// cadence produce ~4k rows a week; per-node storage would have made that 240k.
//
// `At` is a unix timestamp rather than a time.Time so the bucketing the history
// endpoint does (integer division into fixed-width buckets) is expressible in
// SQL on SQLite without date functions, and so a row is 8 bytes here instead of
// a driver-dependent string.
type ProbeSample struct {
	ID int64 `xorm:"pk autoincr 'id'" json:"id"`
	// SubID owns the sample. Indexed together with At because every read is
	// "one subscription, one time window" — see the store's Series.
	SubID string `xorm:"'sub_id' notnull index(sub_at)" json:"sub_id"`
	At    int64  `xorm:"'at' notnull index(sub_at)" json:"at"`

	// Total is every node the run considered — nodes owned by this
	// subscription and present in the running sing-box.
	Total int `xorm:"'total' notnull default(0)" json:"total"`
	// Reachable answered the URL test. availability = Reachable / Total.
	Reachable int `xorm:"'reachable' notnull default(0)" json:"reachable"`

	// Latency figures are over the REACHABLE nodes only. A node that timed out
	// has no latency, and folding its timeout in as "5000ms" would make a
	// subscription look slow when it is actually partly dead — two different
	// problems with two different fixes.
	AvgMs int `xorm:"'avg_ms' notnull default(0)" json:"avg_ms"`
	MinMs int `xorm:"'min_ms' notnull default(0)" json:"min_ms"`
	MaxMs int `xorm:"'max_ms' notnull default(0)" json:"max_ms"`
}

// TableName specifies the table name for ProbeSample.
func (ProbeSample) TableName() string {
	return "subscription_probe_samples"
}
