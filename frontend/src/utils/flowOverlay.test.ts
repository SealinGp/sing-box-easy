import { describe, expect, test } from 'bun:test'
import {
  HEAT_STEPS,
  RANK_N,
  RATE_FLOOR,
  TOP_N,
  buildFlowOverlay,
  dashDurationFor,
  formatRate,
  heatFor,
  isLit,
  ribbonWidthFor,
} from './flowOverlay'
import type { TrafficFrame, TrafficRuleFlow } from '../types/trafficFlow'

const rule = (index: number, exit: string, down: number, extra: Partial<TrafficRuleFlow> = {}): TrafficRuleFlow => ({
  kind: 'rule',
  index,
  exit,
  down,
  up: down / 10,
  connections: 1,
  hosts: [{ host: `h${index}.example`, down }],
  ...extra,
})

const frame = (rules: TrafficRuleFlow[], extra: Partial<TrafficFrame> = {}): TrafficFrame => ({
  at: 1_700_000_000_000,
  totals: { down: rules.reduce((s, r) => s + r.down, 0), up: 0, connections: rules.length, all: rules.length, closed: 0 },
  inbounds: [{ tag: 'tun-in', down: 5000, up: 100, connections: 3 }],
  rules,
  exits: [{ tag: '🤖 AI', down: 4000, up: 400, connections: 2, via: [{ tag: '新加坡03', down: 3000, connections: 1 }] }],
  sources: [{ ip: '192.168.9.20', down: 5000, up: 100, connections: 3 }],
  filtered: false,
  unmatched: 0,
  ...extra,
})

describe('buildFlowOverlay — ribbons', () => {
  test('a rule flow lights the ribbon with that rule index', () => {
    const overlay = buildFlowOverlay(frame([rule(8, '🤖 AI', 4000)]))

    const ribbon = overlay.ribbons.get('rule:8')
    expect(ribbon).toBeDefined()
    expect(ribbon!.down).toBe(4000)
    expect(ribbon!.animated).toBe(true)
  })

  test('the fall-through lights the final ribbon', () => {
    const overlay = buildFlowOverlay(frame([rule(-1, '➡️ 直连', 200, { kind: 'final' })]))

    expect(overlay.ribbons.get('final')?.down).toBe(200)
  })

  test('an inbound flow lights the inbound ribbon by tag', () => {
    const overlay = buildFlowOverlay(frame([]))

    expect(overlay.ribbons.get('in:tun-in')?.connections).toBe(3)
  })

  /**
   * The noise budget. A busy router has hundreds of connections through
   * dozens of rules; animating them all is a screen of moving pulses that says
   * nothing. Only the top N by download rate move — the rest are lit but
   * still, so the shape of the traffic is visible and the motion is meaning.
   */
  test('only the top N rule ribbons animate', () => {
    const rules = Array.from({ length: TOP_N + 5 }, (_, i) => rule(i, 'x', 1000 * (TOP_N + 5 - i)))
    const overlay = buildFlowOverlay(frame(rules))

    const animated = [...overlay.ribbons.entries()].filter(([k, v]) => k.startsWith('rule:') && v.animated)
    expect(animated).toHaveLength(TOP_N)
    // The slowest ones are lit, not animated.
    expect(overlay.ribbons.get(`rule:${TOP_N + 4}`)!.animated).toBe(false)
    expect(overlay.ribbons.get(`rule:${TOP_N + 4}`)!.connections).toBe(1)
  })

  test('a ribbon with no bytes moving is lit but never animated', () => {
    const overlay = buildFlowOverlay(frame([rule(3, 'x', 0)]))

    expect(overlay.ribbons.get('rule:3')!.animated).toBe(false)
  })

  // Every rate here clears RATE_FLOOR on purpose: this test is about RANKING,
  // and a fixture below the floor would pass or fail for the other reason.
  test('the top N is ranked across rule AND final flows together', () => {
    const rules = [
      ...Array.from({ length: TOP_N }, (_, i) => rule(i, 'x', 100_000)),
      rule(-1, 'direct', 999_999, { kind: 'final' }),
    ]
    const overlay = buildFlowOverlay(frame(rules))

    expect(overlay.ribbons.get('final')!.animated).toBe(true)
    expect(overlay.animatedRules).toBe(TOP_N)
  })
})

describe('buildFlowOverlay — exits and unmatched', () => {
  test('exits are keyed by tag and carry their via list', () => {
    const overlay = buildFlowOverlay(frame([]))

    const exit = overlay.exits.get('🤖 AI')
    expect(exit?.connections).toBe(2)
    expect(exit?.via[0]?.tag).toBe('新加坡03')
  })

  /**
   * A rule string sing-box reports that the running rule list does not
   * contain cannot be placed on a row. It still reached an exit, so the exit
   * lights, and the flow is listed so the card can say "N matched by exit
   * only" instead of silently dropping bytes from the picture.
   */
  test('an unmatched flow lights no ribbon and is listed separately', () => {
    const overlay = buildFlowOverlay(
      frame([rule(-1, 'Claude', 50, { kind: 'unmatched', rule: 'domain=old => route(Claude)' })], { unmatched: 1 }),
    )

    expect([...overlay.ribbons.keys()].some((k) => k.startsWith('rule:'))).toBe(false)
    expect(overlay.unmatched).toHaveLength(1)
    expect(overlay.unmatched[0]!.exit).toBe('Claude')
  })

  test('rule rows can be looked up by index for lamps and tooltips', () => {
    const overlay = buildFlowOverlay(frame([rule(11, '🤖 AI', 700)]))

    expect(overlay.rules.get(11)?.hosts[0]?.host).toBe('h11.example')
  })
})

describe('speed → motion', () => {
  /**
   * Log scale, because 10 KB/s and 10 MB/s must BOTH read as motion. Linear
   * makes everything below the fastest edge look stopped.
   */
  test('faster traffic means a faster pulse, on a log scale', () => {
    const kb = dashDurationFor(1_000)
    const tenKb = dashDurationFor(10_000)
    const mb = dashDurationFor(1_000_000)
    const tenMb = dashDurationFor(10_000_000)

    expect(kb).toBeGreaterThan(tenKb)
    expect(tenKb).toBeGreaterThan(mb)
    expect(mb).toBeGreaterThan(tenMb)
    // Each decade shortens the cycle by the same factor.
    expect(kb / tenKb).toBeCloseTo(tenKb / dashDurationFor(100_000), 1)
  })

  test('the cycle is clamped so a trickle still visibly moves and a torrent is not a blur', () => {
    expect(dashDurationFor(1)).toBeLessThanOrEqual(4)
    expect(dashDurationFor(1e12)).toBeGreaterThanOrEqual(0.3)
  })

  test('ribbon width grows with rate but is bounded', () => {
    expect(ribbonWidthFor(0)).toBeGreaterThan(0)
    expect(ribbonWidthFor(1e9)).toBeLessThanOrEqual(6)
    expect(ribbonWidthFor(1e6)).toBeGreaterThan(ribbonWidthFor(1e3))
  })
})

describe('formatRate', () => {
  test('rounds away the floating-point noise a derived rate carries', () => {
    expect(formatRate(391.55309828159835)).toBe('392 B')
    expect(formatRate(0.4)).toBe('0 B')
  })

  test('never shows a negative rate', () => {
    expect(formatRate(-5)).toBe('0 B')
  })
})

/**
 * The noise floor. Without it, "carrying traffic" means "appears in the frame",
 * and on a real router one DNS lookup per rule is enough to light the whole
 * ladder — at which point lighting stops distinguishing anything.
 */
describe('buildFlowOverlay — the rate floor', () => {
  test('a flow at the floor is lit; one below it is not', () => {
    const overlay = buildFlowOverlay(frame([rule(1, 'x', RATE_FLOOR), rule(2, 'x', 10)]))

    expect(overlay.ribbons.get('rule:1')!.lit).toBe(true)
    expect(overlay.ribbons.get('rule:2')!.lit).toBe(false)
  })

  // Download alone would call a 4 MB/s upload idle. Ranking stays by download;
  // only the busy/idle question is answered on the total.
  test('upload counts toward the floor', () => {
    expect(isLit(0, RATE_FLOOR)).toBe(true)
    expect(isLit(RATE_FLOOR / 2, RATE_FLOOR / 2)).toBe(true)
    expect(isLit(RATE_FLOOR - 1, 0)).toBe(false)
  })

  test('a trickling flow never animates, however it ranks', () => {
    const overlay = buildFlowOverlay(frame([rule(1, 'x', 50), rule(2, 'x', 10)]))

    expect(overlay.animatedRules).toBe(0)
    expect(overlay.ribbons.get('rule:1')!.animated).toBe(false)
  })

  // The whole point of ranking among lit flows only: on an idle router the
  // "top 8" are eight nothings, and animating them puts back the motion the
  // floor was added to remove.
  test('ranking ignores flows under the floor', () => {
    const rules = [
      rule(0, 'x', 100_000),
      ...Array.from({ length: TOP_N + 4 }, (_, i) => rule(i + 1, 'x', 20)),
    ]
    const overlay = buildFlowOverlay(frame(rules))

    expect(overlay.animatedRules).toBe(1)
    expect(overlay.ribbons.get('rule:0')!.animated).toBe(true)
  })

  test('a suppressed flow keeps its numbers — dimmed, not dropped', () => {
    const overlay = buildFlowOverlay(frame([rule(4, 'x', 300, { connections: 12 })]))

    const ribbon = overlay.ribbons.get('rule:4')!
    expect(ribbon.down).toBe(300)
    expect(ribbon.connections).toBe(12)
    expect(overlay.rules.get(4)!.hosts).toHaveLength(1)
  })

  // A quiet diagram has to explain itself, or it reads as a broken stream.
  test('belowFloor counts the suppressed rows, and only those', () => {
    const overlay = buildFlowOverlay(
      frame([rule(1, 'x', 100_000), rule(2, 'x', 40), rule(3, 'x', 0, { connections: 0 })]),
    )

    expect(overlay.belowFloor).toBe(1)
  })

  test('a row holding connections but moving nothing still counts as suppressed', () => {
    const overlay = buildFlowOverlay(frame([rule(5, 'x', 0, { connections: 3 })]))

    expect(overlay.belowFloor).toBe(1)
    expect(overlay.ribbons.get('rule:5')!.lit).toBe(false)
  })
})

/**
 * Heat: colour as a function of ABSOLUTE download rate. Relative to the frame's
 * maximum, a 50 KB/s flow on an idle router would read as blazing, and the
 * whole point is to spot an extreme without first reading every number.
 */
describe('heat', () => {
  test('is zero under the floor, then one tier per decade of download', () => {
    expect(heatFor(0, 0)).toBe(0)
    expect(heatFor(RATE_FLOOR - 1, 0)).toBe(0)
    expect(heatFor(RATE_FLOOR, 0)).toBe(1)
    expect(heatFor(HEAT_STEPS[0]! - 1, 0)).toBe(1)
    expect(heatFor(HEAT_STEPS[0]!, 0)).toBe(2)
    expect(heatFor(HEAT_STEPS[1]!, 0)).toBe(3)
    expect(heatFor(HEAT_STEPS[2]!, 0)).toBe(4)
    expect(heatFor(1e12, 0)).toBe(4)
  })

  // Upload lights a flow (see the floor) but it is DOWNLOAD heat: a backup
  // pushing 5 MB/s up with nothing coming down is warm, not blazing.
  test('a flow lit by upload alone is at the base tier', () => {
    expect(heatFor(0, 5_000_000)).toBe(1)
  })

  test('every ribbon and exit carries its heat', () => {
    const overlay = buildFlowOverlay(
      frame([rule(3, '🤖 AI', 2_000_000)], {
        exits: [{ tag: '🤖 AI', down: 2_000_000, up: 0, connections: 1, via: [] }],
        inbounds: [{ tag: 'tun-in', down: 500, up: 0, connections: 1 }],
      }),
    )

    expect(overlay.ribbons.get('rule:3')!.heat).toBe(3)
    expect(overlay.exits.get('🤖 AI')!.heat).toBe(3)
    expect(overlay.ribbons.get('in:tun-in')!.heat).toBe(0)
  })
})

/**
 * Rank: the anchor points. With eight ribbons moving, "which is the busiest"
 * still meant reading eight numbers; the top few now say so themselves.
 */
describe('rank', () => {
  test('the busiest lit rule flows are ranked 1..RANK_N, the rest are not', () => {
    const rules = Array.from({ length: RANK_N + 3 }, (_, i) => rule(i, 'x', 1_000_000 - i * 1000))
    const overlay = buildFlowOverlay(frame(rules))

    for (let i = 0; i < RANK_N; i += 1) expect(overlay.ribbons.get(`rule:${i}`)!.rank).toBe(i + 1)
    expect(overlay.ribbons.get(`rule:${RANK_N}`)!.rank).toBeNull()
  })

  test('ranks are by download, whatever order the wire sent', () => {
    const overlay = buildFlowOverlay(frame([rule(1, 'x', 5_000), rule(2, 'x', 900_000), rule(3, 'x', 50_000)]))

    expect(overlay.ribbons.get('rule:2')!.rank).toBe(1)
    expect(overlay.ribbons.get('rule:3')!.rank).toBe(2)
    expect(overlay.ribbons.get('rule:1')!.rank).toBe(3)
  })

  test('a flow under the floor is never ranked, even when it is all there is', () => {
    const overlay = buildFlowOverlay(frame([rule(1, 'x', 50)]))

    expect(overlay.ribbons.get('rule:1')!.rank).toBeNull()
  })

  test('the fall-through competes for a rank like any rule', () => {
    const overlay = buildFlowOverlay(frame([rule(1, 'x', 5_000), rule(-1, 'direct', 900_000, { kind: 'final' })]))

    expect(overlay.ribbons.get('final')!.rank).toBe(1)
    expect(overlay.ribbons.get('rule:1')!.rank).toBe(2)
  })

  test('inbound ribbons are not ranked — the ladder is what is being compared', () => {
    const overlay = buildFlowOverlay(frame([]))

    expect(overlay.ribbons.get('in:tun-in')!.rank).toBeNull()
  })
})
