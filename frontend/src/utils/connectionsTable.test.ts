import { describe, expect, test } from 'bun:test'
import {
  CLOSED_LIMIT, DEFAULT_COLUMNS, compileSearch, durationOf, exitOf, formatDuration, leafOf, moveColumn,
  normalizeSettings, placeColumn, sourceOptions, trackFrame, viewRows, type TrackedConnection, type ViewOptions,
} from './connectionsTable'
import type { ConnectionRow } from '../types/runtime'

const row = (id: string, over: Partial<ConnectionRow> = {}): ConnectionRow => ({
  id, network: 'tcp', inbound: 'tun/tun-in', source_ip: '192.168.1.2', source_port: '5000',
  destination_ip: '1.2.3.4', destination_port: '443', host: `${id}.example.com`, process_path: '',
  rule: 'final', chains: ['direct'], start: 1_000, upload: 0, download: 0, up_rate: 0, down_rate: 0, fresh: false,
  ...over,
})
const tracked = (id: string, over: Partial<TrackedConnection> = {}): TrackedConnection =>
  ({ ...row(id), closed: false, closedAt: 0, ...over })
const options = (over: Partial<ViewOptions> = {}): ViewOptions =>
  ({ tab: 'active', query: '', source: '', sortKey: 'downRate', ascending: false, labels: {}, now: 10_000, ...over })

describe('trackFrame', () => {
  test('a connection missing from the new frame becomes closed with its rates zeroed', () => {
    const previous = [row('a', { down_rate: 2_000_000, download: 99 }), row('b')]
    const { active, closed } = trackFrame(previous, [], [row('b')], 5_000)
    expect(active.map((c) => c.id)).toEqual(['b'])
    expect(closed).toHaveLength(1)
    expect(closed[0]!).toMatchObject({ id: 'a', closed: true, closedAt: 5_000, down_rate: 0, download: 99 })
    expect(previous[0]!.down_rate).toBe(2_000_000)
  })
  test('newly closed rows go first and history is capped', () => {
    const history = Array.from({ length: CLOSED_LIMIT }, (_, i) => tracked(`old${i}`, { closed: true, closedAt: 1 }))
    const { closed } = trackFrame([row('new')], history, [], 9)
    expect(closed).toHaveLength(CLOSED_LIMIT)
    expect(closed[0]!.id).toBe('new')
    expect(closed.at(-1)?.id).toBe(`old${CLOSED_LIMIT - 2}`)
  })
  test('an unchanged frame closes nothing', () => {
    expect(trackFrame([row('a')], [], [row('a')], 1).closed).toHaveLength(0)
  })
})

describe('chain helpers', () => {
  test('chains are exit first', () => {
    const r = row('a', { chains: ['Media', '流媒体', '新加坡 01'] })
    expect(exitOf(r)).toBe('Media')
    expect(leafOf(r)).toBe('新加坡 01')
  })
  test('a single-hop chain has no separate leaf', () => {
    expect(leafOf(row('a'))).toBe('')
    expect(exitOf(row('a', { chains: [] }))).toBe('')
  })
})

describe('durationOf', () => {
  test('an active connection runs to now, a closed one stops at closedAt', () => {
    expect(durationOf(tracked('a', { start: 1_000 }), 61_000)).toBe(60)
    expect(durationOf(tracked('a', { start: 1_000, closed: true, closedAt: 31_000 }), 999_000)).toBe(30)
  })
  test('an unknown start is zero, not fifty years', () => {
    expect(durationOf(tracked('a', { start: 0 }), 61_000)).toBe(0)
  })
})

describe('compileSearch', () => {
  test('empty is no filter', () => {
    expect(compileSearch('   ')).toBeNull()
  })
  test('regex, case-insensitive', () => {
    const match = compileSearch('google|GITHUB')!
    expect(match('api.github.com')).toBe(true)
    expect(match('example.com')).toBe(false)
  })
  test('an unfinished regex falls back to a literal match instead of throwing', () => {
    const match = compileSearch('rule_set=[geo')!
    expect(match('rule_set=[geosite-cn] => route(direct)')).toBe(true)
    expect(match('final')).toBe(false)
  })
})

describe('viewRows', () => {
  const active = [
    tracked('a', { down_rate: 10, source_ip: '192.168.1.2' }),
    tracked('b', { down_rate: 500, source_ip: '192.168.1.9', host: 'api.github.com' }),
    tracked('c', { down_rate: 10, source_ip: '192.168.1.2' }),
  ]
  const closed = [tracked('z', { closed: true, closedAt: 9_000 })]

  test('tabs choose the pool', () => {
    expect(viewRows(active, closed, options()).map((r) => r.id)).toEqual(['b', 'a', 'c'])
    expect(viewRows(active, closed, options({ tab: 'closed' })).map((r) => r.id)).toEqual(['z'])
    expect(viewRows(active, closed, options({ tab: 'all' }))).toHaveLength(4)
  })
  test('equal rows keep a stable order in both directions', () => {
    expect(viewRows(active, [], options({ ascending: true })).map((r) => r.id)).toEqual(['a', 'c', 'b'])
  })
  test('device filter is exact, search also matches the device label', () => {
    expect(viewRows(active, [], options({ source: '192.168.1.9' })).map((r) => r.id)).toEqual(['b'])
    const labelled = viewRows(active, [], options({ query: 'living room', labels: { '192.168.1.9': 'Living Room TV' } }))
    expect(labelled.map((r) => r.id)).toEqual(['b'])
  })
  test('sorting by source uses the label when there is one', () => {
    const rows = viewRows(active, [], options({ sortKey: 'source', ascending: true, labels: { '192.168.1.2': 'zzz' } }))
    expect(rows.map((r) => r.id)).toEqual(['b', 'a', 'c'])
  })
  test('inputs are not reordered', () => {
    viewRows(active, closed, options({ sortKey: 'host', ascending: true }))
    expect(active.map((r) => r.id)).toEqual(['a', 'b', 'c'])
  })
})

describe('sourceOptions', () => {
  test('counts per device, numeric address order, label when set', () => {
    const rows = [row('a', { source_ip: '192.168.1.10' }), row('b', { source_ip: '192.168.1.9' }), row('c', { source_ip: '192.168.1.10' })]
    expect(sourceOptions(rows, { '192.168.1.9': 'Phone' })).toEqual([
      { ip: '192.168.1.9', label: 'Phone', count: 1 },
      { ip: '192.168.1.10', label: '192.168.1.10', count: 2 },
    ])
  })
})

describe('column settings', () => {
  test('normalizeSettings drops unknown and duplicate columns and blank labels', () => {
    const settings = normalizeSettings({
      columns: ['host', 'bogus', 'host', 'rule'], fullChain: 'yes', dense: true,
      labels: { '10.0.0.1': '  TV ', '10.0.0.2': '   ', '10.0.0.3': 7 },
    })
    expect(settings).toEqual({ columns: ['host', 'rule'], fullChain: false, dense: true, labels: { '10.0.0.1': 'TV' } })
  })
  test('garbage and an empty column list fall back to the defaults', () => {
    expect(normalizeSettings(null).columns).toEqual([...DEFAULT_COLUMNS])
    expect(normalizeSettings('x').columns).toEqual([...DEFAULT_COLUMNS])
    expect(normalizeSettings({ columns: ['bogus'] }).columns).toEqual([...DEFAULT_COLUMNS])
  })
  test('moveColumn swaps neighbours and ignores moves off either end', () => {
    expect(moveColumn(['source', 'host', 'rule'], 'host', -1)).toEqual(['host', 'source', 'rule'])
    expect(moveColumn(['source', 'host', 'rule'], 'host', 1)).toEqual(['source', 'rule', 'host'])
    expect(moveColumn(['source', 'host'], 'source', -1)).toEqual(['source', 'host'])
  })
  test('placeColumn drops a column where the target sits', () => {
    expect(placeColumn(['source', 'host', 'rule', 'chain'], 'chain', 'host')).toEqual(['source', 'chain', 'host', 'rule'])
    expect(placeColumn(['source', 'host', 'rule', 'chain'], 'source', 'rule')).toEqual(['host', 'rule', 'source', 'chain'])
    expect(placeColumn(['source', 'host'], 'source', 'source')).toEqual(['source', 'host'])
  })
})

describe('formatDuration', () => {
  test('units', () => {
    expect(formatDuration(0)).toBe('0s')
    expect(formatDuration(59)).toBe('59s')
    expect(formatDuration(185)).toBe('3m 05s')
    expect(formatDuration(8_040)).toBe('2h 14m')
    expect(formatDuration(-1)).toBe('—')
  })
})
