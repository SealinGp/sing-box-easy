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

/**
 * `'auto'` follows the detected platform; the other two force a layout.
 */
export type LayoutOverride = 'auto' | 'sidebar' | 'topbar'

const LAYOUT_OVERRIDE_KEY = 'sb_layout_override'

const isLayoutOverride = (value: unknown): value is LayoutOverride =>
  value === 'auto' || value === 'sidebar' || value === 'topbar'

/**
 * Reads the stored preference. Storage can throw outright (Safari private
 * browsing) and can hold anything, so both are handled — a bad value simply
 * means "follow the platform".
 */
const readStoredOverride = (): LayoutOverride => {
  try {
    const stored = localStorage.getItem(LAYOUT_OVERRIDE_KEY)
    return isLayoutOverride(stored) ? stored : 'auto'
  } catch {
    return 'auto'
  }
}

// Read at module load, before the first render, so a forced layout is what
// paints initially rather than something that swaps in afterwards.
const layoutOverride = ref<LayoutOverride>(readStoredOverride())

const setLayoutOverride = (value: LayoutOverride) => {
  layoutOverride.value = value
  try {
    if (value === 'auto') {
      localStorage.removeItem(LAYOUT_OVERRIDE_KEY)
    } else {
      localStorage.setItem(LAYOUT_OVERRIDE_KEY, value)
    }
  } catch (err) {
    // Non-fatal: the override still applies for this session.
    console.warn('Could not persist the layout override:', err)
  }
}

/** Platform detection also controls setup defaults; appearance must not change it. */
const isOpenWrt = computed(() => systemType.value === 'openwrt')
const prefersTopbar = computed(() => layoutOverride.value === 'auto'
  ? isOpenWrt.value
  : layoutOverride.value === 'topbar')

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
  prefersTopbar,
  layoutOverride,
  setLayoutOverride,
  ensureDeployment,
})
