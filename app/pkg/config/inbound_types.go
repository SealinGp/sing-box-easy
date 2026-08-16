package config

import C "github.com/sagernet/sing-box/constant"

// Regenerates the frontend's inbound field inventory from the option structs
// the registry below constructs. Run after bumping the sing-box dependency.
//
//go:generate go run ../../../cmd/gen-inbound-schema

// InboundTypes lists every inbound type CreateInboundOptions can construct.
//
// This exists so the schema generator (cmd/gen-inbound-schema) and the registry
// cannot disagree about which types the panel supports. The generator reflects
// over exactly these, and TestInboundTypesAreRegistered asserts every entry is
// actually constructible — so an entry added here without a matching case in
// CreateInboundOptions fails the build rather than emitting an empty field set
// to the frontend.
//
// The reverse direction (a case in CreateInboundOptions missing from this list)
// is NOT caught automatically: a Go type switch cannot be enumerated at runtime.
// Adding a type means touching both places. That was the situation that let
// "anytls" ship in the registry while the frontend never offered it.
//
// Order is the order fields appear in the generated TypeScript, so keep related
// types adjacent — it is the diff people read when sing-box is upgraded.
var InboundTypes = []string{
	// Local / transparent proxies
	C.TypeTun,
	C.TypeRedirect,
	C.TypeTProxy,
	C.TypeDirect,

	// Plain proxy servers
	C.TypeMixed,
	C.TypeHTTP,
	C.TypeSOCKS,

	// Encrypted proxy servers
	C.TypeShadowsocks,
	C.TypeVMess,
	C.TypeVLESS,
	C.TypeTrojan,
	C.TypeNaive,
	C.TypeHysteria,
	C.TypeHysteria2,
	C.TypeTUIC,
	C.TypeShadowTLS,
	C.TypeAnyTLS,
}
