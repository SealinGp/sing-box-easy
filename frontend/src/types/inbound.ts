// Inbound Type definitions for sing-box v1.12.12
// Mirrors the structures from github.com/sagernet/sing-box@v1.12.12/option/inbound.go

import type { DomainStrategy } from './dns'

// Inbound Types
export type InboundType =
  | 'tun'
  | 'redirect'
  | 'tproxy'
  | 'direct'
  | 'socks'
  | 'http'
  | 'mixed'
  | 'shadowsocks'
  | 'vmess'
  | 'trojan'
  | 'naive'
  | 'hysteria'
  | 'hysteria2'
  | 'vless'
  | 'tuic'
  | 'shadowtls'

// Base Inbound structure
export interface BaseInbound {
  type: InboundType
  tag: string
}

// Deprecated InboundOptions (kept for compatibility)
export interface InboundOptions {
  sniff?: boolean
  sniff_override_destination?: boolean
  sniff_timeout?: string
  domain_strategy?: DomainStrategy
  udp_disable_domain_unmapping?: boolean
  detour?: string
}

// Listen Options (common for most inbounds)
export interface ListenOptions extends InboundOptions {
  listen?: string
  listen_port?: number
  bind_interface?: string
  routing_mark?: number
  reuse_addr?: boolean
  netns?: string
  tcp_keep_alive?: string
  tcp_keep_alive_interval?: string
  tcp_fast_open?: boolean
  tcp_multi_path?: boolean
  udp_fragment?: boolean
  udp_timeout?: string
  // Deprecated fields
  proxy_protocol?: boolean
  proxy_protocol_accept_no_header?: boolean
}

// TUN Inbound Options
export interface TunInboundOptions extends InboundOptions {
  interface_name?: string
  mtu?: number
  gso?: boolean
  address?: string[]
  inet4_address?: string | string[]
  inet6_address?: string | string[]
  auto_route?: boolean
  strict_route?: boolean
  inet4_route_address?: string[]
  inet6_route_address?: string[]
  inet4_route_exclude_address?: string[]
  inet6_route_exclude_address?: string[]
  endpoint_independent_nat?: boolean
  udp_timeout?: string
  stack?: string
  include_interface?: string[]
  exclude_interface?: string[]
  include_uid?: number[]
  include_uid_range?: string[]
  exclude_uid?: number[]
  exclude_uid_range?: string[]
  include_android_user?: number[]
  include_package?: string[]
  exclude_package?: string[]
  platform?: {
    http_proxy?: {
      enabled?: boolean
      server?: string
      server_port?: number
      bypass_domain?: string[]
      match_domain?: string[]
    }
  }
}

// Redirect Inbound Options
export interface RedirectInboundOptions extends ListenOptions {}

// TProxy Inbound Options
export interface TProxyInboundOptions extends ListenOptions {}

// Direct Inbound Options
export interface DirectInboundOptions extends ListenOptions {
  network?: string
  override_address?: string
  override_port?: number
}

// SOCKS Inbound Options
export interface SocksInboundOptions extends ListenOptions {
  users?: Array<{
    username: string
    password: string
  }>
}

// HTTP/Mixed Inbound Options
export interface HTTPMixedInboundOptions extends ListenOptions {
  users?: Array<{
    username: string
    password: string
  }>
  tls?: InboundTLSOptions
  set_system_proxy?: boolean
}

// Shadowsocks Inbound Options
export interface ShadowsocksInboundOptions extends ListenOptions {
  method: string
  password: string
  users?: Array<{
    name?: string
    password: string
  }>
  destinations?: Array<{
    name?: string
    server: string
    server_port: number
    password?: string
  }>
  multiplex?: MultiplexOptions
}

// VMess Inbound Options
export interface VMessInboundOptions extends ListenOptions {
  users: Array<{
    name?: string
    uuid: string
    alterId?: number
  }>
  tls?: InboundTLSOptions
  transport?: V2RayTransportOptions
  multiplex?: MultiplexOptions
}

// Trojan Inbound Options
export interface TrojanInboundOptions extends ListenOptions {
  users: Array<{
    name?: string
    password: string
  }>
  tls?: InboundTLSOptions
  fallback?: {
    server?: string
    server_port?: number
  }
  fallback_for_alpn?: Record<string, {
    server?: string
    server_port?: number
  }>
  transport?: V2RayTransportOptions
  multiplex?: MultiplexOptions
}

// Naive Inbound Options
export interface NaiveInboundOptions extends ListenOptions {
  users: Array<{
    username: string
    password: string
  }>
  tls?: InboundTLSOptions
  network?: string
}

// Hysteria Inbound Options
export interface HysteriaInboundOptions extends ListenOptions {
  up?: string
  up_mbps?: number
  down?: string
  down_mbps?: number
  obfs?: string
  users?: Array<{
    name?: string
    auth?: string
    auth_str?: string
  }>
  recv_window_conn?: number
  recv_window_client?: number
  max_conn_client?: number
  disable_mtu_discovery?: boolean
  tls?: InboundTLSOptions
}

// Hysteria2 Inbound Options
export interface Hysteria2InboundOptions extends ListenOptions {
  up_mbps?: number
  down_mbps?: number
  obfs?: {
    type?: string
    password?: string
  }
  users?: Array<{
    name?: string
    password?: string
  }>
  ignore_client_bandwidth?: boolean
  tls?: InboundTLSOptions
  masquerade?: string
  brutal_debug?: boolean
}

// VLESS Inbound Options
export interface VLESSInboundOptions extends ListenOptions {
  users: Array<{
    name?: string
    uuid: string
    flow?: string
  }>
  tls?: InboundTLSOptions
  transport?: V2RayTransportOptions
  multiplex?: MultiplexOptions
}

// TUIC Inbound Options
export interface TUICInboundOptions extends ListenOptions {
  users: Array<{
    name?: string
    uuid: string
    password?: string
  }>
  congestion_control?: string
  auth_timeout?: string
  zero_rtt_handshake?: boolean
  heartbeat?: string
  tls?: InboundTLSOptions
}

// ShadowTLS Inbound Options
export interface ShadowTLSInboundOptions extends ListenOptions {
  version?: number
  password?: string
  users?: Array<{
    name?: string
    password?: string
  }>
  handshake?: {
    server: string
    server_port: number
  }
  handshake_for_server_name?: Record<string, {
    server: string
    server_port: number
  }>
  strict_mode?: boolean
}

// Inbound TLS Options
export interface InboundTLSOptions {
  enabled?: boolean
  server_name?: string
  alpn?: string[]
  min_version?: string
  max_version?: string
  cipher_suites?: string[]
  certificate?: string[]
  certificate_path?: string
  key?: string[]
  key_path?: string
  acme?: {
    domain?: string[]
    data_directory?: string
    default_server_name?: string
    email?: string
    provider?: string
    disable_http_challenge?: boolean
    disable_tls_alpn_challenge?: boolean
    alternative_http_port?: number
    alternative_tls_port?: number
    external_account?: {
      key_id?: string
      mac_key?: string
    }
    dns01_challenge?: {
      provider?: string
      [key: string]: any
    }
  }
  ech?: {
    enabled?: boolean
    pq_signature_schemes_enabled?: boolean
    dynamic_record_sizing_disabled?: boolean
    key?: string[]
    key_path?: string
  }
  reality?: {
    enabled?: boolean
    handshake?: {
      server?: string
      server_port?: number
    }
    private_key?: string
    short_id?: string[]
    max_time_difference?: string
  }
}

// V2Ray Transport Options
export interface V2RayTransportOptions {
  type?: string
  // WebSocket
  path?: string
  headers?: Record<string, string>
  max_early_data?: number
  early_data_header_name?: string
  // HTTP
  host?: string[]
  method?: string
  // gRPC
  service_name?: string
  idle_timeout?: string
  ping_timeout?: string
  permit_without_stream?: boolean
  // HTTPUpgrade
  // (similar to WebSocket)
}

// Multiplex Options
export interface MultiplexOptions {
  enabled?: boolean
  protocol?: string
  max_connections?: number
  min_streams?: number
  max_streams?: number
  padding?: boolean
  brutal?: {
    enabled?: boolean
    up_mbps?: number
    down_mbps?: number
  }
}

// Union type for all inbound options
export type InboundWithOptions =
  | (BaseInbound & { type: 'tun'; options?: TunInboundOptions })
  | (BaseInbound & { type: 'redirect'; options?: RedirectInboundOptions })
  | (BaseInbound & { type: 'tproxy'; options?: TProxyInboundOptions })
  | (BaseInbound & { type: 'direct'; options?: DirectInboundOptions })
  | (BaseInbound & { type: 'socks'; options?: SocksInboundOptions })
  | (BaseInbound & { type: 'http'; options?: HTTPMixedInboundOptions })
  | (BaseInbound & { type: 'mixed'; options?: HTTPMixedInboundOptions })
  | (BaseInbound & { type: 'shadowsocks'; options?: ShadowsocksInboundOptions })
  | (BaseInbound & { type: 'vmess'; options?: VMessInboundOptions })
  | (BaseInbound & { type: 'trojan'; options?: TrojanInboundOptions })
  | (BaseInbound & { type: 'naive'; options?: NaiveInboundOptions })
  | (BaseInbound & { type: 'hysteria'; options?: HysteriaInboundOptions })
  | (BaseInbound & { type: 'hysteria2'; options?: Hysteria2InboundOptions })
  | (BaseInbound & { type: 'vless'; options?: VLESSInboundOptions })
  | (BaseInbound & { type: 'tuic'; options?: TUICInboundOptions })
  | (BaseInbound & { type: 'shadowtls'; options?: ShadowTLSInboundOptions })

// Simplified Inbound type (flattened structure - options merged with base)
// This matches how the JSON is actually structured
export type Inbound = BaseInbound & (
  | TunInboundOptions
  | RedirectInboundOptions
  | TProxyInboundOptions
  | DirectInboundOptions
  | SocksInboundOptions
  | HTTPMixedInboundOptions
  | ShadowsocksInboundOptions
  | VMessInboundOptions
  | TrojanInboundOptions
  | NaiveInboundOptions
  | HysteriaInboundOptions
  | Hysteria2InboundOptions
  | VLESSInboundOptions
  | TUICInboundOptions
  | ShadowTLSInboundOptions
)

// Type guards
export function isTunInbound(inbound: Inbound): inbound is BaseInbound & TunInboundOptions {
  return inbound.type === 'tun'
}

export function isRedirectInbound(inbound: Inbound): inbound is BaseInbound & RedirectInboundOptions {
  return inbound.type === 'redirect'
}

export function isTProxyInbound(inbound: Inbound): inbound is BaseInbound & TProxyInboundOptions {
  return inbound.type === 'tproxy'
}

export function isDirectInbound(inbound: Inbound): inbound is BaseInbound & DirectInboundOptions {
  return inbound.type === 'direct'
}

export function isSocksInbound(inbound: Inbound): inbound is BaseInbound & SocksInboundOptions {
  return inbound.type === 'socks'
}

export function isHTTPInbound(inbound: Inbound): inbound is BaseInbound & HTTPMixedInboundOptions {
  return inbound.type === 'http'
}

export function isMixedInbound(inbound: Inbound): inbound is BaseInbound & HTTPMixedInboundOptions {
  return inbound.type === 'mixed'
}

export function isShadowsocksInbound(inbound: Inbound): inbound is BaseInbound & ShadowsocksInboundOptions {
  return inbound.type === 'shadowsocks'
}

export function isVMessInbound(inbound: Inbound): inbound is BaseInbound & VMessInboundOptions {
  return inbound.type === 'vmess'
}

export function isTrojanInbound(inbound: Inbound): inbound is BaseInbound & TrojanInboundOptions {
  return inbound.type === 'trojan'
}

export function isNaiveInbound(inbound: Inbound): inbound is BaseInbound & NaiveInboundOptions {
  return inbound.type === 'naive'
}

export function isHysteriaInbound(inbound: Inbound): inbound is BaseInbound & HysteriaInboundOptions {
  return inbound.type === 'hysteria'
}

export function isHysteria2Inbound(inbound: Inbound): inbound is BaseInbound & Hysteria2InboundOptions {
  return inbound.type === 'hysteria2'
}

export function isVLESSInbound(inbound: Inbound): inbound is BaseInbound & VLESSInboundOptions {
  return inbound.type === 'vless'
}

export function isTUICInbound(inbound: Inbound): inbound is BaseInbound & TUICInboundOptions {
  return inbound.type === 'tuic'
}

export function isShadowTLSInbound(inbound: Inbound): inbound is BaseInbound & ShadowTLSInboundOptions {
  return inbound.type === 'shadowtls'
}
