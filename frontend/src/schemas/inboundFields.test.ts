import { describe, expect, test } from 'bun:test'
import {
  applyTypeDefaults,
  isFieldFilled,
  pruneForType,
  resolveInboundFields,
  withoutField,
  USER_FIELDS,
} from './inboundFields'
import { humanizeFieldName } from '../utils/fieldLabels'
import { generateShadowsocksPassword } from '../utils/credentials'

/**
 * These cover the schema's decision rules rather than its data. The per-type
 * curation is editorial and changes freely; the rules below are the ones that
 * silently break the form when they are wrong.
 */

describe('isFieldFilled', () => {
  // The one that matters most: with the usual "not undefined and not empty
  // string" test, a `false` boolean reads as filled, which pins the switch
  // visible AND un-removable, since removal is only offered for empty fields.
  test('treats false as empty so an unticked switch stays removable', () => {
    expect(isFieldFilled(false)).toBe(false)
    expect(isFieldFilled(true)).toBe(true)
  })

  // Opposite rule for numbers: sing-box gives 0 meaning (listen_port 0 = pick
  // one, alterId 0 = modern VMess), so it cannot double as "unset".
  test('treats 0 as a real value', () => {
    expect(isFieldFilled(0)).toBe(true)
  })

  test('empty string, empty array and empty object are unset', () => {
    expect(isFieldFilled('')).toBe(false)
    expect(isFieldFilled([])).toBe(false)
    expect(isFieldFilled({})).toBe(false)
  })

  test('populated values are set', () => {
    expect(isFieldFilled('x')).toBe(true)
    expect(isFieldFilled(['x'])).toBe(true)
    expect(isFieldFilled({ a: 1 })).toBe(true)
  })

  test('undefined and null are unset', () => {
    expect(isFieldFilled(undefined)).toBe(false)
    expect(isFieldFilled(null)).toBe(false)
  })
})

describe('pruneForType', () => {
  // The bug this fixes: the old watcher applied defaults on type change but
  // never removed anything, so shadowsocks' method/password rode along into a
  // trojan payload. sing-box decodes inbound options strictly and rejects it.
  test('drops fields the new type does not have', () => {
    const next = pruneForType(
      {
        tag: 'x',
        type: 'shadowsocks',
        listen: '127.0.0.1',
        listen_port: 1080,
        method: '2022-blake3-aes-128-gcm',
        password: 'secret',
      },
      'trojan',
    )

    expect(next).not.toHaveProperty('method')
    expect(next).not.toHaveProperty('password')
    expect(next).toEqual({ tag: 'x', type: 'trojan', listen: '127.0.0.1', listen_port: 1080 })
  })

  // tun genuinely has no listen fields in its option struct, which is why the
  // old `v-if="type !== 'tun'"` special case existed. Now it falls out of data.
  test('drops listen fields when switching to tun', () => {
    const next = pruneForType({ tag: 'x', type: 'mixed', listen: '127.0.0.1', listen_port: 1080 }, 'tun')
    expect(next).not.toHaveProperty('listen')
    expect(next).not.toHaveProperty('listen_port')
    expect(next.tag).toBe('x')
  })

  test('keeps tag and type, which belong to no option struct', () => {
    const next = pruneForType({ tag: 'keep-me', type: 'mixed' }, 'vmess')
    expect(next.tag).toBe('keep-me')
    expect(next.type).toBe('vmess')
  })

  test('does not mutate its input', () => {
    const original = { tag: 'x', type: 'shadowsocks', method: 'none' }
    pruneForType(original, 'trojan')
    expect(original.method).toBe('none')
  })
})

describe('applyTypeDefaults', () => {
  test('seeds core defaults for a new inbound', () => {
    const next = applyTypeDefaults({ tag: '' }, 'mixed')
    expect(next.listen).toBe('127.0.0.1')
  })

  test('never overwrites a value that is already set', () => {
    const next = applyTypeDefaults({ tag: '', listen: '0.0.0.0' }, 'mixed')
    expect(next.listen).toBe('0.0.0.0')
  })

  test('does not seed advanced fields', () => {
    const next = applyTypeDefaults({ tag: '' }, 'mixed')
    expect(next).not.toHaveProperty('tcp_fast_open')
    expect(next).not.toHaveProperty('sniff')
  })

  test('copies array defaults so callers cannot alias the schema', () => {
    const a = applyTypeDefaults({ tag: 'a' }, 'tun')
    const b = applyTypeDefaults({ tag: 'b' }, 'tun')
    ;(a.address as string[]).push('mutated')
    expect(b.address).toEqual(['172.19.0.1/30'])
  })
})

describe('resolveInboundFields', () => {
  const mixed = resolveInboundFields('mixed')
  const find = (key: string) => mixed.find((f) => f.key === key)

  test('promotes the type-characteristic fields, not the "required" ones', () => {
    // sing-box marks only `listen` Required for mixed; users and
    // set_system_proxy are explicitly optional. Tiering follows the doc
    // example instead, which is what the operator actually came to set.
    expect(find('listen')?.tier).toBe('core')
    expect(find('users')?.tier).toBe('typical')
    expect(find('set_system_proxy')?.tier).toBe('typical')
  })

  test('leaves uncurated fields reachable as advanced rather than dropping them', () => {
    expect(find('tcp_fast_open')?.tier).toBe('advanced')
    expect(find('netns')?.tier).toBe('advanced')
  })

  test('forces deprecated fields to advanced however they are curated', () => {
    expect(find('sniff')?.deprecated).toBe(true)
    expect(find('sniff')?.tier).toBe('advanced')
  })

  test('exposes every field in the inventory', () => {
    // Nothing is filtered here: a field omitted at this layer would be one the
    // operator could never reach, even in a config that already uses it.
    expect(mixed.length).toBeGreaterThan(20)
    expect(find('tls')).toBeDefined()
  })

  test('orders core before typical before advanced', () => {
    const tiers = mixed.map((f) => f.tier)
    expect(tiers.indexOf('core')).toBeLessThan(tiers.indexOf('typical'))
    expect(tiers.indexOf('typical')).toBeLessThan(tiers.indexOf('advanced'))
  })

  test('infers controls from the generated kind', () => {
    expect(find('set_system_proxy')?.control).toBe('switch')
    expect(find('listen_port')?.control).toBe('number')
    expect(find('users')?.control).toBe('users')
    expect(find('tls')?.control).toBe('json')
  })

  test('tun carries no listen fields at all', () => {
    const tun = resolveInboundFields('tun')
    expect(tun.find((f) => f.key === 'listen')).toBeUndefined()
    expect(tun.find((f) => f.key === 'listen_port')).toBeUndefined()
  })
})

describe('USER_FIELDS', () => {
  // These shapes are load-bearing: mixed/http/socks/naive carry sing's
  // auth.User (username+password) while the rest use purpose-built structs.
  // Writing the wrong key produces an inbound that authenticates nobody.
  test('mixed uses username, vmess uses uuid', () => {
    expect(USER_FIELDS.mixed?.map((f) => f.key)).toEqual(['username', 'password'])
    expect(USER_FIELDS.vmess?.map((f) => f.key)).toEqual(['name', 'uuid', 'alterId'])
  })

  test('every user shape names exactly one identity field', () => {
    for (const [type, fields] of Object.entries(USER_FIELDS)) {
      const identities = fields?.filter((f) => f.identity) ?? []
      expect(`${type}:${identities.length}`).toBe(`${type}:1`)
    }
  })
})

describe('withoutField', () => {
  test('deletes the key rather than blanking it', () => {
    const next = withoutField({ a: 1, b: 2 }, 'a')
    expect('a' in next).toBe(false)
    expect(next).toEqual({ b: 2 })
  })

  test('does not mutate its input', () => {
    const original = { a: 1 }
    withoutField(original, 'a')
    expect(original.a).toBe(1)
  })
})

describe('humanizeFieldName', () => {
  test('uppercases initialisms rather than title-casing them', () => {
    expect(humanizeFieldName('tcp_fast_open')).toBe('TCP Fast Open')
    expect(humanizeFieldName('udp_timeout')).toBe('UDP Timeout')
    expect(humanizeFieldName('inet4_address')).toBe('IPv4 Address')
  })

  test('splits camelCase, which sing-box uses for alterId', () => {
    expect(humanizeFieldName('alterId')).toBe('Alter ID')
  })

  test('handles ordinary words', () => {
    expect(humanizeFieldName('strict_route')).toBe('Strict Route')
  })
})

describe('generateShadowsocksPassword', () => {
  // Wrong key length is a sing-box startup failure, not a weak password.
  test('2022-blake3-128 yields a 16-byte base64 key', () => {
    const password = generateShadowsocksPassword('2022-blake3-aes-128-gcm')
    expect(atob(password).length).toBe(16)
  })

  test('other 2022-blake3 ciphers yield 32 bytes', () => {
    expect(atob(generateShadowsocksPassword('2022-blake3-aes-256-gcm')).length).toBe(32)
    expect(atob(generateShadowsocksPassword('2022-blake3-chacha20-poly1305')).length).toBe(32)
  })

  test('none takes no password', () => {
    expect(generateShadowsocksPassword('none')).toBe('')
  })

  test('legacy ciphers get an opaque string', () => {
    expect(generateShadowsocksPassword('aes-128-gcm').length).toBe(32)
  })
})
