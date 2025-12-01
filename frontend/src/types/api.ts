// API Response Types
import type { DNSOptions } from './dns'
import type { Inbound as InboundType } from './inbound'

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
  pid?: number
  uptime?: string
}

export interface Outbound {
  tag: string
  type: string
  server?: string
  server_port?: number
  method?: string
  password?: string
  uuid?: string
  alter_id?: number
  security?: string
  network?: string
  tls?: any
  [key: string]: any
}

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
  update_interval: number
  enabled: boolean
  last_update?: string
  node_count?: number
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
  V2RayTransportOptions,
  MultiplexOptions,
} from './inbound'

export interface RouteRule {
  inbound?: string[]
  protocol?: string[]
  network?: string[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
  geosite?: string[]
  source_geoip?: string[]
  geoip?: string[]
  ip_cidr?: string[]
  source_ip_cidr?: string[]
  source_port?: number[]
  port?: number[]
  outbound: string
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
}

export interface CacheFile {
  enabled?: boolean
  path?: string
  cache_id?: string
  store_fakeip?: boolean
  store_rdrc?: boolean
  rdrc_timeout?: string
}

export interface SingBoxConfig {
  log?: LogConfig
  experimental?: {
    clash_api?: ClashAPI
    cache_file?: CacheFile
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
