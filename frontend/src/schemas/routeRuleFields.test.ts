import { describe, expect, test } from 'bun:test'
import {
  ROUTE_RULE_ACTION_TYPE_NAMES,
  actionOf,
  applyActionDefaults,
  pruneForeignFields,
  resolveRouteRuleActionFields,
  validateRouteRuleAction,
} from './routeRuleActionFields'
import {
  CONTENT_MATCHER_KEYS,
  MATCHER_GROUPS,
  resolveMatcherFields,
} from './routeRuleMatcherFields'
import { isRetired, type OptionVersionNote } from './optionSchema'
import { ROUTE_RULE_MATCHER_INVENTORY } from './routeRuleMatcherInventory.generated'

/**
 * Decision rules, not curation data. The exceptions are the regressions at the
 * bottom — each is a bug the hand-written form shipped, verified against a real
 * sing-box 1.13.11, and each is fixed structurally rather than patched.
 */

describe('action names', () => {
  test('is exactly the seven route actions', () => {
    expect(ROUTE_RULE_ACTION_TYPE_NAMES.map(String).sort()).toEqual([
      'direct',
      'hijack-dns',
      'reject',
      'resolve',
      'route',
      'route-options',
      'sniff',
    ])
  })

  // _RuleAction and _DNSRuleAction share the C.RuleActionType* namespace but not
  // the value set. `predefined` is DNS-only and decodes as "unknown rule action".
  test('excludes the DNS-only action', () => {
    expect(ROUTE_RULE_ACTION_TYPE_NAMES.map(String)).not.toContain('predefined')
  })

  // The old form offered six; `direct` was simply missing.
  test('includes direct, which the old form never offered', () => {
    expect(ROUTE_RULE_ACTION_TYPE_NAMES.map(String)).toContain('direct')
  })
})

describe('actionOf', () => {
  test('treats an omitted action as route', () => {
    expect(actionOf({ domain: ['a.example'] })).toBe('route')
    expect(actionOf({ action: '' })).toBe('route')
  })

  test('falls back to route for a DNS-only action rather than throwing', () => {
    expect(actionOf({ action: 'predefined' })).toBe('route')
  })
})

describe('resolveRouteRuleActionFields', () => {
  const keysFor = (a: Parameters<typeof resolveRouteRuleActionFields>[0]) =>
    resolveRouteRuleActionFields(a).map((f) => f.key)

  test('route owns outbound AND the embedded dial options', () => {
    const keys = keysFor('route')
    expect(keys).toContain('outbound')
    // The old form only offered these under route-options, but sing-box accepts
    // them on route too — RouteActionOptions embeds RawRouteOptionsActionOptions.
    expect(keys).toContain('udp_timeout')
    expect(keys).toContain('tls_fragment')
  })

  test('route-options has the dial options but no outbound', () => {
    expect(keysFor('route-options')).not.toContain('outbound')
    expect(keysFor('route-options')).toContain('override_address')
  })

  // DirectActionOptions IS DialerOptions, so reflection finds `detour` — but
  // sing-box answers "detour is not available in the current context".
  // Dropped by domain.ExcludedFields in the generator.
  test('direct excludes detour, which sing-box refuses here', () => {
    const keys = keysFor('direct')
    expect(keys).not.toContain('detour')
    expect(keys).toContain('bind_interface')
  })

  test('hijack-dns has no fields at all', () => {
    expect(keysFor('hijack-dns')).toEqual([])
  })

  test('reject exposes no_drop, which the old form never rendered', () => {
    expect(keysFor('reject')).toContain('no_drop')
  })

  test('resolve exposes the three fields the old form omitted', () => {
    const keys = keysFor('resolve')
    expect(keys).toContain('disable_cache')
    expect(keys).toContain('rewrite_ttl')
    expect(keys).toContain('client_subnet')
  })

  test('resolve reuses the dns-server picker', () => {
    const server = resolveRouteRuleActionFields('resolve').find((f) => f.key === 'server')
    expect(server?.control).toBe('dns-server')
  })
})

describe('pruneForeignFields', () => {
  // THE BUG. There was no pruning at all on the route path, so switching the
  // action shipped the previous action's fields. Against a running panel:
  //   {"action":"reject","outbound":"direct"} -> 200, outbound silently dropped
  //   {"action":"sniff","outbound":"direct"}  -> 400 unknown field "outbound"
  test('drops outbound when switching route -> reject', () => {
    const pruned = pruneForeignFields({ action: 'route', outbound: 'direct' }, 'reject')
    expect(pruned).not.toHaveProperty('outbound')
    expect(pruned.action).toBe('reject')
  })

  test('drops sniffer when switching sniff -> route', () => {
    const pruned = pruneForeignFields(
      { action: 'sniff', sniffer: ['http'], timeout: '300ms' },
      'route',
    )
    expect(pruned).not.toHaveProperty('sniffer')
    expect(pruned).not.toHaveProperty('timeout')
  })

  test('keeps every matcher, which belongs to no action', () => {
    const pruned = pruneForeignFields(
      {
        action: 'route',
        outbound: 'direct',
        domain: ['a.example'],
        rule_set: ['geosite-cn'],
        ip_cidr: ['10.0.0.0/8'],
        process_name: ['curl'],
        invert: true,
      },
      'reject',
    )
    expect(pruned).not.toHaveProperty('outbound')
    expect(pruned.domain).toEqual(['a.example'])
    expect(pruned.rule_set).toEqual(['geosite-cn'])
    expect(pruned.ip_cidr).toEqual(['10.0.0.0/8'])
    expect(pruned.process_name).toEqual(['curl'])
    expect(pruned.invert).toBe(true)
  })

  test('keeps a logical rule logical across an action change', () => {
    const pruned = pruneForeignFields(
      { type: 'logical', mode: 'and', rules: [{ domain: ['a'] }], action: 'route', outbound: 'x' },
      'reject',
    )
    expect(pruned.type).toBe('logical')
    expect(pruned.mode).toBe('and')
    expect(pruned.rules).toHaveLength(1)
    expect(pruned).not.toHaveProperty('outbound')
  })

  // route and route-options share all ten dial options, so switching between
  // them must not discard work.
  test('shared dial options survive route <-> route-options', () => {
    const forward = pruneForeignFields(
      { action: 'route', outbound: 'direct', udp_timeout: '5m', tls_fragment: true },
      'route-options',
    )
    expect(forward).not.toHaveProperty('outbound')
    expect(forward.udp_timeout).toBe('5m')
    expect(forward.tls_fragment).toBe(true)
  })
})

describe('applyActionDefaults', () => {
  test('seeds the reject method, which marshals away when it is "default"', () => {
    expect(applyActionDefaults({}, 'reject').method).toBe('default')
  })

  test('writes the discriminator to action, never to type', () => {
    // A route rule's `type` is its matcher shape. Writing the action there
    // would turn a logical rule into an unparseable one.
    const seeded = applyActionDefaults({ type: 'logical' }, 'reject')
    expect(seeded.action).toBe('reject')
    expect(seeded.type).toBe('logical')
  })
})

describe('validateRouteRuleAction', () => {
  test('route requires an outbound', () => {
    expect(validateRouteRuleAction({ action: 'route' })).toBe('route.rules.errors.outboundRequired')
    expect(validateRouteRuleAction({ action: 'route', outbound: 'direct' })).toBeUndefined()
  })

  // `sing-box check` PASSES for an outbound tag that does not exist, so the
  // client is the only place this can be caught.
  test('route rejects an outbound tag that does not exist', () => {
    expect(
      validateRouteRuleAction({ action: 'route', outbound: 'nope' }, ['direct', 'proxy']),
    ).toBe('route.rules.errors.outboundUnknown')
    expect(
      validateRouteRuleAction({ action: 'route', outbound: 'direct' }, ['direct', 'proxy']),
    ).toBeUndefined()
  })

  test('route-options must set at least one field', () => {
    expect(validateRouteRuleAction({ action: 'route-options' })).toBe(
      'route.rules.errors.routeOptionsEmpty',
    )
    expect(
      validateRouteRuleAction({ action: 'route-options', override_port: 443 }),
    ).toBeUndefined()
    // false counts as empty — an unticked switch is the absence of a setting.
    expect(validateRouteRuleAction({ action: 'route-options', udp_connect: false })).toBe(
      'route.rules.errors.routeOptionsEmpty',
    )
  })

  test('tls_fragment and tls_record_fragment are mutually exclusive', () => {
    expect(
      validateRouteRuleAction({
        action: 'route-options',
        tls_fragment: true,
        tls_record_fragment: true,
      }),
    ).toBe('route.rules.errors.tlsFragmentExclusive')
  })

  test('no_drop is rejected alongside method drop', () => {
    expect(validateRouteRuleAction({ action: 'reject', method: 'drop', no_drop: true })).toBe(
      'route.rules.errors.noDropWithDrop',
    )
    expect(
      validateRouteRuleAction({ action: 'reject', method: 'default', no_drop: true }),
    ).toBeUndefined()
  })

  test('hijack-dns needs nothing', () => {
    expect(validateRouteRuleAction({ action: 'hijack-dns' })).toBeUndefined()
  })
})

describe('matcher groups', () => {
  test('every group renders something', () => {
    for (const group of MATCHER_GROUPS) {
      expect(resolveMatcherFields(group).length).toBeGreaterThan(0)
    }
  })

  test('rule_set is its own group, not a content matcher', () => {
    // It is the ALTERNATIVE to the content matchers, so the mix warning is
    // about it. Grouping it with them would make the warning nonsensical.
    expect(resolveMatcherFields('ruleSet').map((f) => f.key)).toContain('rule_set')
    expect(CONTENT_MATCHER_KEYS).not.toContain('rule_set')
  })

  test('content is what the traffic IS', () => {
    expect(CONTENT_MATCHER_KEYS).toEqual(
      expect.arrayContaining(['domain', 'domain_suffix', 'domain_keyword', 'ip_cidr']),
    )
  })

  test('context is never folded by the rule-set choice', () => {
    // A rule set cannot express where traffic came from, so narrowing one by
    // port or network is correct, not an accidental intersection.
    const context = resolveMatcherFields('context').map((f) => f.key)
    expect(context).toEqual(expect.arrayContaining(['inbound', 'protocol', 'network', 'port']))
    for (const key of context) expect(CONTENT_MATCHER_KEYS).not.toContain(key)
  })

  // An ungrouped matcher shown beside the content matchers would be folded away
  // by the rule-set choice. A matcher a future sing-box adds must not vanish
  // just because a rule set is selected.
  test('an uncurated matcher falls into context, not content', () => {
    const grouped = new Set(MATCHER_GROUPS.flatMap((g) => resolveMatcherFields(g).map((f) => f.key)))
    for (const key of Object.keys(ROUTE_RULE_MATCHER_INVENTORY.default)) {
      expect(grouped.has(key)).toBe(true)
    }
  })

  test('every matcher appears in exactly one group', () => {
    const seen = new Map<string, number>()
    for (const group of MATCHER_GROUPS) {
      for (const f of resolveMatcherFields(group)) {
        seen.set(f.key, (seen.get(f.key) ?? 0) + 1)
      }
    }
    for (const [key, count] of seen) {
      expect(`${key}:${count}`).toBe(`${key}:1`)
    }
  })
})

/**
 * One matcher's version note. The generated inventory is `as const`, so a field
 * with no deprecation has no `removed` key at all and its literal type shares
 * nothing with OptionVersionNote — hence the widening.
 */
const noteFor = (key: string): OptionVersionNote =>
  ROUTE_RULE_MATCHER_INVENTORY.default[
    key as keyof typeof ROUTE_RULE_MATCHER_INVENTORY.default
  ] as OptionVersionNote

describe('retired matchers', () => {
  // sing-box REMOVED these in 1.12.0 and answers with a hard startup error:
  //   "geosite database is deprecated in sing-box 1.8.0 and removed in sing-box 1.12.0"
  // The old form promoted geosite and geoip as curated dropdowns with 12 and 7
  // options — all 19 values produced a rule that could not run.
  test.each(['geosite', 'geoip', 'source_geoip'])('%s is marked removed in 1.12.0', (key) => {
    expect(noteFor(key).removed).toBe('1.12.0')
  })

  test.each(['1.12.0', '1.12.12', '1.13.11'])(
    'the gate withholds them on installed %s',
    (installed) => {
      for (const key of ['geosite', 'geoip', 'source_geoip']) {
        expect(isRetired(noteFor(key), installed)).toBe(true)
      }
    },
  )

  test('the pre-1.10 match-source spelling is retired too', () => {
    expect(noteFor('rule_set_ipcidr_match_source').removed).toBe('1.11.0')
    expect(isRetired(noteFor('rule_set_ipcidr_match_source'), '1.13.11')).toBe(true)
    // The current spelling is not.
    expect(isRetired(noteFor('rule_set_ip_cidr_match_source'), '1.13.11')).toBe(false)
  })
})

describe('port vs port_range', () => {
  // `port` is Listable[uint16]; "8080-8090" is a decode failure. The old field
  // kept range syntax as a string and its placeholder advertised it.
  test('port is numeric and port_range is a separate string field', () => {
    const port = ROUTE_RULE_MATCHER_INVENTORY.default.port
    expect(port.kind).toBe('list')
    expect(port.item).toBe('number')

    const range = ROUTE_RULE_MATCHER_INVENTORY.default.port_range
    expect(range.kind).toBe('list')
    expect(range.item).toBe('string')
  })

  test('both are offered, so a range has somewhere to go', () => {
    const context = resolveMatcherFields('context').map((f) => f.key)
    expect(context).toContain('port')
    expect(context).toContain('port_range')
  })
})
