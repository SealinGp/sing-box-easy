/**
 * Pure logic behind the Connections page: which columns exist, how rows are
 * searched and sorted, and how "closed" connections are remembered.
 *
 * WHY CLOSED CONNECTIONS ARE TRACKED HERE
 * ───────────────────────────────────────
 * sing-box reports ACTIVE connections only — it keeps the last 1000 closed
 * ones internally but exposes them on no endpoint. So "closed" is derived: a
 * connection that was in the previous frame and is missing from this one has
 * ended. That history lives in the browser, is bounded, and starts empty on
 * every page load; the page says so rather than implying a complete record.
 */
import type { ConnectionRow } from '../types/runtime'

export type ColumnKey =
  | 'source' | 'sourcePort' | 'host' | 'destinationIp' | 'network' | 'inbound'
  | 'rule' | 'chain' | 'downRate' | 'upRate' | 'download' | 'upload' | 'duration' | 'start'

/** Every column, in the order the settings dialog offers them. */
export const ALL_COLUMNS: readonly ColumnKey[] = [
  'source', 'sourcePort', 'host', 'destinationIp', 'network', 'inbound',
  'rule', 'chain', 'downRate', 'upRate', 'download', 'upload', 'duration', 'start',
]

export const DEFAULT_COLUMNS: readonly ColumnKey[] = [
  'source', 'host', 'network', 'rule', 'chain', 'downRate', 'upRate', 'download', 'upload', 'duration',
]

/** Numeric columns are right-aligned and sort descending first. */
export const NUMERIC_COLUMNS: ReadonlySet<ColumnKey> = new Set([
  'sourcePort', 'downRate', 'upRate', 'download', 'upload', 'duration', 'start',
])

/** Bounded so a busy router's history cannot grow without limit in a tab left open. */
export const CLOSED_LIMIT = 500

export interface TrackedConnection extends ConnectionRow {
  closed: boolean
  /** Unix ms the connection was first seen missing; 0 while active. */
  closedAt: number
}

export type ConnectionTab = 'active' | 'closed' | 'all'

export interface TableSettings {
  columns: ColumnKey[]
  /** Show the whole exit chain rather than "exit via leaf". */
  fullChain: boolean
  dense: boolean
  /** Source IP → device name. */
  labels: Record<string, string>
}

export function defaultSettings(): TableSettings {
  return { columns: [...DEFAULT_COLUMNS], fullChain: false, dense: false, labels: {} }
}

/**
 * Rebuilds settings from whatever localStorage held. Every field is checked:
 * the value was written by an older build, or by hand, and a column key this
 * build does not know would render a header with no cells.
 */
export function normalizeSettings(raw: unknown): TableSettings {
  const fallback = defaultSettings()
  if (typeof raw !== 'object' || raw === null) return fallback
  const input = raw as Record<string, unknown>

  const known = new Set<string>(ALL_COLUMNS)
  const seen = new Set<string>()
  const columns: ColumnKey[] = []
  if (Array.isArray(input.columns)) {
    for (const key of input.columns) {
      if (typeof key === 'string' && known.has(key) && !seen.has(key)) {
        seen.add(key)
        columns.push(key as ColumnKey)
      }
    }
  }

  const labels: Record<string, string> = {}
  if (typeof input.labels === 'object' && input.labels !== null) {
    for (const [ip, label] of Object.entries(input.labels as Record<string, unknown>)) {
      if (typeof label === 'string' && label.trim() !== '') labels[ip] = label.trim()
    }
  }

  return {
    columns: columns.length > 0 ? columns : fallback.columns,
    fullChain: input.fullChain === true,
    dense: input.dense === true,
    labels,
  }
}

/** Moves `key` one step left (-1) or right (+1). Returns a new array. */
export function moveColumn(columns: readonly ColumnKey[], key: ColumnKey, step: -1 | 1): ColumnKey[] {
  const from = columns.indexOf(key)
  const to = from + step
  if (from < 0 || to < 0 || to >= columns.length) return [...columns]
  const neighbour = columns[to]
  if (neighbour === undefined) return [...columns]
  const next = [...columns]
  next[from] = neighbour
  next[to] = key
  return next
}

/** Moves `key` to where `target` currently sits (drag and drop). Returns a new array. */
export function placeColumn(columns: readonly ColumnKey[], key: ColumnKey, target: ColumnKey): ColumnKey[] {
  if (key === target || !columns.includes(key) || !columns.includes(target)) return [...columns]
  const targetIndex = columns.indexOf(target)
  const without = columns.filter((column) => column !== key)
  without.splice(targetIndex, 0, key)
  return without
}

/**
 * Folds one frame into the tracked state.
 *
 * `previousActive` rows absent from `frame` become closed, newest first, and
 * keep their final byte counts with their rates zeroed — a closed connection
 * moving at 2 MB/s forever would be the most eye-catching row in the table.
 * Neither input is mutated.
 */
export function trackFrame(
  previousActive: readonly ConnectionRow[],
  previousClosed: readonly TrackedConnection[],
  frame: readonly ConnectionRow[],
  now: number,
): { active: TrackedConnection[]; closed: TrackedConnection[] } {
  const live = new Set(frame.map((row) => row.id))
  const justClosed: TrackedConnection[] = previousActive
    .filter((row) => !live.has(row.id))
    .map((row) => ({ ...row, up_rate: 0, down_rate: 0, fresh: false, closed: true, closedAt: now }))

  return {
    active: frame.map((row) => ({ ...row, closed: false, closedAt: 0 })),
    closed: justClosed.length > 0 ? [...justClosed, ...previousClosed].slice(0, CLOSED_LIMIT) : [...previousClosed],
  }
}

/** The outbound the rule named. Chains arrive exit first. */
export function exitOf(row: ConnectionRow): string {
  return row.chains[0] ?? ''
}

/** The node actually dialled, when it differs from the exit. */
export function leafOf(row: ConnectionRow): string {
  return row.chains.length > 1 ? row.chains[row.chains.length - 1] ?? '' : ''
}

/** What the host column shows: the sniffed/requested name, else the address. */
export function hostOf(row: ConnectionRow): string {
  return row.host || row.destination_ip
}

/** Seconds a connection has been (or was) open. */
export function durationOf(row: TrackedConnection, now: number): number {
  if (!row.start) return 0
  const end = row.closed ? row.closedAt : now
  return Math.max(0, Math.floor((end - row.start) / 1000))
}

export function sortValue(row: TrackedConnection, key: ColumnKey, labels: Record<string, string>, now: number): number | string {
  switch (key) {
    case 'source': return labels[row.source_ip] || row.source_ip
    case 'sourcePort': return Number(row.source_port) || 0
    case 'host': return hostOf(row)
    case 'destinationIp': return row.destination_ip
    case 'network': return row.network
    case 'inbound': return row.inbound
    case 'rule': return row.rule
    case 'chain': return exitOf(row)
    case 'downRate': return row.down_rate
    case 'upRate': return row.up_rate
    case 'download': return row.download
    case 'upload': return row.upload
    case 'duration': return durationOf(row, now)
    case 'start': return row.start
  }
}

/**
 * Compiles the search box. An invalid regular expression is not an error — the
 * operator is usually halfway through typing one — so it falls back to a
 * literal, case-insensitive match of what was typed.
 */
export function compileSearch(query: string): ((text: string) => boolean) | null {
  const trimmed = query.trim()
  if (trimmed === '') return null
  try {
    const pattern = new RegExp(trimmed, 'i')
    return (text) => pattern.test(text)
  } catch {
    const needle = trimmed.toLowerCase()
    return (text) => text.toLowerCase().includes(needle)
  }
}

export interface ViewOptions {
  tab: ConnectionTab
  query: string
  /** Exact source IP, or '' for every device. */
  source: string
  sortKey: ColumnKey
  ascending: boolean
  labels: Record<string, string>
  now: number
}

/** The rows a tab draws from, before search and sort. */
export function poolFor(tab: ConnectionTab, active: readonly TrackedConnection[], closed: readonly TrackedConnection[]): TrackedConnection[] {
  if (tab === 'active') return [...active]
  if (tab === 'closed') return [...closed]
  return [...active, ...closed]
}

export function viewRows(
  active: readonly TrackedConnection[],
  closed: readonly TrackedConnection[],
  options: ViewOptions,
): TrackedConnection[] {
  const matches = compileSearch(options.query)
  const rows = poolFor(options.tab, active, closed).filter((row) => {
    if (options.source && row.source_ip !== options.source) return false
    if (!matches) return true
    return matches([
      row.source_ip, options.labels[row.source_ip] ?? '', row.host, row.destination_ip,
      row.rule, row.chains.join(' '), row.network, row.inbound,
    ].join(' '))
  })

  const direction = options.ascending ? 1 : -1
  return rows.sort((a, b) => {
    const x = sortValue(a, options.sortKey, options.labels, options.now)
    const y = sortValue(b, options.sortKey, options.labels, options.now)
    const order = typeof x === 'number' && typeof y === 'number' ? x - y : String(x).localeCompare(String(y))
    // The id breaks ties so equal rows do not swap places every second.
    return order !== 0 ? order * direction : a.id.localeCompare(b.id)
  })
}

export interface SourceOption {
  ip: string
  label: string
  count: number
}

/** Devices holding connections in `rows`, ordered by address. */
export function sourceOptions(rows: readonly ConnectionRow[], labels: Record<string, string>): SourceOption[] {
  const counts = new Map<string, number>()
  for (const row of rows) counts.set(row.source_ip, (counts.get(row.source_ip) ?? 0) + 1)
  return [...counts.entries()]
    .sort(([a], [b]) => a.localeCompare(b, undefined, { numeric: true }))
    .map(([ip, count]) => ({ ip, label: labels[ip] || ip, count }))
}

/** "42s", "3m 05s", "2h 14m". */
export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return '—'
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${String(seconds % 60).padStart(2, '0')}s`
  return `${Math.floor(seconds / 3600)}h ${String(Math.floor((seconds % 3600) / 60)).padStart(2, '0')}m`
}
