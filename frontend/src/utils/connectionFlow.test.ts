import { expect, test } from 'bun:test'
import { connectionView, endpoint, groupConnections } from './connectionFlow'
import type { TrackedConnection } from './connectionsTable'
const row = (id: string, source_ip: string, down_rate = 0): TrackedConnection => ({
  id, source_ip, down_rate, up_rate: 12, source_port: '5000', destination_ip: '192.0.2.2', destination_port: '443',
  network: 'tcp', inbound: 'tun', host: '', process_path: '', rule: '', chains: [], start: 100,
  upload: 0, download: 0, fresh: false, closed: false, closedAt: 0,
})
test('groups preserve held device and connection order and sum only displayed rates', () => {
  const rows = [row('a', '192.0.2.1', 20), row('b', '192.0.2.2', 40), row('c', '192.0.2.1', 60)]
  const groups = groupConnections(rows)
  expect(groups.map(g => g.ip)).toEqual(['192.0.2.1', '192.0.2.2'])
  expect(groups[0]!.rows.map(r => r.id)).toEqual(['a', 'c'])
  expect(groups[0]).toMatchObject({ down: 80, up: 24 })
  expect(rows.map(r => r.id)).toEqual(['a', 'b', 'c'])
  expect(groupConnections([])).toEqual([])
})
test('endpoint brackets IPv6 and keeps missing values readable', () => {
  expect(endpoint('2001:db8::1', '443')).toBe('[2001:db8::1]:443')
  expect(endpoint('[::1]', '80')).toBe('[::1]:80')
  expect(endpoint('example.com', '443')).toBe('example.com:443')
  expect(endpoint('', '')).toBe('–')
})
test('unknown saved view falls back to device groups', () => {
  expect(connectionView('compact')).toBe('compact')
  expect(connectionView('old-table')).toBe('groups')
  expect(connectionView(null)).toBe('groups')
})
