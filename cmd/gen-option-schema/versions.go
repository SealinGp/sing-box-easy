package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// Version-aware deprecation, read from sing-box's own table.
//
// WHY THIS MATTERS MORE THAN IT LOOKS
// ───────────────────────────────────
// The generated inventory describes the sing-box LIBRARY this repo pins. The
// binary the operator actually runs is a separate thing, and it is routinely
// newer — the machine this was written on pins 1.12.12 and runs 1.13.11.
//
// For fields that a newer sing-box ADDED, the mismatch is harmless: the form
// does not offer a knob, which is a missing feature. For fields and types a
// newer sing-box REMOVED, it is not harmless — the form offers something the
// binary rejects. Probed against 1.13.11:
//
//	sniff (and the rest of the legacy inbound fields)  -> config decode fails
//	direct outbound override_address / override_port   -> config decode fails
//	tun inet4_route_address                            -> inbound init fails
//	dns outbound                                       -> config decode fails
//	wireguard outbound                                 -> outbound init fails
//
// Those first two are especially bad, because the inbound form offers the
// sniff family in its own "deprecated" row — inviting a click that makes the
// whole config unloadable. (The save is caught by `sing-box check` before
// anything is written, so it fails loudly rather than corrupting; it is still
// an opaque error for something the UI suggested.)
//
// sing-box records all of this itself, in experimental/deprecated/constants.go,
// with the version that deprecated each item and the version that removes it.
// Parsing that is strictly better than a hand-list: it is versioned, it covers
// TYPES as well as fields, and it updates when the dependency does.
//
// WHAT STILL HAS TO BE HAND-MAINTAINED
// ────────────────────────────────────
// The table's entries are identifiers ("wireguard-outbound"), not field paths.
// Mapping an entry onto the inventory is the `deprecationTargets` table below,
// and it was built by finding each `deprecated.Report(...)` call site in the
// sing-box source rather than by guessing from the names. Recheck those call
// sites on a dependency bump:
//
//	grep -rn 'deprecated\.Option' $(go env GOMODCACHE)/github.com/sagernet/sing-box@<ver>

// deprecationNote is one row of sing-box's deprecation table.
type deprecationNote struct {
	Name              string
	DeprecatedVersion string
	ScheduledVersion  string
	MigrationLink     string
}

// deprecationTable parses experimental/deprecated/constants.go.
//
// The entries are package-level `var OptionX = Note{...}` composite literals,
// so this walks the AST rather than pattern-matching text: a reordered or
// reformatted upstream file should still parse.
func deprecationTable() (map[string]deprecationNote, error) {
	dir, err := singBoxSubdir("experimental/deprecated")
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", dir, err)
	}

	out := map[string]deprecationNote{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				lit, ok := n.(*ast.CompositeLit)
				if !ok {
					return true
				}

				var note deprecationNote
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					key, ok := kv.Key.(*ast.Ident)
					if !ok {
						continue
					}
					value, ok := kv.Value.(*ast.BasicLit)
					if !ok || value.Kind != token.STRING {
						continue
					}
					unquoted := strings.Trim(value.Value, `"`)

					switch key.Name {
					case "Name":
						note.Name = unquoted
					case "DeprecatedVersion":
						note.DeprecatedVersion = unquoted
					case "ScheduledVersion":
						note.ScheduledVersion = unquoted
					case "MigrationLink":
						note.MigrationLink = unquoted
					}
				}

				if note.Name != "" && note.DeprecatedVersion != "" {
					out[note.Name] = note
				}
				return true
			})
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no deprecation notes found in %s; upstream layout probably changed", dir)
	}
	return out, nil
}

// deprecationTarget says which part of which domain a table entry covers.
type deprecationTarget struct {
	// Domain name, matching domain.Name.
	Domain string
	// Whole types this entry retires. Keyed by inventory type name.
	Types []string
	// Fields this entry retires, as type -> field names. A type of "*" applies
	// the fields to every type in the domain that has them.
	Fields map[string][]string
}

// deprecationTargets maps sing-box's deprecation identifiers onto the
// inventory.
//
// Built from the `deprecated.Report(...)` call sites, not from the names —
// several are misleading. "special-outbounds" reads like it covers both `block`
// and `dns`, but option/outbound.go:41-43 reports it for `dns` ONLY, and
// sing-box 1.13.11 does still accept a `block` outbound. Verified by running
// `sing-box check` against each.
var deprecationTargets = map[string][]deprecationTarget{
	// option/outbound.go:55 — fires when an inbound's embedded InboundOptions
	// is non-zero. `detour` lives in that struct on 1.12 but survived into
	// 1.13, and 1.13.11 accepts it, so it is deliberately NOT listed here.
	"inbound-options": {{
		Domain: "Inbound",
		Fields: map[string][]string{"*": {
			"sniff",
			"sniff_override_destination",
			"sniff_timeout",
			"domain_strategy",
			"udp_disable_domain_unmapping",
		}},
	}},

	// protocol/tun/inbound.go:125
	"tun-address-x": {{
		Domain: "Inbound",
		Fields: map[string][]string{"tun": {
			"inet4_address",
			"inet6_address",
			"inet4_route_address",
			"inet6_route_address",
			"inet4_route_exclude_address",
			"inet6_route_exclude_address",
		}},
	}},

	// protocol/tun/inbound.go:130
	"tun-gso": {{
		Domain: "Inbound",
		Fields: map[string][]string{"tun": {"gso"}},
	}},

	// option/outbound.go:43 — `dns` only; `block` still works on 1.13.
	"special-outbounds": {{
		Domain: "Outbound",
		Types:  []string{"dns"},
	}},

	// protocol/wireguard/outbound.go:38
	"wireguard-outbound": {{
		Domain: "Outbound",
		Types:  []string{"wireguard"},
	}},

	// option/direct.go:35 — the outbound only. Inbound `direct` has its own
	// struct and is not reported.
	"destination-override-fields": {{
		Domain: "Outbound",
		Fields: map[string][]string{"direct": {"override_address", "override_port"}},
	}},

	// common/dialer/dialer.go:86,123 — applies wherever dial fields are
	// embedded, which is every outbound and every remote DNS transport.
	"legacy-domain-strategy-options": {
		{Domain: "Outbound", Fields: map[string][]string{"*": {"domain_strategy"}}},
		{Domain: "DNSServer", Fields: map[string][]string{"*": {"domain_strategy"}}},
		// The `direct` route action is DialerOptions too, so it inherits the
		// same retired field. The go/ast pass does not catch this one on its
		// own — the outbound and DNS entries above are also carried purely by
		// this table — so a new domain embedding DialerOptions must be added
		// here explicitly or it silently offers a field scheduled for removal.
		{Domain: "RouteRuleAction", Fields: map[string][]string{"direct": {"domain_strategy"}}},
	},

	// route/rule/rule_default.go:125-133 and rule_dns.go:116 — these two are
	// NOT reported through deprecated.Report at all. They are a hard error at
	// rule construction:
	//
	//	geosite database is deprecated in sing-box 1.8.0 and removed in sing-box 1.12.0
	//
	// Removed in 1.12.0, which is at or below the pinned library AND below every
	// binary an operator can install, so isRetired withholds them everywhere.
	// That is the intent: the route form used to promote `geosite` and `geoip`
	// as curated dropdowns with 12 and 7 options, and every one of those values
	// produced a rule sing-box refuses to start. Verified with `sing-box check`
	// against 1.13.11.
	//
	// The replacement is a rule set — which this repo's own templates and init
	// wizard already use.
	"geosite": {
		{Domain: "RouteRuleMatcher", Fields: map[string][]string{"*": {"geosite"}}},
	},
	"geoip": {
		{Domain: "RouteRuleMatcher", Fields: map[string][]string{"*": {"geoip", "source_geoip"}}},
	},

	// route/rule/rule_default.go:255-257 — the pre-1.10 spelling, renamed to
	// rule_set_ip_cidr_match_source. Still decodes on 1.13.11, but reports.
	"bad-match-source": {
		{Domain: "RouteRuleMatcher", Fields: map[string][]string{"*": {"rule_set_ipcidr_match_source"}}},
	},
}

// singBoxSubdir resolves a path inside the pinned sing-box module.
func singBoxSubdir(sub string) (string, error) {
	dir, err := moduleDir(singBoxModule)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, filepath.FromSlash(sub)), nil
}

// markVersioned stamps fields with the versions from sing-box's deprecation
// table, and marks them deprecated even when neither the Go doc comments nor
// the docs-only list caught them.
//
// A field can be flagged by any of three sources — a `// Deprecated:` comment,
// the domain's docs-only list, or this table. Only this one carries versions,
// which is what lets the UI compare against the binary the operator actually
// runs.
func markVersioned(d domain, optionType string, fields []field, table map[string]deprecationNote) {
	for name, targets := range deprecationTargets {
		note, ok := table[name]
		if !ok {
			// Upstream dropped the entry, usually because the removal already
			// happened. Nothing to stamp; the field is either gone from the
			// struct or still marked by one of the other two sources.
			continue
		}

		for _, target := range targets {
			if target.Domain != d.Name {
				continue
			}

			for _, wanted := range target.Fields[optionType] {
				stampField(fields, wanted, note)
			}
			for _, wanted := range target.Fields["*"] {
				stampField(fields, wanted, note)
			}
		}
	}
}

func stampField(fields []field, name string, note deprecationNote) {
	for i := range fields {
		if fields[i].Name != name {
			continue
		}
		fields[i].Deprecated = true
		fields[i].Since = note.DeprecatedVersion
		fields[i].Removed = note.ScheduledVersion
	}
}

// typeNotes returns the per-type deprecation for a domain, so the type selector
// can flag or block a whole type rather than only its fields. `dns` and
// `wireguard` outbounds are the cases that matter: both parse fine and then
// fail, one at config decode and one at outbound init.
func typeNotes(d domain, table map[string]deprecationNote) map[string]deprecationNote {
	out := map[string]deprecationNote{}

	for name, targets := range deprecationTargets {
		note, ok := table[name]
		if !ok {
			continue
		}
		for _, target := range targets {
			if target.Domain != d.Name {
				continue
			}
			for _, optionType := range target.Types {
				out[optionType] = note
			}
		}
	}

	return out
}
