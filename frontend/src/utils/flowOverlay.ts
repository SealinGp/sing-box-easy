/**
 * Turns one live traffic frame into what the expected-flow diagram needs to
 * light itself: a per-ribbon lookup, a per-exit lookup, and which ribbons move.
 *
 * The frame is already aggregated by the server onto the same keys the layout
 * uses (rule index, exit tag, inbound tag), so this is a re-keying plus two
 * decisions: WHICH ribbons animate, and HOW FAST.
 *
 * WHICH: the top N by download rate, ranked across rule and fall-through flows
 * together. On a busy router hundreds of connections cross dozens of rules,
 * and animating all of them is a screen of moving dashes that says nothing.
 * Everything else with traffic is lit but still, so the shape stays visible
 * and the motion is the signal.
 *
 * HOW FAST: a log scale. 10 KB/s and 10 MB/s must both read as motion; on a
 * linear scale everything below the fastest edge looks stopped.
 */
import { formatBytes } from './formatBytes'
import type { TrafficExitFlow, TrafficFrame, TrafficRuleFlow, TrafficTotals } from '../types/trafficFlow'

/**
 * A rate for display. Rates are derived (bytes ÷ seconds) and so carry
 * floating-point noise that `formatBytes` — written for byte COUNTS, which
 * are integers — passes straight through below 1 KB: "391.55309828159835 B".
 * Rounding first is the whole fix; the "/s" is the caller's, since the label
 * is localised.
 */
export function formatRate(bytesPerSec: number): string {
  return formatBytes(Math.max(0, Math.round(bytesPerSec)))
}

/** How many rule ribbons animate at once. */
export const TOP_N = 8

/** Dash cycle bounds, seconds. */
const SLOWEST_CYCLE = 4
const FASTEST_CYCLE = 0.3
/** The rate at which the cycle is SLOWEST_CYCLE; each decade above halves it. */
const BASE_RATE = 1_000

const MIN_WIDTH = 1.5
const MAX_WIDTH = 6

export interface LiveEdge {
  down: number
  up: number
  connections: number
  /** In the top N — draw the moving dashes. */
  animated: boolean
  /** Seconds per dash cycle; only meaningful when animated. */
  durationSec: number
  /** Stroke width for the lit ribbon. */
  width: number
}

export interface LiveExit extends TrafficExitFlow {
  animated: boolean
}

export interface FlowOverlay {
  /** Keyed by ribbon id — `rule:<index>`, `final`, `in:<tag>`. */
  ribbons: Map<string, LiveEdge>
  /** Keyed by exit tag. */
  exits: Map<string, LiveExit>
  /** Keyed by rule index, for row lamps and tooltips. */
  rules: Map<number, TrafficRuleFlow>
  finalFlow: TrafficRuleFlow | null
  /** Flows whose rule string has no row — lit by exit only. */
  unmatched: TrafficRuleFlow[]
  totals: TrafficTotals
  /** How many rule/final ribbons are animating this frame. */
  animatedRules: number
  at: number
}

/** Seconds per dash cycle for a rate, log-scaled and clamped. */
export function dashDurationFor(bytesPerSec: number): number {
  if (bytesPerSec <= 0) return SLOWEST_CYCLE
  const decades = Math.log10(bytesPerSec / BASE_RATE)
  const cycle = SLOWEST_CYCLE / Math.pow(2, decades)
  return Math.min(SLOWEST_CYCLE, Math.max(FASTEST_CYCLE, cycle))
}

/** Stroke width for a rate — grows a little with each decade, bounded. */
export function ribbonWidthFor(bytesPerSec: number): number {
  if (bytesPerSec <= 0) return MIN_WIDTH
  const grown = MIN_WIDTH + Math.log10(bytesPerSec + 1) * 0.5
  return Math.min(MAX_WIDTH, grown)
}

const edge = (down: number, up: number, connections: number, animated: boolean): LiveEdge => ({
  down,
  up,
  connections,
  animated: animated && down > 0,
  durationSec: dashDurationFor(down),
  width: ribbonWidthFor(down),
})

const ribbonKey = (flow: TrafficRuleFlow): string | null => {
  if (flow.kind === 'rule') return `rule:${flow.index}`
  if (flow.kind === 'final') return 'final'
  return null
}

export function buildFlowOverlay(frame: TrafficFrame, topN: number = TOP_N): FlowOverlay {
  const ribbons = new Map<string, LiveEdge>()
  const rules = new Map<number, TrafficRuleFlow>()
  const unmatched: TrafficRuleFlow[] = []
  let finalFlow: TrafficRuleFlow | null = null

  // The server sorts by `down` descending, but rank explicitly so a client
  // never depends on wire order for a visual decision.
  const placeable = frame.rules.filter((flow) => ribbonKey(flow) !== null)
  const ranked = [...placeable].sort((a, b) => b.down - a.down)
  const top = new Set(ranked.slice(0, topN).map(ribbonKey))

  let animatedRules = 0
  for (const flow of frame.rules) {
    const key = ribbonKey(flow)
    if (key === null) {
      unmatched.push(flow)
      continue
    }
    const live = edge(flow.down, flow.up, flow.connections, top.has(key))
    ribbons.set(key, live)
    if (live.animated) animatedRules += 1
    if (flow.kind === 'rule') rules.set(flow.index, flow)
    else finalFlow = flow
  }

  for (const inbound of frame.inbounds) {
    ribbons.set(`in:${inbound.tag}`, edge(inbound.down, inbound.up, inbound.connections, true))
  }

  const exits = new Map<string, LiveExit>()
  const animatedExits = new Set(
    frame.rules.filter((flow) => top.has(ribbonKey(flow)) && flow.down > 0).map((flow) => flow.exit),
  )
  for (const exit of frame.exits) {
    exits.set(exit.tag, { ...exit, animated: animatedExits.has(exit.tag) })
  }

  return {
    ribbons,
    exits,
    rules,
    finalFlow,
    unmatched,
    totals: frame.totals,
    animatedRules,
    at: frame.at,
  }
}
