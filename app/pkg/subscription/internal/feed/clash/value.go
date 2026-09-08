package clash

import (
	"fmt"
	"strconv"
	"strings"
)

// proxy is one entry of a Clash config's `proxies:` list, kept as a raw map.
//
// It is deliberately NOT a typed struct: Clash/Mihomo carries a different key
// set per proxy type, providers add vendor keys freely, and yaml.v3 would
// either need one struct per type or a lossy union. The accessors below do the
// narrowing, and anything unrecognized is simply ignored rather than failing
// the whole subscription.
type proxy map[string]any

// str returns the first key present as a non-empty string.
func (p proxy) str(keys ...string) string {
	for _, k := range keys {
		v, ok := p[k]
		if !ok || v == nil {
			continue
		}
		if s := scalarString(v); s != "" {
			return s
		}
	}
	return ""
}

// scalarString renders one YAML scalar as a string. Values are not always
// strings on the wire — a numeric password (`password: 123456`) decodes as an
// int — so scalars are stringified rather than dropped.
func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(t)
	}
	return ""
}

// boolean reports whether the first key present is truthy. Clash writes real
// YAML booleans, but "true"/"1" strings show up in hand-edited feeds.
func (p proxy) boolean(keys ...string) bool {
	for _, k := range keys {
		v, ok := p[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case bool:
			return t
		case string:
			switch strings.ToLower(strings.TrimSpace(t)) {
			case "1", "true", "yes":
				return true
			}
		case int:
			return t != 0
		case float64:
			return t != 0
		}
	}
	return false
}

// number returns the first key present as an int.
func (p proxy) number(keys ...string) (int, bool) {
	for _, k := range keys {
		v, ok := p[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case int:
			return t, true
		case int64:
			return int(t), true
		case float64:
			return int(t), true
		case string:
			if n, err := strconv.Atoi(strings.TrimSpace(t)); err == nil {
				return n, true
			}
		}
	}
	return 0, false
}

// port returns the `port` field validated into sing-box's uint16.
func (p proxy) port(keys ...string) (uint16, error) {
	n, ok := p.number(keys...)
	if !ok {
		return 0, fmt.Errorf("missing port")
	}
	if n < 1 || n > 65535 {
		return 0, fmt.Errorf("port out of range: %d", n)
	}
	return uint16(n), nil
}

// sub returns a nested mapping (ws-opts, grpc-opts, reality-opts, …).
// A missing key yields an empty proxy, so callers can chain without nil checks.
func (p proxy) sub(key string) proxy {
	v, ok := p[key]
	if !ok || v == nil {
		return proxy{}
	}
	if m, ok := v.(map[string]any); ok {
		return proxy(m)
	}
	return proxy{}
}

// strList returns a key as a string slice, accepting both a YAML list and a
// comma-separated scalar ("alpn: h2,http/1.1").
func (p proxy) strList(keys ...string) []string {
	for _, k := range keys {
		v, ok := p[k]
		if !ok || v == nil {
			continue
		}
		switch t := v.(type) {
		case []any:
			out := make([]string, 0, len(t))
			for _, item := range t {
				if s := scalarString(item); s != "" {
					out = append(out, s)
				}
			}
			if len(out) > 0 {
				return out
			}
		case string:
			out := make([]string, 0, 4)
			for _, part := range strings.Split(t, ",") {
				if part = strings.TrimSpace(part); part != "" {
					out = append(out, part)
				}
			}
			if len(out) > 0 {
				return out
			}
		}
	}
	return nil
}

// headerHost digs the Host header out of a transport's `headers:` mapping.
// The key's capitalization is provider-dependent, so match case-insensitively.
func (p proxy) headerHost() string {
	headers := p.sub("headers")
	for k := range headers {
		if strings.EqualFold(k, "host") {
			return headers.str(k)
		}
	}
	return ""
}
