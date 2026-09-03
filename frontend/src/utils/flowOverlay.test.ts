import { describe, expect, test } from 'bun:test'
import { TOP_N, buildFlowOverlay, dashDurationFor, formatRate, ribbonWidthFor } from './flowOverlay'
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

  test('the top N is ranked across rule AND final flows together', () => {
    const rules = [
      ...Array.from({ length: TOP_N }, (_, i) => rule(i, 'x', 100)),
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
