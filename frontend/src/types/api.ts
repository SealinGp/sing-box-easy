// API Response Types

export interface InitState {
  singbox_installed: boolean
  config_generated: boolean
  dashboard_installed: boolean
}

export interface InstallTask {
  id: string
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

export interface DNSServer {
  tag: string
  address: string
  address_resolver?: string
  address_strategy?: string
  strategy?: string
  detour?: string
}

export interface DNSRule {
  query_type?: string[]
  rule_set?: string[]
  server: string
  disable_cache?: boolean
  [key: string]: any
}

export interface DNS {
  servers?: DNSServer[]
  rules?: DNSRule[]
  final?: string
  strategy?: string
  disable_cache?: boolean
  disable_expire?: boolean
  independent_cache?: boolean
  fakeip?: {
    enabled: boolean
    inet4_range?: string
    inet6_range?: string
  }
}

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

export interface Inbound {
  tag: string
  type: string
  listen?: string
  listen_port?: number
  sniff?: boolean
  sniff_override_destination?: boolean
  domain_strategy?: string
  [key: string]: any
}

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
  dns?: {
    servers: DNSServer[]
    rules?: DNSRule[]
    hosts?: Record<string, string | string[]>
    final?: string
    strategy?: string
    disable_cache?: boolean
    disable_expire?: boolean
  }
  inbounds?: Inbound[]
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
