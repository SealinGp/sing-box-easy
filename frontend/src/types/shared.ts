// Shared type definitions used across multiple modules

export interface DialerOptions {
  detour?: string
  bind_interface?: string
  inet4_bind_address?: string
  inet6_bind_address?: string
  protect_path?: string
  routing_mark?: number
  reuse_addr?: boolean
  connect_timeout?: string
  tcp_fast_open?: boolean
  tcp_multi_path?: boolean
  udp_fragment?: boolean
  domain_strategy?: 'prefer_ipv4' | 'prefer_ipv6' | 'ipv4_only' | 'ipv6_only' | ''
  fallback_delay?: string
}

export interface V2RayTransportOptions {
  type: 'http' | 'ws' | 'quic' | 'grpc' | 'httpupgrade'
  host?: string[]
  path?: string
  method?: string
  headers?: Record<string, string | string[]>
  idle_timeout?: string
  ping_timeout?: string
  max_early_data?: number
  early_data_header_name?: string
  service_name?: string
  permit_without_stream?: boolean
}

export interface MultiplexOptions {
  enabled?: boolean
  protocol?: 'smux' | 'yamux' | 'h2mux'
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