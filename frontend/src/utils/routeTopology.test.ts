import { describe, expect, test } from 'bun:test'
import { buildRouteTopology, formatCondition, MAX_CONDITION_VALUES } from './routeTopology'
import type { SingBoxConfig } from '../types/api'

/**
 * A cut-down stand-in for `bin/config.json` — the shapes that matter, none of
 * the 200 nodes. Three rules route to `🤖 AI` by different conditions, which is
 * the case the diagram exists to make visible.
 */
const productionish = (): SingBoxConfig =>
  ({
    inbounds: [
      { type: 'tun', tag: 'tun-in' },
      { type: 'mixed', tag: 'mixed-in', listen_port: 7893 },
      { type: 'direct', tag: 'dns-in', listen_port: 53 },
    ],
    outbounds: [
      { type: 'urltest', tag: '🤖 AI', outbounds: ['a', 'b', 'c'] },
      { type: 'direct', tag: '➡️ 直连' },
      { type: 'block', tag: 'block_out' },
      { type: 'selector', tag: 'Google', outbounds: ['🤖 AI', '➡️ 直连'] },
      { type: 'shadowsocks', tag: 'a' },
      { type: 'shadowsocks', tag: 'b' },
      { type: 'shadowsocks', tag: 'c' },
    ],
    route: {
      rules: [
        { inbound: ['dns-in'], action: 'hijack-dns' },
        { action: 'sniff' },
        { ip_cidr: ['192.168.8.0/24'], outbound: '➡️ 直连' },
        { domain_suffix: ['stripe.com', 'binance.us'], outbound: '🤖 AI' },
        { rule_set: 'geosite-google', outbound: 'Google' },
        { rule_set: 'sea-rulesets-ai', outbound: '🤖 AI' },
        { rule_set: ['geoip-google', 'geosite-tiktok'], outbound: '🤖 AI' },
        { rule_set: 'sea-rulesets-reject', outbound: 'block_out' },
      ],
      final: '➡️ 直连',
    },
  }) as unknown as SingBoxConfig

describe('buildRouteTopology — inbounds', () => {
  test('carries tag, type and listen port across', () => {
    const { inbounds } = buildRouteTopology(productionish())

    expect(inbounds).toEqual([
      { tag: 'tun-in', type: 'tun', listenPort: undefined },
      { tag: 'mixed-in', type: 'mixed', listenPort: 7893 },
      { tag: 'dns-in', type: 'direct', listenPort: 53 },
    ])
  })

  test('an inbound with no tag is still listed, so the count stays honest', () => {
    const { inbounds } = buildRouteTopology({
      inbounds: [{ type: 'mixed' }],
    } as unknown as SingBoxConfig)

    expect(inbounds).toHaveLength(1)
    expect(inbounds[0]!.tag).toBe('')
  })
})

describe('buildRouteTopology — exits converge', () => {
  /**
   * The whole reason for the card. Rules 3, 5 and 6 name `🤖 AI` through three
   * unrelated conditions; sing-box has one outbound, so the diagram must draw
   * one node with three ribbons rather than three nodes that happen to share a
   * label.
   */
  test('rules naming the same outbound collapse onto one exit', () => {
    const { exits } = buildRouteTopology(productionish())

    const ai = exits.find((exit) => exit.id === '🤖 AI')
    expect(ai).toBeDefined()
    expect(ai!.ruleIndices).toEqual([3, 5, 6])
    expect(ai!.kind).toBe('outbound')
    expect(ai!.type).toBe('urltest')
    expect(ai!.memberCount).toBe(3)
  })

  test('exits appear in first-reference order', () => {
    const { exits } = buildRouteTopology(productionish())

    expect(exits.map((exit) => exit.id)).toEqual([
      'hijack-dns',
      '➡️ 直连',
      '🤖 AI',
      'Google',
      'block_out',
    ])
  })

  test('a leaf outbound reports no member count', () => {
    const { exits } = buildRouteTopology(productionish())

    expect(exits.find((exit) => exit.id === '➡️ 直连')!.memberCount).toBeUndefined()
  })
})

describe('buildRouteTopology — rules', () => {
  test('a route rule points at its exit and terminates', () => {
    const { rules } = buildRouteTopology(productionish())

    expect(rules[3]).toMatchObject({
      index: 3,
      action: 'route',
      terminal: true,
      exitId: '🤖 AI',
      reachable: true,
    })
  })

  test('an omitted action reads as route — what sing-box defaults to', () => {
    const { rules } = buildRouteTopology({
      route: { rules: [{ domain: ['example.com'], outbound: 'proxy' }] },
      outbounds: [{ type: 'direct', tag: 'proxy' }],
    } as unknown as SingBoxConfig)

    expect(rules[0]!.action).toBe('route')
    expect(rules[0]!.terminal).toBe(true)
  })

  /**
   * `sniff` matches, changes what the rules below it see, and hands over. It
   * has no exit, and drawing one would claim traffic leaves there.
   */
  test('sniff is a pass-through with no exit', () => {
    const { rules } = buildRouteTopology(productionish())

    expect(rules[1]).toMatchObject({ action: 'sniff', terminal: false, exitId: null })
  })

  /**
   * Mirrors `routeprobe/rule_meta.go:39-53`: sing-box's route.go has no case
   * for `direct`, so a rule carrying it falls through. Correcting that here
   * would describe a config that is not the one in force.
   */
  test('the direct ACTION does not terminate, matching the engine', () => {
    const { rules } = buildRouteTopology({
      route: { rules: [{ domain: ['a.com'], action: 'direct' }] },
    } as unknown as SingBoxConfig)

    expect(rules[0]).toMatchObject({ action: 'direct', terminal: false, exitId: null })
  })

  test('hijack-dns terminates into a synthetic exit', () => {
    const { rules, exits } = buildRouteTopology(productionish())

    expect(rules[0]).toMatchObject({ terminal: true, exitId: 'hijack-dns' })
    expect(exits.find((exit) => exit.id === 'hijack-dns')!.kind).toBe('hijack-dns')
  })

  test('reject terminates into a synthetic exit carrying its method', () => {
    const { rules, exits } = buildRouteTopology({
      route: { rules: [{ domain: ['ads.com'], action: 'reject', method: 'drop' }] },
    } as unknown as SingBoxConfig)

    expect(rules[0]).toMatchObject({ terminal: true, exitId: 'reject' })
    const reject = exits.find((exit) => exit.id === 'reject')!
    expect(reject.kind).toBe('reject')
    expect(reject.detail).toBe('drop')
  })

  test('an inbound matcher is recorded as scope', () => {
    const { rules } = buildRouteTopology(productionish())

    expect(rules[0]!.scopedInbounds).toEqual(['dns-in'])
    expect(rules[1]!.scopedInbounds).toEqual([])
  })

  test('a scalar matcher is coerced to an array, as sing-box accepts both', () => {
    const { rules } = buildRouteTopology(productionish())

    const ruleSet = rules[4]!.conditions.find((c) => c.key === 'rule_set')!
    expect(ruleSet.values).toEqual(['geosite-google'])
  })

  test('invert is flagged rather than listed as a condition', () => {
    const { rules } = buildRouteTopology({
      route: { rules: [{ domain: ['a.com'], invert: true, outbound: 'x' }] },
    } as unknown as SingBoxConfig)

    expect(rules[0]!.inverted).toBe(true)
    expect(rules[0]!.conditions.map((c) => c.key)).toEqual(['domain'])
  })
})

describe('buildRouteTopology — missing tags', () => {
  /**
   * sing-box only reports this at START, not from `sing-box check`, so a typo
   * here is otherwise invisible until the service refuses to come up.
   */
  test('an outbound no tag defines is marked missing', () => {
    const { exits } = buildRouteTopology({
      outbounds: [{ type: 'direct', tag: 'direct' }],
      route: { rules: [{ domain: ['a.com'], outbound: 'typo-proxy' }], final: 'direct' },
    } as unknown as SingBoxConfig)

    expect(exits.find((exit) => exit.id === 'typo-proxy')!.kind).toBe('missing')
  })

  test('an endpoint satisfies an outbound reference', () => {
    const { exits } = buildRouteTopology({
      endpoints: [{ type: 'wireguard', tag: 'wg-us-exit-1' }],
      route: { rules: [{ domain: ['a.com'], outbound: 'wg-us-exit-1' }] },
    } as unknown as SingBoxConfig)

    const exit = exits.find((e) => e.id === 'wg-us-exit-1')!
    expect(exit.kind).toBe('outbound')
    expect(exit.type).toBe('wireguard')
  })
})

describe('buildRouteTopology — fall-through', () => {
  test('route.final names the exit', () => {
    const { fallthrough, exits } = buildRouteTopology(productionish())

    expect(fallthrough).toEqual({ exitId: '➡️ 直连', source: 'final' })
    expect(exits.find((exit) => exit.id === '➡️ 直连')!.isFinal).toBe(true)
  })

  /**
   * Three cases sing-box conflates under one word. Reporting "final" for all of
   * them sends someone hunting for a key that is not in their config —
   * `routeprobe/rule_meta.go:70-78` makes the same distinction.
   */
  test('no final falls back to the FIRST outbound', () => {
    const { fallthrough } = buildRouteTopology({
      outbounds: [
        { type: 'urltest', tag: 'auto' },
        { type: 'direct', tag: 'direct' },
      ],
      route: { rules: [] },
    } as unknown as SingBoxConfig)

    expect(fallthrough).toEqual({ exitId: 'auto', source: 'first_outbound' })
  })

  test('no outbounds at all means sing-box synthesises a direct one', () => {
    const { fallthrough, exits } = buildRouteTopology({
      route: { rules: [] },
    } as unknown as SingBoxConfig)

    expect(fallthrough.source).toBe('implicit_direct')
    expect(exits.find((exit) => exit.id === fallthrough.exitId)!.kind).toBe('implicit')
  })

  test('an exit only the fall-through reaches still gets a node', () => {
    const { exits } = buildRouteTopology({
      outbounds: [
        { type: 'direct', tag: 'direct' },
        { type: 'urltest', tag: 'auto' },
      ],
      route: { rules: [{ domain: ['a.com'], outbound: 'auto' }], final: 'direct' },
    } as unknown as SingBoxConfig)

    const final = exits.find((exit) => exit.id === 'direct')!
    expect(final.ruleIndices).toEqual([])
    expect(final.isFinal).toBe(true)
  })
})

describe('buildRouteTopology — unreachable rules', () => {
  /**
   * A terminal rule with no conditions matches everything
   * (`rule_abstract.go:55`), so every rule below it is dead config. Nothing in
   * the panel says so today.
   */
  test('rules below a condition-less terminal rule are unreachable', () => {
    const { rules } = buildRouteTopology({
      outbounds: [{ type: 'direct', tag: 'direct' }],
      route: {
        rules: [
          { domain: ['a.com'], outbound: 'direct' },
          { outbound: 'direct' },
          { domain: ['b.com'], outbound: 'direct' },
        ],
      },
    } as unknown as SingBoxConfig)

    expect(rules.map((rule) => rule.reachable)).toEqual([true, true, false])
    expect(rules[1]!.catchAll).toBe(true)
  })

  test('a condition-less NON-terminal rule swallows nothing', () => {
    const { rules } = buildRouteTopology(productionish())

    // `{ "action": "sniff" }` — this repo's own config has exactly that.
    expect(rules[1]!.catchAll).toBe(false)
    expect(rules.every((rule) => rule.reachable)).toBe(true)
  })

  test('an inverted condition-less rule is not a catch-all', () => {
    const { rules } = buildRouteTopology({
      route: { rules: [{ invert: true, outbound: 'x' }, { domain: ['a.com'], outbound: 'x' }] },
    } as unknown as SingBoxConfig)

    expect(rules[0]!.catchAll).toBe(false)
    expect(rules[1]!.reachable).toBe(true)
  })
})

describe('buildRouteTopology — degenerate input', () => {
  test('a null config yields an empty topology rather than throwing', () => {
    const topology = buildRouteTopology(null)

    expect(topology.inbounds).toEqual([])
    expect(topology.rules).toEqual([])
    expect(topology.fallthrough.source).toBe('implicit_direct')
  })

  test('a rules array containing junk is skipped, not fatal', () => {
    const { rules } = buildRouteTopology({
      route: { rules: [null, 'nonsense', { domain: ['a.com'], outbound: 'x' }] },
    } as unknown as SingBoxConfig)

    expect(rules).toHaveLength(1)
    expect(rules[0]!.index).toBe(2)
  })
})

describe('formatCondition', () => {
  test('spells out values up to the cap', () => {
    expect(formatCondition({ key: 'domain', values: ['a.com', 'b.com'] })).toBe('a.com, b.com')
  })

  test('collapses the tail to a count', () => {
    const values = Array.from({ length: MAX_CONDITION_VALUES + 4 }, (_, i) => `v${i}`)

    expect(formatCondition({ key: 'domain', values })).toBe(
      `${values.slice(0, MAX_CONDITION_VALUES).join(', ')} +4`,
    )
  })

  test('a boolean matcher renders as its own name', () => {
    expect(formatCondition({ key: 'ip_is_private', values: [] })).toBe('ip_is_private')
  })
})
