// Package clash imports a Clash / Mihomo YAML profile as subscription nodes.
//
// Why this exists: the canonical subscription format is a base64-encoded list
// of proxy URIs, and that is what app/pkg/sublink parses. A large share of
// providers, however, serve a full Clash profile instead — either because the
// URL is a Clash-only endpoint or because the panel content-negotiates on the
// User-Agent. Such a body is not base64, so the URI parser cannot even begin;
// only the `proxies:` list matters here and everything else in the profile
// (rules, proxy-groups, DNS) is deliberately ignored, since sing-box-easy owns
// routing itself.
package clash

import (
	"errors"
	"fmt"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"gopkg.in/yaml.v3"
)

// document is the only part of a Clash profile this importer reads.
type document struct {
	Proxies []map[string]any `yaml:"proxies"`
}

// Skipped records one proxy that could not be converted, so the caller can
// report it instead of silently returning a short node list.
type Skipped struct {
	Name   string
	Type   string
	Reason string
}

func (s Skipped) String() string {
	switch {
	case s.Name == "" && s.Type == "":
		return s.Reason
	case s.Name == "":
		return fmt.Sprintf("(%s): %s", s.Type, s.Reason)
	default:
		return fmt.Sprintf("%s (%s): %s", s.Name, s.Type, s.Reason)
	}
}

// decodeProfile reads the `proxies:` list, tolerating entries that are not
// mappings at all.
//
// yaml.v3 reports a shape mismatch inside a list as a *yaml.TypeError, which
// it returns AFTER decoding every element it could — so a single stray entry
// (a bare string, a null, a mis-indented row, a placeholder some panels emit)
// still leaves the valid proxies populated. Treating that error as fatal threw
// away a whole subscription over one bad row, which is exactly what this
// importer promises not to do. A genuine syntax error is still fatal: nothing
// was decoded, so there is nothing to salvage.
func decodeProfile(body []byte) (document, []Skipped, error) {
	var doc document
	err := yaml.Unmarshal(body, &doc)
	if err == nil {
		return doc, nil, nil
	}

	var typeErr *yaml.TypeError
	if !errors.As(err, &typeErr) {
		return document{}, nil, fmt.Errorf("clash profile is not valid yaml: %w", err)
	}

	skipped := make([]Skipped, 0, len(typeErr.Errors))
	for _, msg := range typeErr.Errors {
		skipped = append(skipped, Skipped{Reason: msg})
	}
	return doc, skipped, nil
}

// Detect reports whether a subscription body looks like a Clash profile.
//
// It parses rather than pattern-matches on "proxies:": the string also occurs
// in comments and in `proxy-providers`, and a false positive here would turn a
// clear "not base64" error into a confusing "no proxies found".
func Detect(body []byte) bool {
	doc, _, err := decodeProfile(body)
	if err != nil {
		return false
	}
	return len(doc.Proxies) > 0
}

// Parse converts the `proxies:` list into subscription nodes. Proxies of a
// type sing-box has no outbound for are returned in the skipped list rather
// than failing the import — one exotic entry must not cost the user the other
// hundred.
func Parse(body []byte) ([]*node.SubNode, []Skipped, error) {
	doc, skipped, err := decodeProfile(body)
	if err != nil {
		return nil, nil, err
	}
	if len(doc.Proxies) == 0 {
		return nil, skipped, fmt.Errorf("clash profile has no proxies")
	}

	nodes := make([]*node.SubNode, 0, len(doc.Proxies))
	taken := make(map[string]struct{}, len(doc.Proxies))

	for _, raw := range doc.Proxies {
		p := proxy(raw)
		name, typ := p.str("name"), p.str("type")

		converted, err := p.convert(typ)
		if err != nil {
			skipped = append(skipped, Skipped{Name: name, Type: typ, Reason: err.Error()})
			continue
		}
		if converted.Tag == "" {
			// Port included: two anonymous proxies on one host are a normal
			// port-hopping layout, and they must not collide below.
			converted.Tag = fmt.Sprintf("%s-%s-%s", typ, p.str("server"), p.str("port"))
		}
		// An outbound tag is a primary key to sing-box: a duplicate makes the
		// whole config fail `sing-box check`, and the resulting rollback names
		// the tag but not the two feed entries that collided. Providers do ship
		// repeated names, so uniqueness is settled here instead.
		converted.Tag = uniqueTag(taken, converted.Tag)
		nodes = append(nodes, converted)
	}

	if len(nodes) == 0 {
		return nil, skipped, fmt.Errorf("clash profile has %d proxies but none are supported", len(doc.Proxies))
	}
	return nodes, skipped, nil
}

// uniqueTag returns tag, suffixed if it is already taken, and records it.
func uniqueTag(taken map[string]struct{}, tag string) string {
	candidate := tag
	for i := 2; ; i++ {
		if _, clash := taken[candidate]; !clash {
			taken[candidate] = struct{}{}
			return candidate
		}
		candidate = fmt.Sprintf("%s-%d", tag, i)
	}
}

// convert dispatches on the Clash proxy type. The set mirrors the URI parsers
// registered in sublink/protocol, plus the types Clash spells differently
// ("ss", "socks5" → sing-box "shadowsocks", …).
func (p proxy) convert(typ string) (*node.SubNode, error) {
	switch typ {
	case "ss", "shadowsocks":
		return p.toShadowsocks()
	case "vmess":
		return p.toVMess()
	case "trojan":
		return p.toTrojan()
	case "vless":
		return p.toVLESS()
	case "hysteria2", "hy2":
		return p.toHysteria2()
	case "anytls":
		return p.toAnyTLS()
	case "":
		return nil, fmt.Errorf("proxy has no type")
	default:
		return nil, fmt.Errorf("unsupported proxy type %q", typ)
	}
}
