/**
 * The shape the route topology diagram renders: inbounds → rules → exits.
 *
 * A sibling of `ruleLadder.ts`, and deliberately not the same thing. The ladder
 * answers "where did THIS query go", one probe at a time. This answers "what
 * does the config do at all", with no probe and no running service — the view
 * an operator wants right after an edit, before deciding whether to start.
 *
 * WHY THE RIGHT COLUMN IS A SET, NOT A LIST
 * ─────────────────────────────────────────
 * `route.rules` is a flat, ordered list, and reading it top to bottom hides the
 * fact that many rules share a destination. This repo's own config routes to
 * `🤖 AI` from three unrelated rules — a `domain_suffix` list, a `rule_set`, and
 * a second `rule_set` group. sing-box has ONE outbound there. Drawing three
 * would say there are three exits, so exits are keyed by tag and the rules that
 * reach one are collected onto it. The convergence IS the OR: any of these
 * rules sends traffic out through that single outbound.
 *
 * TERMINATION IS COPIED FROM THE ENGINE, NOT REASONED ABOUT
 * ────────────────────────────────────────────────────────
 * Only `route`, `reject` and `hijack-dns` stop the walk. `sniff` and `resolve`
 * match and then change what the rules below them see, and `direct` — which
 * looks like it should terminate — does not, because sing-box's route.go has no
 * case for it. `app/pkg/routeprobe/rule_meta.go:39-53` mirrors that upstream
 * quirk rather than correcting it, and so does this: a picture that silently
 * fixes the engine is a picture of a config that is not running.
 */

/** What kind of thing a ribbon ends at. */
export type ExitKind =
  /** A tag some `outbounds[]` or `endpoints[]` entry defines. */
  | 'outbound'
  /** `action: reject` — the connection is refused, no outbound involved. */
  | 'reject'
  /** `action: hijack-dns` — answered by the DNS engine, never dialled out. */
  | 'hijack-dns'
  /** A tag nothing defines. sing-box only fails on this at START. */
  | 'missing'
  /** No outbounds at all, so sing-box synthesises a direct one. */
  | 'implicit'

/** One node in the right-hand column. */
export interface TopologyExit {
  /** Stable identity. The outbound tag, or the action name for a synthetic. */
  id: string
  /** What to print on the node. */
  label: string
  kind: ExitKind
  /** sing-box outbound type — `urltest`, `direct`, `shadowsocks`… */
  type?: string
  /** Members of a `selector`/`urltest`. Undefined for a leaf outbound. */
  memberCount?: number
  /** Extra qualifier, e.g. a reject rule's `method`. */
  detail?: string
  /** Rules routing here, in config order. Empty when only the fall-through does. */
  ruleIndices: number[]
  /** Reached by the fall-through, with or without any rule naming it. */
  isFinal: boolean
}

/** One matcher on a rule. Values within a single matcher are OR'd by sing-box. */
export interface TopologyCondition {
  key: string
  /** Always an array: sing-box accepts a scalar for every list-like matcher. */
  values: string[]
}

/** One rung in the middle column. */
export interface TopologyRule {
  /** Position in `route.rules`, as the config numbers it. */
  index: number
  /** The action, resolved — `route` when the key is omitted. */
  action: string
  /** Filled matchers, in inventory order. Conditions are AND'd with each other. */
  conditions: TopologyCondition[]
  /** `invert: true` flips the whole result, so it is not a condition. */
  inverted: boolean
  /** Inbounds this rule is restricted to. Empty means every inbound. */
  scopedInbounds: string[]
  /** Whether this action stops the walk. See the module header. */
  terminal: boolean
  /** The exit reached, or null for a rule that hands over to the next. */
  exitId: string | null
  /** The action's rendered result — a reject method, a resolve server. */
  outcome?: string
  /** Terminal with no conditions: it matches everything below it into itself. */
  catchAll: boolean
  /** False once an earlier `catchAll` rule has swallowed the remaining traffic. */
  reachable: boolean
}

/** One node in the left-hand column. */
export interface TopologyInbound {
  tag: string
  type: string
  listenPort?: number
}

/**
 * Which of sing-box's three fall-through cases applies.
 *
 * Collapsing these into "final" sends someone looking for a `route.final` that
 * is not in their config — the same distinction `routeprobe` draws.
 */
export type FallthroughSource = 'final' | 'first_outbound' | 'implicit_direct'

export interface RouteTopology {
  inbounds: TopologyInbound[]
  rules: TopologyRule[]
  exits: TopologyExit[]
  fallthrough: { exitId: string; source: FallthroughSource }
}
