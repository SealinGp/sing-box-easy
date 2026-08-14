/**
 * Types for the DNS route inspector.
 *
 * The payload deliberately separates fact from prediction:
 *  - `live` is what sing-box itself resolved (ground truth).
 *  - `logged_matches` is sing-box's own record of which rule fired, available
 *    only when it is running with debug logging.
 *  - `attribution` is reconstructed by the backend and may be inexact; it says
 *    so via `exact` and per-rule `unevaluated`.
 */

/** Verdict for one rule against the probed domain. */
export type MatchState = 'matched' | 'not_matched' | 'unevaluated'

export interface DnsAnswer {
  name: string
  type: string
  ttl: number
  data: string
}

export interface DnsLiveResult {
  status: number
  answers: DnsAnswer[]
  elapsed_ms: number
}

export interface DnsRuleEvaluation {
  index: number
  type: string
  state: MatchState
  /** The rule's conditions, rendered for display. */
  summary: string
  /** Conditions that could not be decided offline, e.g. ["rule_set"]. */
  unevaluated?: string[]
  action: string
  server?: string
  strategy?: string
}

export interface DnsAttribution {
  rules: DnsRuleEvaluation[]
  /** First matching rule, or -1 when the query falls through to `final`. */
  matched_index: number
  server: string
  strategy: string
  final_used: boolean
  /**
   * False when an unevaluated rule sits ahead of the decision — it could have
   * matched first, so the prediction below it is a guess.
   */
  exact: boolean
  unevaluated_before: number
}

/** One decision line sing-box printed for the query. */
export interface DnsLoggedMatch {
  logged_index: number
  /** `logged_index` decoded back to a dns.rules position, or -1. */
  config_index: number
  description: string
  action: string
  /** True when the decoded index was corroborated by the rule's conditions. */
  verified: boolean
  raw: string
}

/** One configured upstream's answer, for comparing resolvers. */
export interface DnsServerResult {
  tag: string
  type: string
  address?: string
  skipped?: string
  error?: string
  records: string[]
  elapsed_ms: number
}

export interface DnsProbeResult {
  domain: string
  query_type: string
  live?: DnsLiveResult
  live_error?: string
  attribution: DnsAttribution
  logged_matches: DnsLoggedMatch[]
  /** Machine-readable note about the log evidence; translated by the UI. */
  log_status?: '' | 'no_lines' | 'ambiguous' | 'read_error'
  /** Underlying message when log_status is 'read_error'. */
  log_error?: string
  servers: DnsServerResult[]
  /** Two reachable servers returned different records. */
  disagreement: boolean
}

export interface DnsProbeRequest {
  domain: string
  type?: string
  compare_servers?: boolean
}

/** Record types the backend accepts. */
export const DNS_QUERY_TYPES = ['A', 'AAAA', 'CNAME', 'TXT', 'MX', 'NS'] as const
