import { ref, watch } from 'vue'
import { defaultSettings, normalizeSettings, type TableSettings } from '../utils/connectionsTable'

const STORAGE_KEY = 'sbe-connections-table'

/**
 * Column layout and device labels for the Connections table, per browser.
 *
 * localStorage rather than the panel's settings API: this is a view preference
 * — which columns one person wants on one screen — and a phone and a desktop
 * reasonably disagree about it.
 */
export function useConnectionTableSettings() {
  const read = (): TableSettings => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      return raw === null ? defaultSettings() : normalizeSettings(JSON.parse(raw))
    } catch {
      return defaultSettings()
    }
  }

  const settings = ref<TableSettings>(read())

  watch(settings, (value) => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
    } catch {
      // Private mode or a full quota: the table works, it just forgets.
    }
  }, { deep: true })

  /** Replaces the settings with a patched copy. */
  const update = (patch: Partial<TableSettings>) => {
    settings.value = { ...settings.value, ...patch }
  }

  const reset = () => {
    settings.value = defaultSettings()
  }

  return { settings, update, reset }
}
