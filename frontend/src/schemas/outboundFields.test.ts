import { describe, expect, test } from 'bun:test'
import {
  OUTBOUND_GROUP_TYPES,
  OUTBOUND_TYPE_NAMES,
  OUTBOUND_TYPE_NOTES,
  OUTBOUND_TYPES_NEEDING_SERVER,
  pruneForType,
  resolveOutboundFields,
} from './outboundFields'
import { compareVersions, isRetired, isDeprecatedIn } from './optionSchema'

describe('outbound type names', () => {
  // The form kept its own hardcoded list of 17 while the registry built 20.
  test('includes the three the hardcoded list omitted', () => {
    for (const type of ['anytls', 'dns', 'shadowsocksr']) {
      expect(OUTBOUND_TYPE_NAMES).toContain(type as never)
    }
  })

  test('groups are derived, not hand-listed', () => {
    expect([...OUTBOUND_GROUP_TYPES].sort()).toEqual(['selector', 'urltest'])
  })

  test('terminal types need no server', () => {
    for (const type of ['direct', 'block', 'dns', 'selector', 'urltest', 'tor']) {
      expect(OUTBOUND_TYPES_NEEDING_SERVER).not.toContain(type as never)
    }
    expect(OUTBOUND_TYPES_NEEDING_SERVER).toContain('shadowsocks' as never)
  })
})

describe('field-less types', () => {
  // option.StubOptions is `struct{}`. These are behaviours, not configurations.
  test('block and dns resolve to no fields without erroring', () => {
    expect(resolveOutboundFields('block')).toEqual([])
    expect(resolveOutboundFields('dns')).toEqual([])
  })
})

describe('resolveOutboundFields', () => {
  test('promotes credentials the old form could not edit', () => {
    // socks/http matched "needs a server" but were excluded from the password
    // predicate, so an authenticating proxy could not be configured at all.
    const socks = resolveOutboundFields('socks')
    expect(socks.find((f) => f.key === 'username')).toBeDefined()
    expect(socks.find((f) => f.key === 'password')).toBeDefined()
  })

  test('hysteria gets the three fields it cannot start without', () => {
    const hysteria = resolveOutboundFields('hysteria')
    for (const key of ['auth_str', 'up_mbps', 'down_mbps']) {
      expect(hysteria.find((f) => f.key === key)?.tier).toBe('core')
    }
  })

  test('method is a fixed vocabulary, not free text', () => {
    const method = resolveOutboundFields('shadowsocks').find((f) => f.key === 'method')
    expect(method?.control).toBe('select')
    expect(method?.options).toContain('aes-128-gcm')
  })

  test('groups use the dedicated member controls', () => {
    const selector = resolveOutboundFields('selector')
    expect(selector.find((f) => f.key === 'outbounds')?.control).toBe('outbound-list')
    // `default` must be one of `outbounds`; a plain select could offer anything.
    expect(selector.find((f) => f.key === 'default')?.control).toBe('outbound-member')
  })

  test('urltest exposes tolerance, which only the rules engine could set', () => {
    expect(resolveOutboundFields('urltest').find((f) => f.key === 'tolerance')).toBeDefined()
  })

  test('protocol extras are reachable, as advanced', () => {
    const vless = resolveOutboundFields('vless')
    expect(vless.find((f) => f.key === 'flow')).toBeDefined()
    expect(vless.find((f) => f.key === 'tls')).toBeDefined()
    expect(vless.find((f) => f.key === 'transport')).toBeDefined()
  })
})

describe('pruneForType', () => {
  // The old handler wiped the model to {type, tag}, discarding dialer options
  // that both types share and that did not need clearing.
  test('keeps fields the new type also has', () => {
    const next = pruneForType(
      { tag: 'x', type: 'shadowsocks', server: '1.1.1.1', method: 'aes-128-gcm', detour: 'd' },
      'trojan',
    )
    expect(next.server).toBe('1.1.1.1')
    expect(next.detour).toBe('d')
    expect(next).not.toHaveProperty('method')
  })

  test('switching to a group drops the dial fields', () => {
    const next = pruneForType({ tag: 'g', type: 'trojan', server: '1.1.1.1' }, 'selector')
    expect(next).toEqual({ tag: 'g', type: 'selector' })
  })
})

describe('version gating', () => {
  test('compareVersions orders release versions', () => {
    expect(compareVersions('1.13.11', '1.13.0')).toBeGreaterThan(0)
    expect(compareVersions('1.12.12', '1.13.0')).toBeLessThan(0)
    expect(compareVersions('1.13.0', '1.13.0')).toBe(0)
    expect(compareVersions('v1.13.0', '1.13.0')).toBe(0)
  })

  // A prerelease of X carries X's removals, so it must gate like X. Both
  // suffix shapes have to agree — they did not before: "1.13.0-beta.1" split
  // into four segments and sorted above "1.13.0" while "1.13.0-rc" sorted equal.
  test('a prerelease of X counts as X, whatever the suffix shape', () => {
    expect(compareVersions('1.13.0-beta.1', '1.13.0')).toBe(0)
    expect(compareVersions('1.13.0-rc', '1.13.0')).toBe(0)
    expect(compareVersions('1.13.0+build.5', '1.13.0')).toBe(0)
    expect(isRetired(OUTBOUND_TYPE_NOTES.dns!, '1.13.0-beta.1')).toBe(true)
  })

  // dns and wireguard outbounds both parse and then fail on 1.13: one at
  // config decode, one at outbound init. `sing-box check` passing proves
  // nothing, which is why this gate exists.
  test('dns and wireguard are retired on 1.13, live on 1.12', () => {
    const dns = OUTBOUND_TYPE_NOTES.dns!
    const wireguard = OUTBOUND_TYPE_NOTES.wireguard!
    expect(dns.removed).toBe('1.13.0')
    expect(wireguard.removed).toBe('1.13.0')

    expect(isRetired(dns, '1.13.11')).toBe(true)
    expect(isRetired(wireguard, '1.13.11')).toBe(true)
    expect(isRetired(dns, '1.12.12')).toBe(false)
    expect(isDeprecatedIn(dns, '1.12.12')).toBe(true)
  })

  test('an undetected version hides nothing', () => {
    // Guessing "probably removed" would hide fields from someone whose binary
    // we merely failed to detect.
    expect(isRetired(OUTBOUND_TYPE_NOTES.dns!, undefined)).toBe(false)
  })
})
