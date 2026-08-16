package config

import C "github.com/sagernet/sing-box/constant"

// OutboundTypes lists every outbound type CreateOutboundOptions can construct.
//
// Same contract as InboundTypes and DNSTypes: the schema generator reflects
// over exactly these, and TestOutboundTypesAreRegistered fails the build if one
// is listed without a matching case.
//
// Three of these were registered on the backend and unreachable from the UI —
// `anytls`, `dns` and `shadowsocksr` — because the form kept its own hardcoded
// list. That is the same drift that hid `anytls` on the inbound side.
//
// Two carry a health warning that the generated inventory now records from
// sing-box's own deprecation table, so the form can gate them against the
// INSTALLED binary rather than the pinned library:
//
//   - `dns` is deprecated since 1.11 and rejected at config decode by 1.13;
//     the route action `hijack-dns` replaces it.
//   - `wireguard` is deprecated since 1.11 and fails at outbound init on 1.13;
//     the WireGuard *endpoint* replaces it.
//
// `block` is NOT in that boat despite reading like a sibling of `dns`:
// sing-box reports the deprecation for `dns` only, and 1.13.11 still accepts a
// block outbound. Verified with `sing-box check`.
var OutboundTypes = []string{
	// Terminal behaviours, no upstream server
	C.TypeDirect,
	C.TypeBlock,
	C.TypeDNS,

	// Plain proxies
	C.TypeSOCKS,
	C.TypeHTTP,

	// Encrypted proxies
	C.TypeShadowsocks,
	C.TypeVMess,
	C.TypeVLESS,
	C.TypeTrojan,
	C.TypeHysteria,
	C.TypeHysteria2,
	C.TypeTUIC,
	C.TypeShadowTLS,
	C.TypeAnyTLS,
	C.TypeShadowsocksR,

	// Tunnels and transports
	C.TypeWireGuard,
	C.TypeSSH,
	C.TypeTor,

	// Groups — these reference other outbounds by tag rather than dialing
	C.TypeSelector,
	C.TypeURLTest,
}

// OutboundGroupTypes are the types whose options carry an `outbounds` list of
// other outbound tags instead of a server to dial.
//
// The frontend has needed this three separate times (a form predicate, a badge
// colour, and a table filter) and the backend a fourth (IsEndpointType in
// group_classify.go). Exported here so the schema generator and the group
// handlers can share one definition.
var OutboundGroupTypes = []string{C.TypeSelector, C.TypeURLTest}

// IsOutboundGroupType reports whether the type groups other outbounds.
func IsOutboundGroupType(outboundType string) bool {
	for _, known := range OutboundGroupTypes {
		if known == outboundType {
			return true
		}
	}
	return false
}

// IsKnownOutboundType reports whether the type can be constructed by the
// registry. Used by request validation so an unknown type is a field-level
// error rather than a decode failure from deep inside sing-box.
func IsKnownOutboundType(outboundType string) bool {
	for _, known := range OutboundTypes {
		if known == outboundType {
			return true
		}
	}
	return false
}
