/**
 * One frame of the live traffic overlay — the wire shape of a `frame` event on
 * `GET /traffic/flow/stream`. Mirrors `app/pkg/trafficflow/types.go`.
 *
 * Every list is keyed by something the expected-flow diagram already has a
 * node for (an inbound tag, a rule index, an outbound tag), which is what lets
 * the browser light the existing drawing rather than lay out a second one. The
 * one exception is `via`: the leaf a group actually dialled through, which the
 * expected diagram deliberately does not draw.
 *
 * Rates are BYTES PER SECOND, derived server-side from two samples and the
 * wall-clock time between them. A `Fresh` connection (first sighting) reports 0
 * and must not be read as idle — that is why the first frame after connecting
 * shows structure but no motion.
 */

export type TrafficRuleKind = 'rule' | 'final' | 'unmatched'

export interface TrafficTotals {
  down: number
  up: number
  /** After the filter. */
  connections: number
  /** Before the filter. */
  all: number
  /** Vanished since the previous sample. */
  closed: number
}

export interface TrafficInboundFlow {
  tag: string
  down: number
  up: number
  connections: number
}

export interface TrafficHostFlow {
  host: string
  down: number
}

export interface TrafficRuleFlow {
  kind: TrafficRuleKind
  /** Valid for `kind === 'rule'`; -1 otherwise. */
  index: number
  /** The raw string sing-box reported, for `kind === 'unmatched'`. */
  rule?: string
  /** The outbound the rule sent traffic to — the last chain element. */
  exit: string
  down: number
  up: number
  connections: number
  /** Top destinations, highest rate first, at most three. */
  hosts: TrafficHostFlow[]
}

export interface TrafficViaFlow {
  tag: string
  down: number
  connections: number
}

export interface TrafficExitFlow {
  tag: string
  down: number
  up: number
  connections: number
  /** Leaf nodes a group dialled through; empty for a leaf outbound. */
  via: TrafficViaFlow[]
}

/**
 * One client address currently holding connections — an entry in the source
 * filter's picker.
 *
 * Collected server-side BEFORE the filter, so the list keeps every device on
 * offer while narrowed to one of them, and ordered by address rather than by
 * rate so the entries do not reshuffle under the cursor once a second.
 */
export interface TrafficSourceFlow {
  ip: string
  down: number
  up: number
  connections: number
}

export interface TrafficFrame {
  /** Sample time, unix milliseconds. */
  at: number
  totals: TrafficTotals
  inbounds: TrafficInboundFlow[]
  /** Sorted by `down`, highest first. */
  rules: TrafficRuleFlow[]
  /** Sorted by `down`, highest first. */
  exits: TrafficExitFlow[]
  /** Every client address in the snapshot, pre-filter, ordered by address. */
  sources: TrafficSourceFlow[]
  filtered: boolean
  /** Connections whose rule string has no index in the running rule list. */
  unmatched: number
}

/** What an operator narrows the stream to. Sent as query parameters. */
export interface TrafficFilter {
  sourceIp: string
  host: string
}
