import { describe, expect, test } from 'bun:test'
import { parseOutboundJson } from './outboundJson'
describe('native node paste', () => {
  test('legacy VMess copy drops display username and preserves native options', () => {
    const raw = { type: 'vmess', tag: 'test-out', server: '192.0.2.10', server_port: 443, uuid: '00000000-0000-4000-8000-000000000001', username: 'display', tls: { enabled: true, server_name: 'example.com' }, transport: { type: 'ws', path: '/proxy' } }
    const parsed = parseOutboundJson(JSON.stringify(raw))
    expect(parsed.username).toBeUndefined()
    expect(parsed.tls).toEqual(raw.tls)
    expect(parsed.transport).toEqual(raw.transport)
  })
  test.each(['socks', 'http'])('%s retains actual authentication username', type => {
    expect(parseOutboundJson(JSON.stringify({ type, username: 'login', password: 'test' })).username).toBe('login')
  })
  test.each(['null', '[]', '{', '{"type":"unsupported"}', '{"type":"vmess","listen_port":443}'])('rejects invalid or server-side input %s', raw => {
    expect(() => parseOutboundJson(raw)).toThrow()
  })
})
