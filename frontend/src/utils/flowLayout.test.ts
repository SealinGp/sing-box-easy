import { describe, expect, test } from 'bun:test'
import { displayWidth, fitToBox, layoutRouteFlow, truncateToWidth } from './flowLayout'
import { buildRouteTopology } from './routeTopology'
import type { SingBoxConfig } from '../types/api'

const config = (): SingBoxConfig =>
  ({
    inbounds: [
      { type: 'tun', tag: 'tun-in' },
      { type: 'mixed', tag: 'mixed-in', listen_port: 7893 },
    ],
    outbounds: [
      { type: 'direct', tag: 'direct' },
      { type: 'urltest', tag: 'AI', outbounds: ['a', 'b'] },
      { type: 'selector', tag: 'Stream', outbounds: ['a'] },
      { type: 'shadowsocks', tag: 'a' },
      { type: 'shadowsocks', tag: 'b' },
    ],
    route: {
      rules: [
        { action: 'sniff' },
        { domain: ['ai.com'], outbound: 'AI' },
        { domain: ['tv.com'], outbound: 'Stream' },
        { domain: ['x.com'], outbound: 'AI' },
      ],
      final: 'direct',
    },
  }) as unknown as SingBoxConfig

const layout = () => layoutRouteFlow(buildRouteTopology(config()))

describe('layoutRouteFlow — geometry', () => {
  test('rule rows descend in config order and never overlap', () => {
    const { rules } = layout()

    expect(rules.map((row) => row.index)).toEqual([0, 1, 2, 3])
    for (let i = 1; i < rules.length; i += 1) {
      expect(rules[i]!.y).toBeGreaterThanOrEqual(rules[i - 1]!.y + rules[i - 1]!.height)
    }
  })

  test('the three columns do not overlap horizontally', () => {
    const { inbounds, rules, exits } = layout()

    const inboundRight = Math.max(...inbounds.map((box) => box.x + box.width))
    const ruleLeft = Math.min(...rules.map((box) => box.x))
    const ruleRight = Math.max(...rules.map((box) => box.x + box.width))
    const exitLeft = Math.min(...exits.map((box) => box.x))

    expect(ruleLeft).toBeGreaterThan(inboundRight)
    expect(exitLeft).toBeGreaterThan(ruleRight)
  })

  test('every box fits inside the reported canvas', () => {
    const result = layout()
    const boxes = [...result.inbounds, ...result.rules, ...result.exits, result.fallthrough]

    for (const box of boxes) {
      expect(box.x).toBeGreaterThanOrEqual(0)
      expect(box.y).toBeGreaterThanOrEqual(0)
      expect(box.x + box.width).toBeLessThanOrEqual(result.width)
      expect(box.y + box.height).toBeLessThanOrEqual(result.height)
    }
  })

  test('height grows with the rule count — the canvas is content-sized', () => {
    const short = layoutRouteFlow(
      buildRouteTopology({
        route: { rules: [{ domain: ['a'], outbound: 'x' }] },
      } as unknown as SingBoxConfig),
    )

    expect(layout().height).toBeGreaterThan(short.height)
  })
})

describe('layoutRouteFlow — ribbons', () => {
  test('a terminal rule gets exactly one ribbon to its exit', () => {
    const { ribbons } = layout()

    const fromRule1 = ribbons.filter((r) => r.kind === 'rule' && r.ruleIndex === 1)
    expect(fromRule1).toHaveLength(1)
    expect(fromRule1[0]!.exitId).toBe('AI')
  })

  test('a pass-through rule gets no ribbon — nothing leaves there', () => {
    const { ribbons } = layout()

    expect(ribbons.some((r) => r.kind === 'rule' && r.ruleIndex === 0)).toBe(false)
  })

  /**
   * The point of the diagram. Rules 1 and 3 name the same outbound, so both
   * ribbons must land on the same node — at different heights, so the fan
   * reads as "either of these", not as one line drawn twice.
   */
  test('converging ribbons share an exit and fan across its edge', () => {
    const { ribbons } = layout()

    const toAI = ribbons.filter((r) => r.exitId === 'AI' && r.kind === 'rule')
    expect(toAI.map((r) => r.ruleIndex)).toEqual([1, 3])
    expect(toAI[0]!.toY).not.toBe(toAI[1]!.toY)
  })

  test('every inbound feeds the ladder entry', () => {
    const { ribbons, inbounds } = layout()

    const entry = ribbons.filter((r) => r.kind === 'inbound')
    expect(entry).toHaveLength(inbounds.length)
    expect(new Set(entry.map((r) => `${r.toX},${r.toY}`)).size).toBe(1)
  })

  /**
   * The ladder is entered at the top. Centring the inbound column against the
   * full canvas instead parks it beside the middle rule, which reads as
   * "traffic enters halfway down" — the one thing an ordered ladder must not
   * say.
   */
  test('inbound ribbons land at the first rule, not at the canvas midpoint', () => {
    const { entry, rules, height } = layout()

    expect(entry.y).toBeLessThan(rules[0]!.y + rules[0]!.height)
    expect(entry.y).toBeLessThan(height / 2)
  })

  test('the inbound column hangs off the entry and stays on the canvas', () => {
    const many = layoutRouteFlow(
      buildRouteTopology({
        inbounds: [{ type: 'tun', tag: 'a' }],
        route: {
          rules: Array.from({ length: 30 }, (_, i) => ({ domain: [`d${i}.com`], outbound: 'x' })),
        },
      } as unknown as SingBoxConfig),
    )

    expect(many.inbounds[0]!.y).toBeGreaterThanOrEqual(0)
    expect(many.inbounds[0]!.y).toBeLessThan(many.rules[1]!.y + many.rules[1]!.height)
  })

  test('the fall-through is its own ribbon, marked so it can be dashed', () => {
    const { ribbons } = layout()

    const final = ribbons.filter((r) => r.kind === 'final')
    expect(final).toHaveLength(1)
    expect(final[0]!.exitId).toBe('direct')
  })

  test('paths are well-formed cubic curves', () => {
    for (const ribbon of layout().ribbons) {
      expect(ribbon.d).toMatch(/^M [\d.-]+ [\d.-]+ C /)
    }
  })
})

describe('layoutRouteFlow — exit ordering', () => {
  /**
   * Exits are placed by the average position of the rules pointing at them, so
   * ribbons stay roughly parallel. First-reference order (what the model
   * produces) would put `direct` — reached only by the fall-through — at the
   * top, crossing every other ribbon on the way down.
   */
  test('exits sort by the mean index of their incoming rules', () => {
    const { exits } = layout()

    expect(exits.map((exit) => exit.id)).toEqual(['AI', 'Stream', 'direct'])
  })

  test('a fall-through-only exit sinks to the bottom', () => {
    const { exits } = layout()

    expect(exits[exits.length - 1]!.id).toBe('direct')
  })
})

describe('text fitting', () => {
  test('counts CJK and emoji as double width', () => {
    expect(displayWidth('abc')).toBe(3)
    expect(displayWidth('直连')).toBe(4)
    expect(displayWidth('🤖')).toBe(2)
  })

  test('leaves a string that fits untouched', () => {
    expect(truncateToWidth('short', 10)).toBe('short')
  })

  test('clips to the budget and marks the clip', () => {
    expect(truncateToWidth('abcdefghij', 5)).toBe('abcd…')
  })

  test('never splits a double-width glyph across the budget', () => {
    // 4 units of budget: '直' + '连' is exactly 4, but the ellipsis needs one,
    // so only '直' survives.
    expect(truncateToWidth('直连节点', 4)).toBe('直…')
  })
})

describe('fitToBox', () => {
  test('fits by pixel width at the font size it will be drawn at', () => {
    // 100px at 12px font ≈ 15 half-em units — "abcdefghijklmno" fits, longer clips.
    expect(fitToBox('abcdefghijklmno', 100, 12)).toBe('abcdefghijklmno')
    expect(fitToBox('abcdefghijklmnopqrstuvwxyz', 100, 12)).toMatch(/…$/)
  })

  test('a larger font fits fewer characters in the same box', () => {
    const small = fitToBox('abcdefghijklmnopqrst', 100, 9)
    const large = fitToBox('abcdefghijklmnopqrst', 100, 16)

    expect(displayWidth(small)).toBeGreaterThan(displayWidth(large))
  })
})
