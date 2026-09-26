package config

import C "github.com/sagernet/sing-box/constant"

// DNSTypes lists every DNS server type CreateDNSOptions can construct.
//
// Same contract as InboundTypes: the schema generator reflects over exactly
// these, and TestDNSTypesAreRegistered fails the build if one is listed here
// without a matching case.
//
// "h3" — NOT "http3". sing-box's constant is C.DNSTypeHTTP3 = "h3" and its
// transport registers under that name. The panel used to spell it "http3",
// which broke both directions: a valid "h3" config could not be parsed, and a
// config the panel wrote with "http3" could not be started.
//
// The legacy transport (an absent type, or "legacy") is deliberately ABSENT.
// sing-box parses it into LegacyDNSServerOptions and immediately upgrades it to
// a typed server, so it never reaches the registry and can never be the type of
// a server the panel is asked to build. Nothing should be creating one.
//
// Order is the order fields appear in the generated TypeScript — keep related
// types adjacent, it is the diff people read on a sing-box upgrade.
var DNSTypes = []string{
	// Remote transports, plain
	C.DNSTypeUDP,
	C.DNSTypeTCP,

	// Remote transports, encrypted
	C.DNSTypeTLS,
	C.DNSTypeQUIC,
	C.DNSTypeHTTPS,
	C.DNSTypeHTTP3,

	// Resolved locally, no upstream address
	C.DNSTypeLocal,
	C.DNSTypeHosts,
	C.DNSTypeFakeIP,
	C.DNSTypeDHCP,

	// Requires a sing-box built with the tailscale tag to run, but always
	// parseable.
	C.DNSTypeTailscale,
}

// IsKnownDNSType reports whether the type can be constructed by the registry.
// Used by request validation so an unknown type is a field-level error rather
// than a `sing-box check` failure pointing at the file.
func IsKnownDNSType(dnsType string) bool {
	for _, known := range DNSTypes {
		if known == dnsType {
			return true
		}
	}
	return false
}
