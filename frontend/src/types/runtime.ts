/**
 * Wire types for the /runtime endpoints: what the RUNNING sing-box is doing,
 * as opposed to what the config document says.
 */

export interface RuntimeMember {
  name: string
  /** sing-box's display name: "VLESS", "URLTest", … Empty when the running config has no such outbound. */
  type: string
  /** Last URL-test latency in ms; 0 means untested or failed. */
  delay: number
  /** True when the member is itself a group. */
  group: boolean
}

export interface RuntimeGroup {
  name: string
  type: string
  /** The member currently in use. */
  now: string
  /** Latency of the node `now` ultimately resolves to. */
  delay: number
  /** Only selectors can be switched by hand. */
  switchable: boolean
  members: RuntimeMember[]
}

export interface RuntimeProxies {
  groups: RuntimeGroup[]
  /** False when `experimental.cache_file` is off: a selection is lost on restart. */
  selection_persisted: boolean
}

export interface NodeDelay {
  name: string
  delay: number
}

export interface GroupDelays {
  group: string
  /** Members that did not answer are absent. */
  delays: Record<string, number>
}

export interface ConnectionRow {
  id: string
  network: string
  /** `<inboundType>/<inboundTag>`, e.g. "tun/tun-in". */
  inbound: string
  source_ip: string
  source_port: string
  destination_ip: string
  destination_port: string
  host: string
  process_path: string
  rule: string
  /** EXIT FIRST: the outbound the rule named, ending at the node actually dialled. */
  chains: string[]
  /** Unix ms; 0 when unknown. */
  start: number
  upload: number
  download: number
  up_rate: number
  down_rate: number
  /** First sighting: rates are 0 because there is no baseline, not because it is idle. */
  fresh: boolean
}

export interface ConnectionTotals {
  connections: number
  down_rate: number
  up_rate: number
  download_total: number
  upload_total: number
  memory: number
}

export interface ConnectionsFrame {
  at: number
  totals: ConnectionTotals
  connections: ConnectionRow[]
}
