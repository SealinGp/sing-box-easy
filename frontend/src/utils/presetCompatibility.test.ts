import { describe, expect, it } from 'bun:test'
import { compatibleInbound } from './presetCompatibility'

const legacy = { type: 'mixed', tag: 'mixed-in', listen: '127.0.0.1', listen_port: 7890, sniff: true, sniff_override_destination: true, domain_strategy: 'prefer_ipv4' } as const
describe('inbound preset compatibility', () => {
  it('retains options supported by a 1.12 core', () => {
    expect(compatibleInbound(legacy, '1.12.12')).toEqual(legacy)
  })
  it('sends only supported listener options to a 1.14 core', () => {
    expect(compatibleInbound(legacy, '1.14.0')).toEqual({ type: 'mixed', tag: 'mixed-in', listen: '127.0.0.1', listen_port: 7890 })
  })
  it('does not guess a version or mutate the original preset', () => {
    expect(compatibleInbound(legacy, undefined)).toEqual(legacy)
    compatibleInbound(legacy, '1.14.0')
    expect(legacy.sniff).toBe(true)
  })
})
