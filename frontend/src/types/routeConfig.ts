import type { RouteRuleActionTypeName } from '../schemas/routeRuleActionInventory.generated'
export interface RouteRule {
  // Common fields (matching criteria)
  // NOTE: sing-box accepts scalar OR array on the wire for every list-like
  // matcher (e.g. `"inbound": "dns-in"` is equivalent to
  // `"inbound": ["dns-in"]`). The backend round-trips whatever shape lives in
  // config.json. To keep the UI simple, this interface represents the
  // *post-normalization* shape — always coerce raw responses with
  // `normalizeRouteRule()` in RoutingRules.vue before assigning into typed
  // state. Direct consumers of the wire payload must accept scalar | array.
  // From the generated inventory rather than hand-listed. The literal union
  // that used to sit here was missing `direct` — the same drift the schema
  // pattern exists to remove, one copy at a time.
  action?: RouteRuleActionTypeName
  inbound?: string[]
  protocol?: string[]
  network?: string[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
  geosite?: string[] | string
  source_geoip?: string[]
  geoip?: string[] | string
  ip_cidr?: string[]
  source_ip_cidr?: string[]
  source_port?: number[]
  port?: number[]
  // sing-box 1.14+ (see schemas/routeRuleMatcherInventory.ts)
  source_mac_address?: string[]
  source_hostname?: string[]
  rule_set?: string[] | string

  // Action: route
  outbound?: string

  // Action: reject
  method?: 'default' | 'drop'
  no_drop?: boolean

  // Action: route-options
  override_address?: string
  override_port?: number
  network_strategy?: string
  fallback_delay?: string
  udp_disable_domain_unmapping?: boolean
  udp_connect?: boolean
  udp_timeout?: string
  tls_fragment?: boolean
  tls_fragment_fallback_delay?: string
  tls_record_fragment?: string

  // Action: sniff
  sniffer?: string[] | string
  timeout?: string

  // Action: resolve
  server?: string
  strategy?: string
  disable_cache?: boolean
  rewrite_ttl?: number | null
  client_subnet?: string | null

  [key: string]: any
}

export interface RuleSet {
  tag: string
  type: string
  format: string
  path?: string
  url?: string
  download_detour?: string
  update_interval?: string
}
