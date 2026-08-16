import { describe, expect, test } from 'bun:test'
import {
  DNS_SERVER_TYPE_NAMES,
  DNS_TYPES_NEEDING_SERVER,
  applyTypeDefaults,
  pruneForType,
  resolveDNSServerFields,
} from './dnsServerFields'

/**
 * Like the inbound suite, these pin the decision rules rather than the curation
 * data. The type-name assertions are the exception: `h3` vs `http3` is the exact
 * regression that made a valid config unopenable and a saved one unstartable,
 * so it is worth a literal check on the frontend as well as the backend.
 */

describe('DNS type names', () => {
  test('HTTP/3 is spelled h3, never http3', () => {
    expect(DNS_SERVER_TYPE_NAMES).toContain('h3')
    expect(DNS_SERVER_TYPE_NAMES).not.toContain('http3')
  })

  test('includes tailscale, which the registry used to omit', () => {
    expect(DNS_SERVER_TYPE_NAMES).toContain('tailscale')
  })

  test('excludes the legacy transport, which sing-box upgrades on read', () => {
    expect(DNS_SERVER_TYPE_NAMES).not.toContain('legacy')
    expect(DNS_SERVER_TYPE_NAMES).not.toContain('')
  })
})

describe('DNS_TYPES_NEEDING_SERVER', () => {
  // Derived from the inventory rather than hand-listed — this replaced the
  // `needsServerAddress` array, which had a byte-identical twin named
  // `supportsDetour`.
  test('covers exactly the remote transports', () => {
    // Compared as plain strings: the array is a union type, and asserting
    // against a literal list of that union would just restate the type.
    const actual = DNS_TYPES_NEEDING_SERVER.map(String).sort()
    expect(actual).toEqual(['h3', 'https', 'quic', 'tcp', 'tls', 'udp'])
  })

  test('excludes the types resolved without an upstream', () => {
    for (const type of ['local', 'hosts', 'fakeip', 'dhcp', 'tailscale']) {
      expect(DNS_TYPES_NEEDING_SERVER).not.toContain(type as never)
    }
  })
})

describe('resolveDNSServerFields', () => {
  test('promotes the address and the fields people actually edit', () => {
    const udp = resolveDNSServerFields('udp')
    const find = (key: string) => udp.find((f) => f.key === key)

    expect(find('server')?.tier).toBe('core')
    expect(find('server_port')?.tier).toBe('typical')
    expect(find('detour')?.tier).toBe('typical')
    expect(find('detour')?.control).toBe('outbound')
  })

  test('exposes dialer options that were unreachable before, as advanced', () => {
    const udp = resolveDNSServerFields('udp')
    for (const key of ['bind_interface', 'connect_timeout', 'domain_resolver']) {
      expect(udp.find((f) => f.key === key)?.tier).toBe('advanced')
    }
  })

  test('local has no upstream address at all', () => {
    const local = resolveDNSServerFields('local')
    expect(local.find((f) => f.key === 'server')).toBeUndefined()
    expect(local.find((f) => f.key === 'server_port')).toBeUndefined()
    // It is not empty, though — it owns the dialer options.
    expect(local.length).toBeGreaterThan(5)
  })

  test('hosts gets the dedicated map editor, not a JSON textarea', () => {
    const hosts = resolveDNSServerFields('hosts')
    const predefined = hosts.find((f) => f.key === 'predefined')
    expect(predefined?.control).toBe('hosts')
    expect(predefined?.tier).toBe('core')
  })

  test('https exposes method and headers, which the old form could not edit', () => {
    const https = resolveDNSServerFields('https')
    expect(https.find((f) => f.key === 'method')).toBeDefined()
    expect(https.find((f) => f.key === 'headers')).toBeDefined()
    expect(https.find((f) => f.key === 'path')?.tier).toBe('typical')
  })

  test('deprecated fields are forced to advanced', () => {
    const udp = resolveDNSServerFields('udp')
    const strategy = udp.find((f) => f.key === 'domain_strategy')
    expect(strategy?.deprecated).toBe(true)
    expect(strategy?.tier).toBe('advanced')
  })

  test('orders core before typical before advanced', () => {
    const tiers = resolveDNSServerFields('udp').map((f) => f.tier)
    expect(tiers.indexOf('core')).toBeLessThan(tiers.indexOf('typical'))
    expect(tiers.indexOf('typical')).toBeLessThan(tiers.indexOf('advanced'))
  })
})

describe('pruneForType', () => {
  // There was no type-change handler at all in the old form, so switching type
  // in the add dialog carried the previous type's keys into the payload — and
  // sing-box decodes DNS options strictly.
  test('drops fields the new type does not have', () => {
    const next = pruneForType(
      { tag: 'x', type: 'https', server: '1.1.1.1', server_port: 443, path: '/dns-query' },
      'local',
    )
    expect(next).toEqual({ tag: 'x', type: 'local' })
  })

  test('keeps tag and type, which belong to no option struct', () => {
    const next = pruneForType({ tag: 'keep', type: 'udp' }, 'hosts')
    expect(next.tag).toBe('keep')
    expect(next.type).toBe('hosts')
  })

  test('does not mutate its input', () => {
    const original = { tag: 'x', type: 'https', path: '/dns-query' }
    pruneForType(original, 'local')
    expect(original.path).toBe('/dns-query')
  })
})

describe('applyTypeDefaults', () => {
  test('seeds fakeip ranges, which are useless when empty', () => {
    const next = applyTypeDefaults({ tag: 'f' }, 'fakeip')
    expect(next.inet4_range).toBe('198.18.0.0/15')
    expect(next.inet6_range).toBe('fc00::/18')
  })

  test('never overwrites a value already set', () => {
    const next = applyTypeDefaults({ tag: 'f', inet4_range: '10.0.0.0/8' }, 'fakeip')
    expect(next.inet4_range).toBe('10.0.0.0/8')
  })

  test('does not invent an address for remote types', () => {
    // `server` is core but has no default: a wrong upstream silently resolving
    // would be worse than an empty field the operator must fill.
    expect(applyTypeDefaults({ tag: 'u' }, 'udp').server).toBeUndefined()
  })
})
