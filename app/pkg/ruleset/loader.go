// Package ruleset resolves a sing-box `route.rule_set` tag to a matcher that
// can be evaluated in this process, without a running sing-box.
//
// This exists because route rules in the wild are overwhelmingly rule-set
// driven: the production config this was built against has 27 route rules, 19
// of which carry no condition other than `rule_set`. Any offline reasoning
// about routing that treats a rule set as opaque therefore answers "I cannot
// tell" for almost every rule, which is not an answer.
//
// The contents are not re-downloaded. Remote sets are read from the same cache
// file sing-box itself populates, and local sets from their configured path —
// so what is evaluated here is what sing-box would evaluate, not a fresh copy
// that may differ from the one in force.
//
// Every failure is reported as a machine-readable Reason on the Set rather than
// as a silent miss. "geoip-cn has never been downloaded" and "geoip-cn does not
// contain this address" are opposite answers, and collapsing them into a false
// match verdict would make the whole feature untrustworthy.
package ruleset

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/sagernet/sing-box/option"
)

// Reason explains why a rule set could not be evaluated. It is a stable key,
// not prose: the UI translates it, and prose here would arrive in whichever
// language this process happens to speak.
type Reason string

const (
	// ReasonOK means the set was loaded and can be matched against.
	ReasonOK Reason = ""
	// ReasonUnknownTag means no rule set in the config carries this tag. The
	// config would not start, so this is a config error rather than a gap.
	ReasonUnknownTag Reason = "unknown_tag"
	// ReasonNotCached means a remote set has never been downloaded, or was
	// downloaded under a different cache_id.
	ReasonNotCached Reason = "not_cached"
	// ReasonCacheUnavailable means the cache file itself could not be read —
	// missing, or held by a running sing-box that would not share the lock.
	ReasonCacheUnavailable Reason = "cache_unavailable"
	// ReasonCacheDisabled means the config has no experimental.cache_file, so
	// remote sets live only in the running process's memory.
	ReasonCacheDisabled Reason = "cache_disabled"
	// ReasonFileMissing means a local set's path does not exist.
	ReasonFileMissing Reason = "file_missing"
	// ReasonUnsupportedVersion means the .srs binary is newer than the
	// sing-box library this panel is built against. See DecodeBinary.
	ReasonUnsupportedVersion Reason = "unsupported_srs_version"
	// ReasonParseError means the content was found but could not be decoded.
	ReasonParseError Reason = "parse_error"
	// ReasonInline is not a failure: an inline set carries its rules in the
	// config, and is loaded from there.
	ReasonInline Reason = ""
)

// Set is one rule set, loaded or explained.
type Set struct {
	Tag string `json:"tag"`
	// Type is the configured source: local, remote or inline.
	Type string `json:"type"`
	// Available reports whether Rules can be matched against. When false,
	// Reason says why and the caller must treat the set as undecidable —
	// NOT as a non-match.
	Available bool   `json:"available"`
	Reason    Reason `json:"reason,omitempty"`
	// Detail carries the underlying error for ReasonParseError and friends.
	Detail string `json:"detail,omitempty"`
	// RuleCount is the number of headless rules in the set, for display.
	RuleCount int `json:"rule_count,omitempty"`
	// UpdatedAt is when sing-box last downloaded a remote set. A stale set is
	// still usable, but a routing surprise is often just an old set.
	UpdatedAt time.Time `json:"updated_at,omitempty"`

	rules []matcher
}

// Match evaluates the set against a target.
//
// A set is a disjunction: sing-box returns true as soon as any of its rules
// matches (route/rule/rule_set_local.go). An unavailable set always answers
// VerdictUnknown, so a caller that never inspects Available still cannot
// mistake "could not read geoip-cn" for "the address is not in geoip-cn".
func (s *Set) Match(target Target) Verdict {
	if s == nil || !s.Available {
		return VerdictUnknown
	}
	unknown := false
	for _, rule := range s.rules {
		switch rule.match(target) {
		case VerdictYes:
			return VerdictYes
		case VerdictUnknown:
			unknown = true
		}
	}
	if unknown {
		return VerdictUnknown
	}
	return VerdictNo
}

// Loader resolves tags against one config. It is not safe for concurrent use
// by multiple goroutines without the mutex it already holds internally; a
// Loader is cheap, so prefer one per request.
type Loader struct {
	mu      sync.Mutex
	ctx     context.Context
	configs map[string]option.RuleSet
	cache   *CacheFile
	// cacheErr is the reason the cache file is unusable, resolved once, with
	// cacheDetail carrying the underlying message.
	cacheErr    Reason
	cacheDetail string
	cacheOnce sync.Once
	cachePath string
	cacheID   string
	loaded    map[string]*Set
}

// NewLoader builds a loader for one sing-box config.
//
// It takes the whole option set rather than just the rule sets because the
// cache file path lives under `experimental`, and a remote set is unreadable
// without it.
func NewLoader(ctx context.Context, options *option.Options) *Loader {
	loader := &Loader{
		ctx:     ctx,
		configs: make(map[string]option.RuleSet),
		loaded:  make(map[string]*Set),
	}
	if loader.ctx == nil {
		loader.ctx = context.Background()
	}
	if options == nil {
		return loader
	}
	if options.Route != nil {
		for _, set := range options.Route.RuleSet {
			loader.configs[set.Tag] = set
		}
	}
	if options.Experimental != nil && options.Experimental.CacheFile != nil {
		cacheFile := options.Experimental.CacheFile
		// `enabled: false` means sing-box keeps nothing on disk, so there is
		// no cache to read even if a path is configured.
		if cacheFile.Enabled {
			loader.cachePath = strings.TrimSpace(cacheFile.Path)
			if loader.cachePath == "" {
				loader.cachePath = DefaultCachePath
			}
			loader.cacheID = strings.TrimSpace(cacheFile.CacheID)
		}
	}
	return loader
}

// Tags lists every rule-set tag the config declares, in config order.
func (l *Loader) Tags() []string {
	tags := make([]string, 0, len(l.configs))
	for tag := range l.configs {
		tags = append(tags, tag)
	}
	return tags
}

// Get resolves one tag, loading it on first use. It never returns nil.
func (l *Loader) Get(tag string) *Set {
	l.mu.Lock()
	defer l.mu.Unlock()

	if set, ok := l.loaded[tag]; ok {
		return set
	}
	set := l.load(tag)
	l.loaded[tag] = set
	return set
}

// Close releases the cache file handle, if one was opened.
func (l *Loader) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cache != nil {
		l.cache.Close()
		l.cache = nil
	}
}
