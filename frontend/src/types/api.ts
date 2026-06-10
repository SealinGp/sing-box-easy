// API Response Types
import type { DNSOptions } from './dns'
import type { Inbound as InboundType } from './inbound'
import type { Outbound } from './outbound'

// Business response codes (matches backend Code type)
export const Code = {
  Success: 0,         // Operation successful
  BadRequest: 1,      // Invalid request parameters
  NotFound: 2,        // Resource not found
  InternalError: 3,   // Internal server error
  ValidationError: 4, // Validation failed
  Conflict: 5,        // Resource conflict (e.g., duplicate)
  Unauthorized: 6,    // Unauthorized access
  Forbidden: 7,       // Forbidden operation
  ServiceError: 8,    // External service error
  ConfigError: 9,     // Configuration error
  OperationFailed: 10, // Operation failed
} as const

export type Code = (typeof Code)[keyof typeof Code]

// Standard response structure from backend
export interface BasicResponse<T = unknown> {
  code: Code
  data: T
  msg: string
}

// API Error class for handling non-success responses
export class ApiError extends Error {
  code: Code

  constructor(code: Code, message: string) {
    super(message)
    this.code = code
    this.name = 'ApiError'
  }
}

export interface InitState {
  initialized: boolean
  steps: {
    sing_box_installed: boolean
    config_generated: boolean
    dashboard_installed: boolean
  }
  sing_box_version?: string
  init_time?: string
}

export interface InstallTask {
  id?: string
  task_id?: string
  status: 'running' | 'completed' | 'failed'
  message: string
  error?: string
}

export interface DashboardTask {
  id: string
  status: 'running' | 'completed' | 'failed'
  message: string
  error?: string
}

export interface ServiceStatus {
  status: 'running' | 'stopped' | 'unknown'
  running?: boolean
  pid?: number
  uptime?: string
  // Unix seconds when the process started; 0/undefined when unknown or stopped.
  started_at?: number
}

// A bounded chunk of recent sing-box logs returned by GET /service/logs.
export interface ServiceLogs {
  lines: string[]
  // Opaque journald cursor; feed back on the next poll for incremental fetch.
  cursor: string
  // Where the lines came from: journald (systemd), file (log.output), or none.
  source: 'journald' | 'file' | 'none'
}

// Metadata for a stored historical config version (no content).
export interface ConfigVersion {
  id: number
  size: number
  created_at: string
}

// Application settings exposed to the frontend.
export interface AppSettings {
  settings: Record<string, string>
  config_versions_keep: number
}

// Re-export Outbound types from outbound.ts (DialerOptions is from shared.ts)
export type {
  Outbound,
  OutboundType,
  BaseOutbound,
  ServerOptions,
  DirectOutboundOptions,
  BlockOutboundOptions,
  DNSOutboundOptions,
  SocksOutboundOptions,
  HTTPOutboundOptions,
  ShadowsocksOutboundOptions,
  VMessOutboundOptions,
  TrojanOutboundOptions,
  WireGuardOutboundOptions,
  HysteriaOutboundOptions,
  Hysteria2OutboundOptions,
  VLESSOutboundOptions,
  TUICOutboundOptions,
  ShadowTLSOutboundOptions,
  TorOutboundOptions,
  SSHOutboundOptions,
  SelectorOutboundOptions,
  URLTestOutboundOptions,
  OutboundTLSOptions,
  V2RayTransportOptions,
  MultiplexOptions,
} from './outbound'

// Re-export DialerOptions from shared.ts
export type { DialerOptions } from './shared'

export interface OutboundGroup {
  tag: string
  type: 'selector' | 'urltest' | 'direct' | 'block'
  outbounds?: string[]
  [key: string]: any
}

export interface ParsedNode {
  protocol: string
  outbound: Outbound
}

export interface Subscription {
  id: string
  name: string
  url: string
  update_interval: string
  enabled: boolean
  // Whether the cron auto-updater refreshes this subscription. The backend
  // only auto-refreshes when BOTH enabled and auto_update are true.
  auto_update?: boolean
  last_update?: string
  node_count?: number
}

// Response payload for POST /subscriptions/:id/update.
// Backend computes a 3-way diff (add/update/delete) and returns both the
// raw tags/keys and pre-computed counts.
export interface SubscriptionUpdateResult {
  message: string
  id: string
  added_tags: string[] | null
  updated_tags: string[] | null
  deleted_keys: string[] | null
  added: number
  updated: number
  deleted: number
}

// Re-export DNS types from dns.ts
export type {
  DNSServerOptions as DNSServer,
  DNSRule,
  DNSOptions as DNS,
  DomainStrategy,
  DNSServerType,
  LegacyDNSServerOptions,
  LocalDNSServerOptions,
  RemoteDNSServerOptions,
  RemoteTLSDNSServerOptions,
  RemoteHTTPSDNSServerOptions,
  FakeIPDNSServerOptions,
  DHCPDNSServerOptions,
  HostsDNSServerOptions,
  DefaultDNSRule,
  LogicalDNSRule,
  DNSRuleActionType,
  DNSRuleType,
  DNSLogicalMode,
  TLSOptions,
} from './dns'

// Re-export Inbound types from inbound.ts
export type {
  InboundType,
  BaseInbound,
  ListenOptions,
  TunInboundOptions,
  RedirectInboundOptions,
  TProxyInboundOptions,
  DirectInboundOptions,
  SocksInboundOptions,
  HTTPMixedInboundOptions,
  ShadowsocksInboundOptions,
  VMessInboundOptions,
  TrojanInboundOptions,
  NaiveInboundOptions,
  HysteriaInboundOptions,
  Hysteria2InboundOptions,
  VLESSInboundOptions,
  TUICInboundOptions,
  ShadowTLSInboundOptions,
  InboundTLSOptions,
} from './inbound'

export interface RouteRule {
  // Common fields (matching criteria)
  // NOTE: sing-box accepts scalar OR array on the wire for every list-like
  // matcher (e.g. `"inbound": "dns-in"` is equivalent to
  // `"inbound": ["dns-in"]`). The backend round-trips whatever shape lives in
  // config.json. To keep the UI simple, this interface represents the
  // *post-normalization* shape — always coerce raw responses with
  // `normalizeRouteRule()` in RoutingRules.vue before assigning into typed
  // state. Direct consumers of the wire payload must accept scalar | array.
  action?: 'route' | 'reject' | 'route-options' | 'sniff' | 'resolve' | 'hijack-dns'
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
  url?: string
  download_detour?: string
  update_interval?: string
}

// Re-export Inbound types from inbound.ts
export type { Inbound } from './inbound'

export interface LogConfig {
  disabled?: boolean
  level?: string
  output?: string
  timestamp?: boolean
}

export interface ClashAPI {
  external_controller?: string
  external_ui?: string
  external_ui_download_url?: string
  external_ui_download_detour?: string
  secret?: string
  default_mode?: string
  access_control_allow_origin?: string[]
  access_control_allow_private_network?: boolean
}

export interface CacheFile {
  enabled?: boolean
  path?: string
  cache_id?: string
  store_fakeip?: boolean
  store_rdrc?: boolean
  rdrc_timeout?: string
}

export interface V2RayAPI {
  listen?: string
  stats?: V2RayStatsService
}

export interface V2RayStatsService {
  enabled?: boolean
  inbounds?: string[]
  outbounds?: string[]
  users?: string[]
}

export interface SingBoxConfig {
  log?: LogConfig
  experimental?: {
    clash_api?: ClashAPI
    cache_file?: CacheFile
    v2ray_api?: V2RayAPI
  }
  dns?: DNSOptions
  inbounds?: InboundType[]
  outbounds?: Outbound[]
  route?: {
    rules?: RouteRule[]
    rule_set?: RuleSet[]
    final?: string
    auto_detect_interface?: boolean
    override_android_vpn?: boolean
  }
}

export interface DefaultRuleSet {
  tag: string
  description: string
  type: string
  format: string
  url: string
}
