import { describe, expect, test } from 'bun:test'
import { latencyStats, parseNetworkInfo } from './browserNetwork'

describe('browser network measurements', () => {
  test('failures do not become zero-latency successes', () => {
    expect(latencyStats([null, 20, 40, 60, null])).toEqual({ min: 20, avg: 40, max: 60 })
    expect(latencyStats([null, null])).toBeNull()
    expect(latencyStats([])).toBeNull()
  })
  test('normalizes both providers and removes duplicate locations', () => {
    expect(parseNetworkInfo('ipip', { ret: 'ok', data: { ip: '203.0.113.1', location: ['Country', 'City', 'City', 'ISP'] } })).toEqual({ ip: '203.0.113.1', location: 'Country · City', organization: 'ISP' })
    expect(parseNetworkInfo('ipsb', { ip: '2001:db8::1', country: 'Country', asn_organization: 'ISP' })).toEqual({ ip: '2001:db8::1', location: 'Country', organization: 'ISP' })
  })
  test('rejects provider errors and malformed IP addresses', () => {
    for (const value of [null, {}, { ip: '999.1.1.1' }, { ip: 'example.com' }, { ip: 'invalid::ip' }]) {
      expect(() => parseNetworkInfo('ipsb', value)).toThrow()
    }
    expect(() => parseNetworkInfo('ipip', { ret: 'error', data: { ip: '203.0.113.1' } })).toThrow()
  })
})
