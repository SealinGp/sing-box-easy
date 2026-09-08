package ruleset

import (
	"os"
	"strings"

	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

// load resolves one tag. Callers hold l.mu.
func (l *Loader) load(tag string) *Set {
	options, ok := l.configs[tag]
	if !ok {
		return &Set{Tag: tag, Reason: ReasonUnknownTag}
	}

	set := &Set{Tag: tag, Type: options.Type}

	switch options.Type {
	case C.RuleSetTypeInline, "":
		// Inline sets carry their rules in the config itself, so there is
		// nothing to read and nothing that can be stale.
		set.Type = C.RuleSetTypeInline
		l.buildRules(set, options.InlineOptions.Rules)

	case C.RuleSetTypeLocal:
		content, err := os.ReadFile(options.LocalOptions.Path)
		if err != nil {
			if os.IsNotExist(err) {
				set.Reason = ReasonFileMissing
			} else {
				set.Reason = ReasonCacheUnavailable
			}
			set.Detail = err.Error()
			return set
		}
		l.decodeInto(set, options.Format, content)

	case C.RuleSetTypeRemote:
		saved, reason, detail := l.remoteContent(tag)
		if reason != ReasonOK {
			set.Reason = reason
			set.Detail = detail
			return set
		}
		set.UpdatedAt = saved.LastUpdated
		l.decodeInto(set, options.Format, saved.Content)

	default:
		set.Reason = ReasonParseError
		set.Detail = "unknown rule-set type: " + options.Type
	}

	return set
}

// remoteContent pulls a downloaded rule set out of sing-box's cache file.
func (l *Loader) remoteContent(tag string) (*savedBinary, Reason, string) {
	l.cacheOnce.Do(func() {
		if l.cachePath == "" {
			l.cacheErr = ReasonCacheDisabled
			return
		}
		cache, err := OpenCacheFile(l.cachePath, l.cacheID)
		if err != nil {
			l.cacheErr = ReasonCacheUnavailable
			l.cacheDetail = err.Error()
			return
		}
		l.cache = cache
	})

	if l.cacheErr != ReasonOK {
		return nil, l.cacheErr, l.cacheDetail
	}

	saved := l.cache.LoadRuleSet(tag)
	if saved == nil || len(saved.Content) == 0 {
		return nil, ReasonNotCached, ""
	}
	return saved, ReasonOK, ""
}

// decodeInto parses rule-set content and populates the set, or explains why it
// could not.
func (l *Loader) decodeInto(set *Set, format string, content []byte) {
	rules, err := Decode(format, content)
	if err != nil {
		if strings.Contains(err.Error(), unsupportedVersionMarker) {
			set.Reason = ReasonUnsupportedVersion
		} else {
			set.Reason = ReasonParseError
		}
		set.Detail = err.Error()
		return
	}
	l.buildRules(set, rules)
}

// buildRules compiles option rules into matchers.
//
// It deliberately does NOT call route/rule.NewHeadlessRule, even though that
// would be the engine's own code and therefore the most faithful option.
// Importing route/rule pulls the entire sing-box runtime into this binary —
// sing-tun, gvisor, tfo-go, quic-go — for a panel that currently depends on
// `option` and `constant` alone and ships as an OpenWrt ipk where size is a
// real constraint. matcher.go reimplements the matching semantics instead, with
// the upstream line references that pin each rule.
func (l *Loader) buildRules(set *Set, rules []option.HeadlessRule) {
	compiled := make([]matcher, 0, len(rules))
	for i, ruleOptions := range rules {
		rule, err := compile(ruleOptions)
		if err != nil {
			set.Reason = ReasonParseError
			set.Detail = "rule[" + itoa(i) + "]: " + err.Error()
			return
		}
		compiled = append(compiled, rule)
	}
	set.rules = compiled
	set.RuleCount = len(compiled)
	set.Available = true
}
