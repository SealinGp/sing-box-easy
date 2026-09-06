package subprobe

import "time"

// DefaultProbeURL is the target a subscription is tested against unless it
// names its own.
//
// It MUST be https. sing-box discards an `http://` test URL and silently
// substitutes its own default (see clashapi.DefaultDelayURL), so a plain-HTTP
// value would produce latency figures for an endpoint the operator never chose,
// with nothing anywhere reporting the substitution.
const DefaultProbeURL = "https://www.gstatic.com/generate_204"

// Settings are the operator-tunable knobs, read fresh on every run so an edit
// in the UI takes effect on the next tick rather than at the next restart —
// the same contract the subscription updater's InfoKeywordsProvider has.
type Settings struct {
	// Interval between full sweeps.
	Interval time.Duration
	// Timeout bounds ONE node's URL test.
	Timeout time.Duration
	// Concurrency is how many nodes of one subscription are tested at once.
	Concurrency int

	// MaxAge and MaxPoints are the two retention bounds. Both are applied
	// after every run and whichever bites first wins; a non-positive value
	// disables that bound.
	//
	// Two bounds rather than one because they fail differently: an age window
	// alone stops bounding anything if the operator shortens the interval (a
	// week at 1-minute sampling is 10,080 rows per subscription), and a count
	// alone silently shortens the visible history when they do the same. The
	// count protects the disk; the age protects the meaning of "7 days".
	MaxAge    time.Duration
	MaxPoints int
}

// Defaults sized for a home router: 4 subscriptions at these settings hold
// ~8k rows (~600 KB) at steady state.
const (
	DefaultInterval    = 10 * time.Minute
	DefaultTimeout     = 5 * time.Second
	DefaultConcurrency = 8
	DefaultMaxAge      = 7 * 24 * time.Hour
	DefaultMaxPoints   = 2016 // 7 days at the default 10-minute interval
)

// Bounds. The interval floor keeps a mistyped value from turning the panel into
// a load generator against the provider's nodes. The timeout ceiling is set by
// the Clash API client's own 10s request timeout — a longer probe timeout would
// be cut short by the transport before sing-box could answer 504, turning every
// slow node into an unexplained transport error.
const (
	MinInterval  = time.Minute
	MaxInterval  = 24 * time.Hour
	MinTimeout   = time.Second
	MaxTimeout   = 8 * time.Second
	MinPoints    = 60
	MaxPoints    = 20000
	MinRetention = 24 * time.Hour
	MaxRetention = 90 * 24 * time.Hour
)

// Normalize clamps every field into range and substitutes defaults for zero
// values, so a partially-populated or mis-stored Settings can never disable
// retention or spin the prober at an unusable rate.
func (s Settings) Normalize() Settings {
	if s.Interval <= 0 {
		s.Interval = DefaultInterval
	}
	s.Interval = clampDuration(s.Interval, MinInterval, MaxInterval)

	if s.Timeout <= 0 {
		s.Timeout = DefaultTimeout
	}
	s.Timeout = clampDuration(s.Timeout, MinTimeout, MaxTimeout)

	if s.Concurrency <= 0 {
		s.Concurrency = DefaultConcurrency
	}
	if s.Concurrency > 32 {
		s.Concurrency = 32
	}

	if s.MaxAge <= 0 {
		s.MaxAge = DefaultMaxAge
	}
	s.MaxAge = clampDuration(s.MaxAge, MinRetention, MaxRetention)

	if s.MaxPoints <= 0 {
		s.MaxPoints = DefaultMaxPoints
	}
	if s.MaxPoints < MinPoints {
		s.MaxPoints = MinPoints
	}
	if s.MaxPoints > MaxPoints {
		s.MaxPoints = MaxPoints
	}

	return s
}

func clampDuration(v, min, max time.Duration) time.Duration {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
