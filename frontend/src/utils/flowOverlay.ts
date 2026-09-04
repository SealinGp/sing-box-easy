/**
 * Turns one live traffic frame into what the expected-flow diagram needs to
 * light itself: a per-ribbon lookup, a per-exit lookup, and which ribbons move.
 *
 * The frame is already aggregated by the server onto the same keys the layout
 * uses (rule index, exit tag, inbound tag), so this is a re-keying plus three
 * decisions: WHICH ribbons count as carrying traffic, which of those animate,
 * and HOW FAST.
 *
 * WHICH COUNT: everything at or above `RATE_FLOOR`. Without a floor, "carrying
 * traffic" means "appears in the frame", and on a real router that is nearly
 * every rule — one DNS lookup or one keepalive is enough — so the whole ladder
 * lights and the distinction stops meaning anything.
 *
 * WHICH MOVE: the top N by download rate among those, ranked across rule and
 * fall-through flows together. On a busy router hundreds of connections cross
 * dozens of rules, and animating all of them is a screen of moving pulses that
 * says nothing. Everything else above the floor is lit but still, so the shape
 * stays visible and the motion is the signal.
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

/**
 * Below this combined rate, a flow is trickle rather than traffic.
 *
 * On a real router almost every rule carries SOMETHING — a DNS lookup, a
 * keepalive, one connection at 200 B/s — so "has an entry in the frame" stops
 * discriminating and the ladder lights end to end. Anything the operator can
 * act on is orders of magnitude above this; 1 KB/s is the knee, and the exact
 * value matters less than there being one.
 *
 * Down and up are summed: a backup pushing 4 MB/s upstream is not trickle, and
 * ranking (which is by download) must not be what decides whether a row counts
 * as busy.
 *
 * Suppressed flows are dimmed, never dropped — their rates, connection counts
 * and hosts stay in the tooltip, and `belowFloor` says how many there are, so
 * a quiet diagram is explained rather than mysterious.
 */
export const RATE_FLOOR = 1_024

/** Pulse traversal bounds, seconds — one trip from inbound to exit. */
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
  /**
   * Carrying enough to be worth looking at — at or above `RATE_FLOOR`.
   *
   * This, not "the frame has an entry", is what the diagram treats as
   * carrying traffic. A trickling ribbon stays at the idle opacity.
   */
  lit: boolean
  /** In the top N of the LIT flows — draw the travelling pulse. */
  animated: boolean
  /** Seconds per pulse traversal; only meaningful when animated. */
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
  /** Rule/final flows with traffic, but under `RATE_FLOOR`. Dimmed, not dropped. */
  belowFloor: number
  at: number
}

/** Seconds per pulse traversal for a rate, log-scaled and clamped. */
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

/** Whether a flow clears the noise floor. See `RATE_FLOOR`. */
export function isLit(down: number, up: number): boolean {
  return down + up >= RATE_FLOOR
}

const edge = (down: number, up: number, connections: number, animated: boolean): LiveEdge => {
  const lit = isLit(down, up)
  return {
    down,
    up,
    connections,
    lit,
    // Motion is reserved for what is both busy and among the busiest. On a
    // router where EVERYTHING is trickle, nothing moves — which is the truth,
    // and quieter than eight pulses ranking a set of nothings.
    animated: animated && lit && down > 0,
    durationSec: dashDurationFor(down),
    width: lit ? ribbonWidthFor(down) : MIN_WIDTH,
  }
}

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
  // never depends on wire order for a visual decision. Ranking happens among
  // the LIT flows only: the top 8 of a trickling router are still trickle, and
  // animating them would put the motion back that the floor just removed.
  const placeable = frame.rules.filter((flow) => ribbonKey(flow) !== null)
  const ranked = placeable.filter((flow) => isLit(flow.down, flow.up)).sort((a, b) => b.down - a.down)
  const top = new Set(ranked.slice(0, topN).map(ribbonKey))

  const belowFloor = placeable.filter(
    (flow) => !isLit(flow.down, flow.up) && (flow.down > 0 || flow.up > 0 || flow.connections > 0),
  ).length

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
    belowFloor,
    at: frame.at,
  }
}
