import { describe, expect, test } from 'bun:test'
import {
  DNS_RULE_ACTION_TYPE_NAMES,
  actionOf,
  applyActionDefaults,
  pruneForeignFields,
  resolveDNSRuleActionFields,
  validateDNSRuleAction,
} from './dnsRuleActionFields'

/**
 * These pin the decision rules, not the curation data — a test asserting that
 * `client_subnet` sits at order 120 would just restate the file.
 *
 * The exceptions are the four defect regressions at the bottom. Each is a bug the
 * hand-written form shipped, and each is fixed structurally here rather than by a
 * patch, so a test is the only thing keeping the fix from being refactored away.
 */

describe('action names', () => {
  test('is exactly the four DNS rule actions', () => {
    expect(DNS_RULE_ACTION_TYPE_NAMES.map(String).sort()).toEqual([
      'predefined',
      'reject',
      'route',
      'route-options',
    ])
  })

  // _RuleAction (route rules) and _DNSRuleAction share the C.RuleActionType*
  // constant namespace but not the value set. A DNS rule naming one of these
  // fails to decode with "unknown DNS rule action".
  test('excludes the route-rule-only actions', () => {
    for (const routeOnly of ['direct', 'hijack-dns', 'sniff', 'resolve']) {
      expect(DNS_RULE_ACTION_TYPE_NAMES.map(String)).not.toContain(routeOnly)
    }
  })
})

describe('actionOf', () => {
  // DNSRuleAction.UnmarshalJSONContext rewrites "" to "route" before
  // dispatching, so a rule with no action IS a route rule.
  test('treats an omitted action as route', () => {
    expect(actionOf({ domain: ['a.example'] })).toBe('route')
    expect(actionOf({ action: '' })).toBe('route')
  })

  test('falls back to route for an unknown action rather than throwing', () => {
    expect(actionOf({ action: 'hijack-dns' })).toBe('route')
  })
})

describe('resolveDNSRuleActionFields', () => {
  const keysFor = (action: Parameters<typeof resolveDNSRuleActionFields>[0]) =>
    resolveDNSRuleActionFields(action).map((f) => f.key)

  test('route owns server; route-options does not', () => {
    expect(keysFor('route')).toContain('server')
    expect(keysFor('route-options')).not.toContain('server')
  })

  test('route-options is not empty — its four fields are the whole struct', () => {
    // The old form rendered nothing at all for this action.
    expect(keysFor('route-options').sort()).toEqual([
      'client_subnet',
      'disable_cache',
      'rewrite_ttl',
      'strategy',
    ])
  })

  test('reject exposes no_drop, which the old form never rendered', () => {
    expect(keysFor('reject')).toContain('no_drop')
  })

  test('predefined exposes answer/ns/extra, not just rcode', () => {
    expect(keysFor('predefined').sort()).toEqual(['answer', 'extra', 'ns', 'rcode'])
  })

  // Regression for the two namedKinds entries in cmd/gen-option-schema. Both Go
  // types lie about their wire format: DNSRCode is an int that marshals as
  // "NXDOMAIN", and DNSRecordOptions embeds dns.RR but marshals as a string.
  // Without those entries rcode renders as a number spinner and answer as a JSON
  // textarea of unusable objects.
  test('rcode is a string select, not a number', () => {
    const rcode = resolveDNSRuleActionFields('predefined').find((f) => f.key === 'rcode')
    expect(rcode?.kind).toBe('string')
    expect(rcode?.control).toBe('select')
    expect(rcode?.options).toContain('NXDOMAIN')
  })

  test('answer is a list of strings, so it renders as chips', () => {
    const answer = resolveDNSRuleActionFields('predefined').find((f) => f.key === 'answer')
    expect(answer?.kind).toBe('list')
    expect(answer?.item).toBe('string')
    expect(answer?.control).toBe('chips')
  })

  test('server uses the dns-server picker, not a free text box', () => {
    const server = resolveDNSRuleActionFields('route').find((f) => f.key === 'server')
    expect(server?.control).toBe('dns-server')
    expect(server?.tier).toBe('core')
  })

  // sing-box rejects anything outside these two with "unknown reject method".
  // The list once also offered success/refused/nxdomain.
  test('reject offers exactly the two methods sing-box accepts', () => {
    const method = resolveDNSRuleActionFields('reject').find((f) => f.key === 'method')
    expect(method?.options).toEqual(['default', 'drop'])
  })
})

describe('pruneForeignFields', () => {
  // DEFECT 1: the old save switch deleted method/no_drop/rcode but not server,
  // so route -> route-options emitted a field the struct has no place for.
  test('drops server when switching route -> route-options', () => {
    const pruned = pruneForeignFields(
      { action: 'route', server: 'dns-remote', strategy: 'ipv4_only' },
      'route-options',
    )
    expect(pruned).not.toHaveProperty('server')
    expect(pruned.strategy).toBe('ipv4_only')
    expect(pruned.action).toBe('route-options')
  })

  test('keeps match conditions, which belong to no action', () => {
    const pruned = pruneForeignFields(
      {
        action: 'route',
        server: 'dns-remote',
        domain: ['a.example'],
        domain_suffix: ['.b.example'],
        rule_set: ['geosite-cn'],
        ip_cidr: ['10.0.0.0/8'],
        clash_mode: 'Direct',
        invert: true,
      },
      'reject',
    )
    expect(pruned).not.toHaveProperty('server')
    expect(pruned.domain).toEqual(['a.example'])
    expect(pruned.domain_suffix).toEqual(['.b.example'])
    expect(pruned.rule_set).toEqual(['geosite-cn'])
    expect(pruned.ip_cidr).toEqual(['10.0.0.0/8'])
    expect(pruned.clash_mode).toBe('Direct')
    expect(pruned.invert).toBe(true)
  })

  // DEFECT 3: preservedFields was discarded whenever the action changed, so
  // changing a logical rule's action rewrote it as a flat default rule.
  test('keeps a logical rule logical across an action change', () => {
    const pruned = pruneForeignFields(
      {
        type: 'logical',
        mode: 'and',
        rules: [{ domain: ['a.example'] }, { domain_suffix: ['.b.example'] }],
        action: 'route',
        server: 'dns-remote',
      },
      'reject',
    )
    expect(pruned.type).toBe('logical')
    expect(pruned.mode).toBe('and')
    expect(pruned.rules).toHaveLength(2)
    expect(pruned).not.toHaveProperty('server')
  })

  // The denylist is derived from the inventories, so a key nothing has heard of
  // is kept by construction — the opposite of the EDITED_FIELDS allowlist, where
  // a field nobody remembered to list was destroyed on save.
  test('keeps a key no action has ever heard of', () => {
    const pruned = pruneForeignFields({ action: 'reject', some_future_matcher: 'x' }, 'reject')
    expect(pruned.some_future_matcher).toBe('x')
  })

  test('shared fields survive route <-> route-options in both directions', () => {
    const forward = pruneForeignFields(
      { action: 'route', strategy: 'ipv4_only', disable_cache: true, rewrite_ttl: 60 },
      'route-options',
    )
    expect(forward.strategy).toBe('ipv4_only')
    expect(forward.disable_cache).toBe(true)
    expect(forward.rewrite_ttl).toBe(60)

    const back = pruneForeignFields(forward, 'route')
    expect(back.strategy).toBe('ipv4_only')
    expect(back.rewrite_ttl).toBe(60)
  })
})

describe('applyActionDefaults', () => {
  // DEFECT 4: openEditRuleModal seeded NXDOMAIN for a missing rcode, but absent
  // means NOERROR — so opening an existing rule and pressing Update changed what
  // it did. Defaults apply to new records only, and the value is now NOERROR.
  test('seeds NOERROR for a new predefined rule, never NXDOMAIN', () => {
    expect(applyActionDefaults({}, 'predefined').rcode).toBe('NOERROR')
  })

  test('never overwrites a value an existing rule already carries', () => {
    expect(applyActionDefaults({ rcode: 'NXDOMAIN' }, 'predefined').rcode).toBe('NXDOMAIN')
  })

  test('seeds the reject method, which marshals away when it is "default"', () => {
    expect(applyActionDefaults({}, 'reject').method).toBe('default')
  })

  test('writes the discriminator to action, never to type', () => {
    // `type` on a DNS rule is the matcher shape. Writing the action there would
    // turn a logical rule into an unparseable one.
    const seeded = applyActionDefaults({ type: 'logical' }, 'reject')
    expect(seeded.action).toBe('reject')
    expect(seeded.type).toBe('logical')
  })
})

describe('validateDNSRuleAction', () => {
  test('route requires a server', () => {
    expect(validateDNSRuleAction({ action: 'route' })).toBe('dns.rules.form.errors.serverRequired')
    expect(validateDNSRuleAction({ action: 'route', server: 'dns-remote' })).toBeUndefined()
  })

  // "empty DNS route option action" — the struct's own UnmarshalJSON rejects an
  // all-zero value, which sing-box check reports as an opaque upstream string.
  test('route-options must set at least one field', () => {
    expect(validateDNSRuleAction({ action: 'route-options' })).toBe(
      'dns.rules.form.errors.routeOptionsEmpty',
    )
    expect(validateDNSRuleAction({ action: 'route-options', strategy: 'ipv4_only' })).toBeUndefined()
    // false counts as empty — an unticked switch is the absence of a setting.
    expect(validateDNSRuleAction({ action: 'route-options', disable_cache: false })).toBe(
      'dns.rules.form.errors.routeOptionsEmpty',
    )
    expect(validateDNSRuleAction({ action: 'route-options', disable_cache: true })).toBeUndefined()
  })

  // "no_drop is not available in current context".
  test('no_drop is rejected alongside method drop', () => {
    expect(validateDNSRuleAction({ action: 'reject', method: 'drop', no_drop: true })).toBe(
      'dns.rules.form.errors.noDropWithDrop',
    )
    expect(
      validateDNSRuleAction({ action: 'reject', method: 'default', no_drop: true }),
    ).toBeUndefined()
    expect(validateDNSRuleAction({ action: 'reject', method: 'drop' })).toBeUndefined()
  })

  test('a rule with no action at all is validated as a route rule', () => {
    expect(validateDNSRuleAction({ domain: ['a.example'] })).toBe(
      'dns.rules.form.errors.serverRequired',
    )
  })
})
