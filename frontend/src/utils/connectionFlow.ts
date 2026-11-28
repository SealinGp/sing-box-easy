import type { TrackedConnection } from './connectionsTable'

export type ConnectionView = 'groups' | 'compact'
export function connectionView(value: unknown): ConnectionView {
  return value === 'compact' ? 'compact' : 'groups'
}
export function endpoint(address: string, port: string): string {
  const host = address.includes(':') && !address.startsWith('[') ? `[${address}]` : address
  return `${host || '–'}${port ? `:${port}` : ''}`
}
/** Preserve input order, including the held order while details are inspected. */
export function groupConnections(rows: readonly TrackedConnection[]) {
  const groups = new Map<string, { ip: string; rows: TrackedConnection[]; down: number; up: number }>()
  for (const row of rows) {
    let group = groups.get(row.source_ip)
    if (!group) {
      group = { ip: row.source_ip, rows: [], down: 0, up: 0 }
      groups.set(row.source_ip, group)
    }
    group.rows.push(row)
    group.down += row.down_rate
    group.up += row.up_rate
  }
  return [...groups.values()]
}
