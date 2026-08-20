/**
 * Types for the route simulator.
 *
 * This is a PRE-flight tool and the payload is shaped around that. sing-box's
 * Clash API already reports where connections went; what it cannot answer is
 * "will this config do what I meant?" — which is the question asked right after
 * an edit, when the traffic has not been sent yet.
 *
 * Because nothing is observed, everything here is a prediction, and the payload
 * is explicit about how good a prediction it is:
 *  - `exact` is false whenever a rule ahead of the decision could not be
 *    evaluated; it could have matched first, making the answer below it a guess.
 *  - per-rule `unevaluated` names the conditions responsible.
 *  - per-rule `rule_sets` says which set could not be read, and why.
 */

/** Verdict for one rule against the probed destination. */
export type RouteMatchState = 'matched' | 'not_matched' | 'unevaluated' | 'skipped'

/** Why a rule set could not be consulted. Keys, translated by the UI. */
export type RuleSetReason =
  | 'unknown_tag'
  | 'not_cached'
  | 'cache_unavailable'
  | 'cache_disabled'
  | 'file_missing'
  | 'unsupported_srs_version'
  | 'parse_error'

export interface RouteRuleSetStatus {
  tag: string
  state: RouteMatchState
  reason?: RuleSetReason
  detail?: string
  /** When sing-box last downloaded the set. A surprise is often a stale set. */
  updated_at_unix?: number
}

export interface RouteRuleEvaluation {
  index: number
  type: string
  state: RouteMatchState
  /** The rule's conditions, rendered for display. */
  summary: string
  /** Conditions that could not be decided offline, e.g. ["clash_mode"]. */
  unevaluated?: string[]
  action: string
  /** The action's result: an outbound tag for a route, a server for a resolve. */
  outcome?: string
  /** Whether this action would stop rule matching. */
  terminal: boolean
  /** What a non-terminal rule changed for the rules below it. */
  effect?: string
  rule_sets?: RouteRuleSetStatus[]
}

/** Which config key produced the outbound when no rule matched. */
export type OutboundSource = 'rule' | 'route.final' | 'first_outbound' | 'implicit_direct'

export interface RouteProbeResult {
  destination: string
  domain?: string
  ip?: string
  /** Where `ip` came from. "dns" means the panel asked sing-box to resolve it. */
  ip_source?: 'input' | 'dns'
  resolve_error?: string

  port: number
  network: string
  inbound?: string
  protocol?: string

  rules: RouteRuleEvaluation[]
  /** The deciding rule, or -1 when the destination falls through. */
  matched_index: number
  outbound: string
  outbound_source: OutboundSource
  action: string
  final_used: boolean

  /** True only when every rule ahead of the decision was fully evaluated. */
  exact: boolean
  unevaluated_before: number
  /** Machine keys for rule-set problems, translated by the UI. */
  warnings?: string[]
}

export interface RouteProbeRequest {
  destination: string
  port?: number
  network?: string
  inbound?: string
  source_ip?: string
  protocol?: string
}

/** Networks a route rule can key on. */
export const ROUTE_NETWORKS = ['tcp', 'udp'] as const

/**
 * Protocols sing-box's sniffers can report. Offered because a single
 * `protocol: dns` rule near the top of a config otherwise leaves EVERY
 * prediction marked inexact, which teaches the reader to ignore the warning.
 */
export const SNIFFED_PROTOCOLS = ['tls', 'http', 'quic', 'dns', 'bittorrent', 'ssh', 'rdp'] as const
