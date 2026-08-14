import { ref } from 'vue'
import { userService } from '../services'

// Cached deployment auth mode. `true` (the default) means login is required;
// the backend reports `false` when authentication is disabled (server.auth:
// disabled, or "auto" on OpenWrt) and the UI then skips the login flow and
// hides the profile / user-management surfaces.
const authEnabled = ref(true)
let loaded: Promise<boolean> | null = null

// ensureAuthMode fetches /auth/status once and caches the answer for the
// session. On failure it keeps the safe default (login required) so a flaky
// backend never silently opens the UI.
export const ensureAuthMode = (): Promise<boolean> => {
  if (!loaded) {
    loaded = userService
      .getAuthStatus()
      .then((status) => {
        authEnabled.value = status.auth_enabled
        return authEnabled.value
      })
      .catch(() => {
        loaded = null // allow a retry on the next navigation
        return true
      })
  }
  return loaded
}

export const useAuthMode = () => ({ authEnabled, ensureAuthMode })
