// Outbound Type definitions for sing-box v1.12.12
// Mirrors the structures from github.com/sagernet/sing-box@v1.12.12/option/outbound.go

import type { DomainStrategy } from './dns'

// Outbound Types
export type OutboundType =
  | 'direct'
  | 'block'
  | 'dns'
  | 'socks'
  | 'http'
  | 'shadowsocks'
  | 'vmess'
  | 'trojan'
  | 'wireguard'
  | 'hysteria'
  | 'hysteria2'
  | 'vless'
  | 'tuic'
  | 'shadowtls'
  | 'tor'
  | 'ssh'
  | 'selector'
  | 'urltest'

// Base Outbound structure
export interface BaseOutbound {
  type: OutboundType
  tag: string
}

// Dialer Options (common for most outbounds)
export interface DialerOptions {
  detour?: string
  bind_interface?: string
  inet4_bind_address?: string
  inet6_bind_address?: string
  protect_path?: string
  routing_mark?: number
  reuse_addr?: boolean
  netns?: string
  connect_timeout?: string
  tcp_fast_open?: boolean
  tcp_multi_path?: boolean
  udp_fragment?: boolean
  domain_strategy?: DomainStrategy
  fallback_delay?: string
}

// Server Options (common for proxy outbounds)
export interface ServerOptions {
  server: string
  server_port: number
}

// Direct Outbound
export interface DirectOutboundOptions extends DialerOptions {
  override_address?: string
  override_port?: number
}

// Block Outbound (no additional options)
export interface BlockOutboundOptions {}

// DNS Outbound (no additional options)
export interface DNSOutboundOptions {}

// SOCKS Outbound
export interface SocksOutboundOptions extends DialerOptions, ServerOptions {
  version?: string
  username?: string
  password?: string
  network?: string
  udp_over_tcp?: boolean | {
    enabled?: boolean
    version?: number
  }
}

// HTTP Outbound
export interface HTTPOutboundOptions extends DialerOptions, ServerOptions {
  username?: string
  password?: string
  tls?: OutboundTLSOptions
  path?: string
  headers?: Record<string, string | string[]>
}

// Shadowsocks Outbound
export interface ShadowsocksOutboundOptions extends DialerOptions, ServerOptions {
  method: string
  password: string
  plugin?: string
  plugin_opts?: string
  network?: string
  udp_over_tcp?: boolean | {
    enabled?: boolean
    version?: number
  }
  multiplex?: MultiplexOptions
}

// VMess Outbound
export interface VMessOutboundOptions extends DialerOptions, ServerOptions {
  uuid: string
  security?: string
  alter_id?: number
  global_padding?: boolean
  authenticated_length?: boolean
  network?: string
  tls?: OutboundTLSOptions
  packet_encoding?: string
  transport?: V2RayTransportOptions
  multiplex?: MultiplexOptions
}

// Trojan Outbound
export interface TrojanOutboundOptions extends DialerOptions, ServerOptions {
  password: string
  network?: string
  tls?: OutboundTLSOptions
  transport?: V2RayTransportOptions
  multiplex?: MultiplexOptions
}

// WireGuard Outbound
export interface WireGuardOutboundOptions extends DialerOptions, ServerOptions {
  system_interface?: boolean
  interface_name?: string
  local_address?: string[]
  private_key: string
  peers: Array<{
    server?: string
    server_port?: number
    public_key: string
    pre_shared_key?: string
    allowed_ips?: string[]
    reserved?: number[]
  }>
  peer_public_key?: string
  pre_shared_key?: string
  reserved?: number[]
  workers?: number
  mtu?: number
  network?: string
}

// Hysteria Outbound
export interface HysteriaOutboundOptions extends DialerOptions, ServerOptions {
  up?: string
  up_mbps?: number
  down?: string
  down_mbps?: number
  obfs?: string
  auth?: string
  auth_str?: string
  recv_window_conn?: number
  recv_window?: number
  disable_mtu_discovery?: boolean
  network?: string
  tls?: OutboundTLSOptions
}

// Hysteria2 Outbound
export interface Hysteria2OutboundOptions extends DialerOptions, ServerOptions {
  up_mbps?: number
  down_mbps?: number
  obfs?: {
    type?: string
    password?: string
  }
  password?: string
  network?: string
  tls?: OutboundTLSOptions
  brutal_debug?: boolean
}

// VLESS Outbound
export interface VLESSOutboundOptions extends DialerOptions, ServerOptions {
  uuid: string
  flow?: string
  network?: string
  tls?: OutboundTLSOptions
  packet_encoding?: string
  transport?: V2RayTransportOptions
  multiplex?: MultiplexOptions
}

// TUIC Outbound
export interface TUICOutboundOptions extends DialerOptions, ServerOptions {
  uuid: string
  password?: string
  congestion_control?: string
  udp_relay_mode?: string
  udp_over_stream?: boolean
  zero_rtt_handshake?: boolean
  heartbeat?: string
  network?: string
  tls?: OutboundTLSOptions
}

// ShadowTLS Outbound
export interface ShadowTLSOutboundOptions extends DialerOptions, ServerOptions {
  version?: number
  password?: string
  tls?: OutboundTLSOptions
}

// Tor Outbound
export interface TorOutboundOptions extends DialerOptions {
  executable_path?: string
  extra_args?: string[]
  data_directory?: string
  torrc?: Record<string, any>
}

// SSH Outbound
export interface SSHOutboundOptions extends DialerOptions, ServerOptions {
  user?: string
  password?: string
  private_key?: string
  private_key_path?: string
  private_key_passphrase?: string
  host_key?: string[]
  host_key_algorithms?: string[]
  client_version?: string
}

// Selector Outbound (group)
export interface SelectorOutboundOptions extends DialerOptions {
  outbounds: string[]
  default?: string
  interrupt_exist_connections?: boolean
}

// URLTest Outbound (group)
export interface URLTestOutboundOptions extends DialerOptions {
  outbounds: string[]
  url?: string
  interval?: string
  tolerance?: number
  idle_timeout?: string
  interrupt_exist_connections?: boolean
}

// Outbound TLS Options
export interface OutboundTLSOptions {
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

// Union type for all outbound options
export type Outbound = BaseOutbound & (
  | DirectOutboundOptions
  | BlockOutboundOptions
  | DNSOutboundOptions
  | SocksOutboundOptions
  | HTTPOutboundOptions
  | ShadowsocksOutboundOptions
  | VMessOutboundOptions
  | TrojanOutboundOptions
  | WireGuardOutboundOptions
  | HysteriaOutboundOptions
  | Hysteria2OutboundOptions
  | VLESSOutboundOptions
  | TUICOutboundOptions
  | ShadowTLSOutboundOptions
  | TorOutboundOptions
  | SSHOutboundOptions
  | SelectorOutboundOptions
  | URLTestOutboundOptions
)

// Type guards
export function isDirectOutbound(outbound: Outbound): outbound is BaseOutbound & DirectOutboundOptions {
  return outbound.type === 'direct'
}

export function isBlockOutbound(outbound: Outbound): outbound is BaseOutbound & BlockOutboundOptions {
  return outbound.type === 'block'
}

export function isDNSOutbound(outbound: Outbound): outbound is BaseOutbound & DNSOutboundOptions {
  return outbound.type === 'dns'
}

export function isSocksOutbound(outbound: Outbound): outbound is BaseOutbound & SocksOutboundOptions {
  return outbound.type === 'socks'
}

export function isHTTPOutbound(outbound: Outbound): outbound is BaseOutbound & HTTPOutboundOptions {
  return outbound.type === 'http'
}

export function isShadowsocksOutbound(outbound: Outbound): outbound is BaseOutbound & ShadowsocksOutboundOptions {
  return outbound.type === 'shadowsocks'
}

export function isVMessOutbound(outbound: Outbound): outbound is BaseOutbound & VMessOutboundOptions {
  return outbound.type === 'vmess'
}

export function isTrojanOutbound(outbound: Outbound): outbound is BaseOutbound & TrojanOutboundOptions {
  return outbound.type === 'trojan'
}

export function isWireGuardOutbound(outbound: Outbound): outbound is BaseOutbound & WireGuardOutboundOptions {
  return outbound.type === 'wireguard'
}

export function isHysteriaOutbound(outbound: Outbound): outbound is BaseOutbound & HysteriaOutboundOptions {
  return outbound.type === 'hysteria'
}

export function isHysteria2Outbound(outbound: Outbound): outbound is BaseOutbound & Hysteria2OutboundOptions {
  return outbound.type === 'hysteria2'
}

export function isVLESSOutbound(outbound: Outbound): outbound is BaseOutbound & VLESSOutboundOptions {
  return outbound.type === 'vless'
}

export function isTUICOutbound(outbound: Outbound): outbound is BaseOutbound & TUICOutboundOptions {
  return outbound.type === 'tuic'
}

export function isShadowTLSOutbound(outbound: Outbound): outbound is BaseOutbound & ShadowTLSOutboundOptions {
  return outbound.type === 'shadowtls'
}

export function isTorOutbound(outbound: Outbound): outbound is BaseOutbound & TorOutboundOptions {
  return outbound.type === 'tor'
}

export function isSSHOutbound(outbound: Outbound): outbound is BaseOutbound & SSHOutboundOptions {
  return outbound.type === 'ssh'
}

export function isSelectorOutbound(outbound: Outbound): outbound is BaseOutbound & SelectorOutboundOptions {
  return outbound.type === 'selector'
}

export function isURLTestOutbound(outbound: Outbound): outbound is BaseOutbound & URLTestOutboundOptions {
  return outbound.type === 'urltest'
}

// Helper to check if outbound is a proxy type (not group)
export function isProxyOutbound(outbound: Outbound): boolean {
  return !['selector', 'urltest'].includes(outbound.type)
}

// Helper to check if outbound is a group type
export function isGroupOutbound(outbound: Outbound): boolean {
  return ['selector', 'urltest'].includes(outbound.type)
}
