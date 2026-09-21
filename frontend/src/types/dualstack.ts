/**
 * Types for the dual-stack domain test.
 *
 * The payload keeps four things apart on purpose, because a config can fail
 * any one of them while passing the others:
 *
 *   resolve   what sing-box returns for A and for AAAA
 *   predict   where each of those addresses WOULD be routed
 *   dial      whether it actually connects
 *   observe   which rule actually carried it
 *
 * The interesting row is the one where `predicted` and `observed` disagree —
 * that is the config bug the whole feature exists to surface.
 */

/**
 * Per-family outcome. Four states, not two.
 *
 * The two extra ones are the difference between a useful report and a
 * misleading one: a proxied domain SHOULD have no AAAA (that is what an IPv6
 * split does), and a panel on a host without global IPv6 cannot test IPv6 at
 * all. Rendering either as "unreachable" flags a working config as broken.
 */
export type Reachability = 'reachable' | 'unreachable' | 'no_address' | 'untested'

/** Whether the prediction and the observation agree. */
export type Agreement = 'agree' | 'differ' | 'unknown'

export type Family = 'ipv4' | 'ipv6'

/** How the DNS hijack was located in the config. */
export type DnsEndpointSource = 'none' | 'inbound' | 'protocol_dns'

export interface DualStackDnsEndpoint {
  source: DnsEndpointSource
  /** The inbound tag the hijack rule named, when it named one. */
  inbound?: string
  /** The inbound's configured bind, verbatim. */
  listen?: string
  port?: number
  /** host:port with a wildcard bind rewritten to loopback, ready to query. */
  address?: string
}

export interface DualStackPrediction {
  outbound: string
  /** Which config key produced it: rule, route.final, first_outbound, … */
  source?: string
  /** The deciding route rule, or -1 for a fall-through. */
  matched_index: number
  /**
   * False when an undecidable rule sat ahead of the decision. A mismatch
   * against an inexact prediction is the prediction's fault, not the
   * config's, so the UI must not call it a disagreement.
   */
  exact: boolean
  error?: string
}

export interface DualStackDial {
  status: Reachability
  elapsed_ms: number
  error?: string
  /**
   * The peer as the kernel reports it. An IPv4-mapped peer on an IPv6 row
   * means the dial never went out over IPv6.
   */
  peer?: string
  tls_handshake?: boolean
}

export interface DualStackObserved {
  /** sing-box's own rule string, verbatim. There is no rule index in it. */
  rule: string
  /** The outbound the rule named — the LAST element of `chains`. */
  outbound: string
  /** The leaf actually dialled; for a group, the member elected. */
  via?: string
  inbound?: string
  host?: string
}

export interface DualStackFamily {
  family: Family
  query_type: string
  addresses: string[]
  /** The address actually dialled — the first returned. */
  tested?: string
  dns_elapsed_ms: number
  dns_error?: string
  predicted?: DualStackPrediction
  dial?: DualStackDial
  observed?: DualStackObserved
  status: Reachability
  agreement: Agreement
}

export interface DualStackDomain {
  domain: string
  families: DualStackFamily[]
  error?: string
}

export interface DualStackResult {
  domains: DualStackDomain[]
  dns_endpoint: DualStackDnsEndpoint
  port: number
  client_ipv4: boolean
  client_ipv6: boolean
  /** Why correlation was unavailable; applies to every row at once. */
  observe_error?: string
}

export interface DualStackRequest {
  domains: string[]
  port?: number
  tls?: boolean
  skip_dial?: boolean
  timeout_ms?: number
}

/** At most this many domains per request; mirrors dualstack.MaxDomains. */
export const MAX_DOMAINS = 10
