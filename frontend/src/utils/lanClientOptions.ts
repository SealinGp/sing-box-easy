/**
 * Picker options for a route rule's `source_mac_address` / `source_hostname`,
 * built from the LAN clients the router reports (GET /system/lan-clients).
 *
 * Pure so it can be tested without the component: the picker is markup only.
 */
import type { LanClient } from '../types/api'

export type LanClientPickMode = 'mac' | 'hostname'

export interface LanClientOption {
  value: string
  label: string
  online: boolean
  /** Held by the rule but not seen on the LAN right now. */
  missing?: boolean
}

function macLabel(client: LanClient): string {
  return [client.hostname, client.ip, client.mac].filter(Boolean).join(' · ')
}

/**
 * One option per MAC (mac mode) or per announced hostname (hostname mode).
 *
 * Values the rule already holds are always present: a MultiSelect renders
 * nothing for a value with no option, so a device that is offline or was
 * renamed would vanish from the form while still living in the rule — and the
 * next save would look like the operator removed it.
 */
export function buildLanClientOptions(
  clients: readonly LanClient[],
  mode: LanClientPickMode,
  held: readonly string[],
): LanClientOption[] {
  const options: LanClientOption[] = []
  const index = new Map<string, LanClientOption>()

  for (const client of clients) {
    if (mode === 'mac') {
      const option = { value: client.mac, label: macLabel(client), online: client.online }
      options.push(option)
      index.set(client.mac, option)
      continue
    }
    if (!client.hostname) continue
    const existing = index.get(client.hostname)
    if (existing) {
      existing.online ||= client.online
      continue
    }
    const option = {
      value: client.hostname,
      label: [client.hostname, client.ip].filter(Boolean).join(' · '),
      online: client.online,
    }
    options.push(option)
    index.set(client.hostname, option)
  }

  for (const value of held) {
    const key = mode === 'mac' ? value.toLowerCase() : value
    if (index.has(key) || index.has(value)) continue
    const option = { value, label: value, online: false, missing: true }
    options.push(option)
    index.set(key, option)
  }
  return options
}

/**
 * A MAC typed by hand, in the canonical lower-case colon form, or undefined
 * when it is not one. Accepts `-` separators and a bare 12-digit form, which
 * is how MACs are printed on device labels.
 */
export function normalizeMacInput(input: string): string | undefined {
  const hex = input.trim().toLowerCase().replace(/[:-]/g, '')
  if (!/^[0-9a-f]{12}$/.test(hex)) return undefined
  return hex.match(/../g)!.join(':')
}
