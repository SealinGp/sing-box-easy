import { computed, ref } from 'vue'
import { userService } from '../services'
import type { SystemType } from '../types/api'

/**
 * Cached answers to "how is this deployment configured?" — fetched once from
 * the public /auth/status endpoint and shared by every consumer.
 *
 * Two things depend on it:
 *  - `authEnabled`: whether there is a login flow at all. `true` is the safe
 *    default, so a failed fetch never silently opens the UI.
 *  - `systemType`: the platform family. OpenWrt routers render a top bar
 *    instead of a sidebar, because LuCI already occupies the left edge of the
 *    operator's screen.
 */
const authEnabled = ref(true)
const systemType = ref<SystemType>('unknown')

/** OpenWrt gets the horizontal navigation layout. */
const isOpenWrt = computed(() => systemType.value === 'openwrt')

let loaded: Promise<void> | null = null

/**
 * Fetches the deployment status once and caches it for the session. On failure
 * it keeps the safe defaults (login required, unknown platform) and clears the
 * cache so the next navigation retries.
 */
export const ensureDeployment = (): Promise<void> => {
  if (!loaded) {
    loaded = userService
      .getAuthStatus()
      .then((status) => {
        authEnabled.value = status.auth_enabled
        systemType.value = status.system_type ?? 'unknown'
      })
      .catch(() => {
        loaded = null // allow a retry on the next navigation
      })
  }
  return loaded
}

export const useDeployment = () => ({
  authEnabled,
  systemType,
  isOpenWrt,
  ensureDeployment,
})
