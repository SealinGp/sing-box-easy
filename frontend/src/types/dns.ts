// DNS Type definitions for sing-box v1.12.12
// Mirrors the structures from github.com/sagernet/sing-box@v1.12.12/option/dns.go

// Domain Strategy types
export type DomainStrategy =
  | 'prefer_ipv4'
  | 'prefer_ipv6'
  | 'ipv4_only'
  | 'ipv6_only'

// DNS Server Types
export type DNSServerType =
  | 'local'      // Local system DNS
  | 'udp'        // UDP DNS
  | 'tcp'        // TCP DNS
  | 'tls'        // DNS over TLS
  | 'https'      // DNS over HTTPS
  | 'h3'         // DNS over HTTP/3 — sing-box spells this 'h3', not 'http3'
  | 'quic'       // DNS over QUIC
  | 'dhcp'       // DHCP DNS
  | 'fakeip'     // FakeIP DNS
  | 'hosts'      // Hosts file DNS
  | 'legacy'     // Legacy (for backward compatibility)

// Base DNS Server Options
export interface BaseDNSServerOptions {
  tag: string
  type?: DNSServerType
}

// Legacy DNS Server Options (backward compatibility)
export interface LegacyDNSServerOptions {
  tag: string
  type?: 'legacy'
  address: string
  address_resolver?: string
  address_strategy?: DomainStrategy
  address_fallback_delay?: string
  strategy?: DomainStrategy
  detour?: string
  client_subnet?: string
}

// Domain Resolver Options
export interface DomainResolverOptions {
  server?: string
  strategy?: DomainStrategy
}

// DNS-specific Dialer Options (used by Local and Remote DNS)
export interface DNSDialerOptions {
  detour?: string
  bind_interface?: string
  inet4_bind_address?: string
  inet6_bind_address?: string
  routing_mark?: number
  reuse_addr?: boolean
  connect_timeout?: string
  tcp_fast_open?: boolean
  tcp_multi_path?: boolean
  udp_fragment?: boolean
  domain_resolver?: DomainResolverOptions
  fallback_delay?: string
}

// Local DNS Server Options
export interface LocalDNSServerOptions extends DNSDialerOptions {
  tag: string
  type: 'local'
}

// DNS Server Address Options
export interface DNSServerAddressOptions {
  server: string
  server_port?: number
}

// Remote DNS Server Options (UDP, TCP, QUIC)
export interface RemoteDNSServerOptions extends DNSDialerOptions, DNSServerAddressOptions {
  tag: string
  type: 'udp' | 'tcp' | 'quic'
}

// TLS Options
export interface TLSOptions {
  enabled?: boolean
  disable_sni?: boolean
  server_name?: string
  insecure?: boolean
  alpn?: string[]
  min_version?: string
  max_version?: string
  cipher_suites?: string[]
  certificate?: string
  certificate_path?: string
  ech?: {
    enabled?: boolean
    pq_signature_schemes_enabled?: boolean
    dynamic_record_sizing_disabled?: boolean
    config?: string[]
  }
  utls?: {
    enabled?: boolean
    fingerprint?: string
  }
  reality?: {
    enabled?: boolean
    public_key?: string
    short_id?: string
  }
}

// Remote TLS DNS Server Options (TLS)
export interface RemoteTLSDNSServerOptions extends DNSDialerOptions, DNSServerAddressOptions {
  tag: string
  type: 'tls'
  tls?: TLSOptions
}

// Remote HTTPS DNS Server Options (HTTPS, HTTP3)
export interface RemoteHTTPSDNSServerOptions extends DNSDialerOptions, DNSServerAddressOptions {
  tag: string
  type: 'https' | 'h3'
  tls?: TLSOptions
  path?: string
  method?: string
  headers?: Record<string, string | string[]>
}

// FakeIP DNS Server Options
export interface FakeIPDNSServerOptions {
  tag: string
  type: 'fakeip'
  inet4_range?: string
  inet6_range?: string
}

// DHCP DNS Server Options
export interface DHCPDNSServerOptions extends DNSDialerOptions {
  tag: string
  type: 'dhcp'
  interface?: string
}

// Hosts DNS Server Options
export interface HostsDNSServerOptions {
  tag: string
  type: 'hosts'
  path?: string[]
  predefined?: Record<string, string | string[]>
}

// Union type for all DNS server options
export type DNSServerOptions =
  | LegacyDNSServerOptions
  | LocalDNSServerOptions
  | RemoteDNSServerOptions
  | RemoteTLSDNSServerOptions
  | RemoteHTTPSDNSServerOptions
  | FakeIPDNSServerOptions
  | DHCPDNSServerOptions
  | HostsDNSServerOptions

/**
 * DNS rule actions, from the generated inventory.
 *
 * This was hand-written as `'route' | 'return' | 'reject'`, which was wrong in
 * both directions: `return` is not a sing-box action at all, and `route-options`
 * and `predefined` were missing. It was the third copy of this list — DNSRules.vue
 * carried a second, correct one — which is exactly the drift the generator exists
 * to remove. Now there is one source, checked against the option structs.
 */
import type { DNSRuleActionTypeName } from '../schemas/dnsRuleActionInventory.generated'

export type DNSRuleActionType = DNSRuleActionTypeName

// DNS Rule Reject Method. sing-box accepts exactly these two (constant/rule.go).
export type DNSRejectMethod = 'default' | 'drop'

// DNS Rule Type
export type DNSRuleType = 'default' | 'logical'

// DNS Rule Logical Mode
export type DNSLogicalMode = 'and' | 'or'

// Base DNS Rule
export interface BaseDNSRule {
  type?: DNSRuleType

  // Rule actions
  action?: DNSRuleActionType
  server?: string              // For 'route' action
  disable_cache?: boolean
  rewrite_ttl?: number
  client_subnet?: string

  // Reject action
  method?: DNSRejectMethod
}

// Default DNS Rule (with conditions)
export interface DefaultDNSRule extends BaseDNSRule {
  type?: 'default'

  // Rule conditions
  inbound?: string[]
  ip_version?: 4 | 6
  query_type?: string[]       // A, AAAA, CNAME, etc.
  network?: string[]          // tcp, udp
  auth_user?: string[]
  protocol?: string[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
  geosite?: string[]
  source_geoip?: string[]
  geoip?: string[]
  source_ip_cidr?: string[]
  source_ip_is_private?: boolean
  ip_cidr?: string[]
  ip_is_private?: boolean
  source_port?: number[]
  source_port_range?: string[]
  port?: number[]
  port_range?: string[]
  process_name?: string[]
  process_path?: string[]
  package_name?: string[]
  user?: string[]
  user_id?: number[]
  clash_mode?: string
  wifi_ssid?: string[]
  wifi_bssid?: string[]
  rule_set?: string[]
  rule_set_ip_cidr_match_source?: boolean
  invert?: boolean
  outbound?: string[]
}

// Logical DNS Rule (combination of rules)
export interface LogicalDNSRule extends BaseDNSRule {
  type: 'logical'
  mode: DNSLogicalMode
  rules: DNSRule[]
}

// Union type for all DNS rules
export type DNSRule = DefaultDNSRule | LogicalDNSRule

// DNS Client Options
export interface DNSClientOptions {
  strategy?: DomainStrategy
  disable_cache?: boolean
  disable_expire?: boolean
  independent_cache?: boolean
  cache_capacity?: number
  client_subnet?: string
}

// Legacy FakeIP Options (for backward compatibility)
export interface LegacyFakeIPOptions {
  enabled: boolean
  inet4_range?: string
  inet6_range?: string
}

// Raw DNS Options (core fields)
export interface RawDNSOptions extends DNSClientOptions {
  servers?: DNSServerOptions[]
  rules?: DNSRule[]
  final?: string
  reverse_mapping?: boolean
}

// DNS Options (complete)
export interface DNSOptions extends RawDNSOptions {
  // Legacy FakeIP support (deprecated, use fakeip server type instead)
  fakeip?: LegacyFakeIPOptions
}

// Type guards for DNS server options
export function isLegacyDNSServer(server: DNSServerOptions): server is LegacyDNSServerOptions {
  return !server.type || server.type === 'legacy'
}

export function isLocalDNSServer(server: DNSServerOptions): server is LocalDNSServerOptions {
  return server.type === 'local'
}

export function isRemoteDNSServer(server: DNSServerOptions): server is RemoteDNSServerOptions {
  return server.type === 'udp' || server.type === 'tcp' || server.type === 'quic'
}

export function isTLSDNSServer(server: DNSServerOptions): server is RemoteTLSDNSServerOptions {
  return server.type === 'tls'
}

export function isHTTPSDNSServer(server: DNSServerOptions): server is RemoteHTTPSDNSServerOptions {
  return server.type === 'https' || server.type === 'h3'
}

export function isFakeIPDNSServer(server: DNSServerOptions): server is FakeIPDNSServerOptions {
  return server.type === 'fakeip'
}

export function isDHCPDNSServer(server: DNSServerOptions): server is DHCPDNSServerOptions {
  return server.type === 'dhcp'
}

export function isHostsDNSServer(server: DNSServerOptions): server is HostsDNSServerOptions {
  return server.type === 'hosts'
}

// Type guards for DNS rules
export function isDefaultDNSRule(rule: DNSRule): rule is DefaultDNSRule {
  return !rule.type || rule.type === 'default'
}

export function isLogicalDNSRule(rule: DNSRule): rule is LogicalDNSRule {
  return rule.type === 'logical'
}
