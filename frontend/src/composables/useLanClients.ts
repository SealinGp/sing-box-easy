import { readonly, ref } from 'vue'
import { systemService } from '../services'
import type { LanClient } from '../types/api'

/**
 * LAN clients for the route rule pickers, shared by every picker on the page.
 *
 * Cached briefly rather than for the session: leases and liveness change by
 * the minute, but one rule dialog renders two pickers (MAC and hostname) and
 * both should cost one request.
 */
const FRESH_MS = 15_000

const clients = ref<LanClient[]>([])
/** undefined until the first answer; false off OpenWrt or on failure. */
const supported = ref<boolean | undefined>(undefined)
const loading = ref(false)
let fetchedAt = 0
let inflight: Promise<void> | null = null

function refresh(force = false): Promise<void> {
  if (inflight) return inflight
  if (!force && supported.value !== undefined && Date.now() - fetchedAt < FRESH_MS) {
    return Promise.resolve()
  }
  loading.value = true
  inflight = systemService
    .getLanClients()
    .then((result) => {
      clients.value = result.clients ?? []
      supported.value = result.supported
      for (const warning of result.warnings ?? []) {
        console.warn('[lan-clients]', warning)
      }
    })
    .catch((err) => {
      // Non-fatal: the picker degrades to typing MACs / hostnames by hand.
      console.error('Failed to fetch LAN clients:', err)
      supported.value = false
    })
    .finally(() => {
      fetchedAt = Date.now()
      loading.value = false
      inflight = null
    })
  return inflight
}

export function useLanClients() {
  void refresh()
  return {
    clients: readonly(clients),
    supported: readonly(supported),
    loading: readonly(loading),
    refresh,
  }
}
