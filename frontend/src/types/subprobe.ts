/**
 * Subscription quality metrics — availability and latency, sampled over time.
 *
 * Mirrors app/pkg/subprobe. The backend keeps ONE aggregate row per
 * subscription per run, so everything here is already folded: there is no
 * per-node history to page through, only the latest run's detail.
 */

/** One plotted observation. Already bucketed by the server for wide ranges. */
export interface ProbePoint {
  /** RFC3339 timestamp of the sample (or of the bucket's start). */
  at: string
  /** Nodes that were tested. Untestable ones are excluded — see `skipped`. */
  total: number
  /** Nodes that answered. availability = reachable / total. */
  reachable: number
  /**
   * Mean latency over the REACHABLE nodes only. A node that timed out has no
   * latency, and folding its timeout in would make a partly-dead subscription
   * read as a slow one — two different problems with two different fixes.
   */
  avg_ms: number
  min_ms: number
  max_ms: number
  skipped?: number
}

/** One node's outcome in the most recent run. */
export interface ProbeNodeResult {
  tag: string
  delay_ms?: number
  /** Empty when the node answered. */
  error?: string
  /**
   * True when the node could not be tested at all — it is in the config but
   * not in the running sing-box. NOT the same as being down, and excluded from
   * the availability denominator for that reason.
   */
  skipped?: boolean
}

/** The scheduler's own state, so the UI can explain an empty chart. */
export interface ProbeStatus {
  running: boolean
  in_flight: boolean
  last_run?: string
  last_error?: string
  /** Go duration string, e.g. "10m0s". */
  interval: string
}

/** Payload of GET /subscription-probes/status. */
export interface ProbeStatusResponse {
  status: ProbeStatus
  /** Newest sample per subscription id. Absent id = never probed. */
  latest: Record<string, ProbePoint>
  /** Rows currently stored, across all subscriptions — the disk figure. */
  sample_count: number
  retention: number
  max_points: number
  timeout_ms: number
  interval_secs: number
}

/** The ranges the history endpoint accepts. The server picks the bucket. */
export type ProbeRange = '1h' | '6h' | '24h' | '7d' | '30d'

/** Payload of GET /subscription-probes/:id/history. */
export interface ProbeHistoryResponse {
  id: string
  range: ProbeRange
  /** 0 when the range is served unbucketed. */
  bucket_seconds: number
  points: ProbePoint[]
}

/** Payload of GET /subscription-probes/:id/nodes. */
export interface ProbeNodesResponse {
  /**
   * False until the first sweep since the panel started: the per-node detail
   * is in memory only. Reported rather than served as an empty list, which
   * would read as "every node is fine".
   */
  available: boolean
  at?: string
  sample?: Omit<ProbePoint, 'at'>
  results?: ProbeNodeResult[]
}

/** Payload of PUT /subscription-probes/settings. */
export interface ProbeSettings {
  interval_secs: number
  timeout_ms: number
  retention: number
  max_points: number
  sample_count: number
}
