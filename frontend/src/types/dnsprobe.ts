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

import type { RuleSetStatus } from './ruleSet'

export type { RuleSetStatus, RuleSetReason, RuleSetTier } from './ruleSet'

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
  /**
   * Conditions that could not be decided offline.
   *
   * `rule_set` and `query_type` used to live here on every rule that carried
   * them. Both are now decided — the query type was always known, and rule
   * sets are read from sing-box's own cache — so seeing either here means the
   * set genuinely could not be read (`rule_sets` says which, and why) or the
   * caller did not name a record type.
   */
  unevaluated?: string[]
  action: string
  server?: string
  strategy?: string
  /** Per-set detail when the rule references any. Same shape route reports. */
  rule_sets?: RuleSetStatus[]
  /**
   * Whether a match on this rule ENDS the walk.
   *
   * Not derivable from the action name. sing-box's matchDNS switch returns
   * only for route, reject and predefined — so `evaluate` and `route-options`
   * match, change what the rules below them see, and hand over. A config that
   * races three `evaluate` resolvers has three matched rules and no decision,
   * and a ladder that treated the first match as the answer would name the
   * wrong server with full confidence.
   */
  terminal?: boolean
  /** What a non-terminal match changed for the rules below it. */
  effect?: string
}

export interface DnsAttribution {
  rules: DnsRuleEvaluation[]
  /** First matching rule, or -1 when the query falls through to `final`. */
  matched_index: number
  server: string
  strategy: string
  /** Which key supplied `strategy`: the matched rule, or dns.strategy. */
  strategy_source?: 'rule' | 'default'
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
  /** Why the server was not queried; translated by the UI. */
  skip_reason?: 'detour' | 'unsupported_type'
  /** The detour name or server type behind skip_reason. */
  skip_detail?: string
  error?: string
  records: string[]
  elapsed_ms: number
}

/** The DNS server a query was routed to, resolved against the config. */
export interface DnsServerDetail {
  tag: string
  type: string
  /** host:port; empty for local server types (hosts, fakeip). */
  address?: string
  /** Outbound the server is reached through, when set. */
  detour?: string
  /** False when the tag matches no configured server. */
  found: boolean
}

export interface DnsProbeResult {
  domain: string
  query_type: string
  live?: DnsLiveResult
  live_error?: string
  attribution: DnsAttribution
  /** Which server the query used, with its address. Absent when answered locally. */
  resolved_server?: DnsServerDetail
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
