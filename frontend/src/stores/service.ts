import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { ServiceStatus } from '../types/api'
import { serviceControlService } from '../services'

/**
 * Shared sing-box service status.
 *
 * The sidebar subscribes via startPolling() so the running/stopped indicator
 * stays live on every page (e.g. right after a restart). Polling is
 * ref-counted: the single interval runs only while at least one component is
 * subscribed, and pauses while the tab is hidden. The Overview page does NOT
 * subscribe — it reads/refreshes on demand so it stays manual-refresh only.
 */
export const useServiceStore = defineStore('service', () => {
  const status = ref<ServiceStatus | null>(null)
  const lastUpdated = ref<number | null>(null)
  const error = ref(false)

  let timer: ReturnType<typeof setInterval> | null = null
  let subscribers = 0

  const fetch = async () => {
    try {
      const { data } = await serviceControlService.getServiceStatus()
      status.value = data
      error.value = false
      lastUpdated.value = Date.now()
    } catch {
      error.value = true
    }
  }

  // Skip work while the tab is backgrounded; the next visible tick refreshes.
  const tick = () => {
    if (typeof document !== 'undefined' && document.hidden) return
    void fetch()
  }

  /**
   * Begin (or join) polling. Returns an unsubscribe function; callers should
   * invoke it on unmount. Default cadence is 5s.
   */
  const startPolling = (intervalMs = 5000): (() => void) => {
    subscribers++
    if (timer === null) {
      void fetch()
      timer = setInterval(tick, intervalMs)
    }
    return () => stopPolling()
  }

  const stopPolling = () => {
    subscribers = Math.max(0, subscribers - 1)
    if (subscribers === 0 && timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  return { status, lastUpdated, error, fetch, startPolling, stopPolling }
})
