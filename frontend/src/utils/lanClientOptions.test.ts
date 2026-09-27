import { describe, expect, test } from 'bun:test'
import { buildLanClientOptions, normalizeMacInput } from './lanClientOptions'
import type { LanClient } from '../types/api'

const clients: LanClient[] = [
  { mac: '00:19:0f:34:84:73', ip: '192.168.31.240', hostname: 'nas', online: true, sources: ['static', 'lease'] },
  { mac: 'aa:21:e7:a7:28:e4', ip: '192.168.31.209', online: false, sources: ['neighbor'] },
  // Two devices announcing the same hostname — one option, not two.
  { mac: '11:22:33:44:55:66', ip: '192.168.31.10', hostname: 'nas', online: false, sources: ['lease'] },
]

describe('buildLanClientOptions', () => {
  test('mac mode offers every client by MAC, labelled with name and IP', () => {
    const options = buildLanClientOptions(clients, 'mac', [])
    expect(options.map((o) => o.value)).toEqual([
      '00:19:0f:34:84:73',
      'aa:21:e7:a7:28:e4',
      '11:22:33:44:55:66',
    ])
    expect(options[0]!.label).toContain('nas')
    expect(options[0]!.label).toContain('192.168.31.240')
    expect(options[0]!.online).toBe(true)
  })

  test('hostname mode offers each announced name once, skipping unnamed clients', () => {
    const options = buildLanClientOptions(clients, 'hostname', [])
    expect(options.map((o) => o.value)).toEqual(['nas'])
    // Online if ANY device with that name is online.
    expect(options[0]!.online).toBe(true)
  })

  test('a value the rule holds but the LAN does not show is kept, flagged', () => {
    const options = buildLanClientOptions(clients, 'mac', ['00:00:5e:00:53:01'])
    const kept = options.find((o) => o.value === '00:00:5e:00:53:01')
    expect(kept?.missing).toBe(true)
  })

  test('a held MAC in another case matches the discovered one instead of duplicating', () => {
    const options = buildLanClientOptions(clients, 'mac', ['00:19:0F:34:84:73'])
    expect(options.filter((o) => o.value.toLowerCase() === '00:19:0f:34:84:73')).toHaveLength(1)
  })
})

describe('normalizeMacInput', () => {
  test.each([
    ['00:19:0F:34:84:73', '00:19:0f:34:84:73'],
    ['00-19-0f-34-84-73', '00:19:0f:34:84:73'],
    [' 00190f348473 ', '00:19:0f:34:84:73'],
  ])('%s -> %s', (input, want) => {
    expect(normalizeMacInput(input)).toBe(want)
  })

  test.each(['', 'nas', '00:19:0f:34:84', '00:19:0f:34:84:zz'])('rejects %p', (input) => {
    expect(normalizeMacInput(input)).toBeUndefined()
  })
})
