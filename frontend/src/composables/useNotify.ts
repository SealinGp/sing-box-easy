import { useToast } from 'primevue/usetoast'
import { useI18n } from 'vue-i18n'
import { ApiError } from '../types/api'

/**
 * Single, app-wide notification helper built on PrimeVue's Toast (mounted
 * globally in App.vue, top-center). Every dashboard view should use this
 * instead of calling `toast.add(...)` directly so severity/summary/life
 * defaults stay consistent and API errors are surfaced uniformly.
 *
 * Must be called from a component `setup()` — `useToast()` and `useI18n()`
 * both rely on the active component injection context.
 */
export function useNotify() {
  const toast = useToast()
  const { t } = useI18n()

  const success = (detail: string, summary = t('common.success'), life = 3000) =>
    toast.add({ severity: 'success', summary, detail, life })

  const error = (detail: string, summary = t('common.error'), life = 4000) =>
    toast.add({ severity: 'error', summary, detail, life })

  const info = (detail: string, summary?: string, life = 3000) =>
    toast.add({ severity: 'info', summary, detail, life })

  const warn = (detail: string, summary?: string, life = 3000) =>
    toast.add({ severity: 'warn', summary, detail, life })

  /**
   * Show an error toast for a thrown API/transport failure. The shared axios
   * interceptor (services/api.ts) throws an ApiError carrying the backend's
   * `msg`, so the most informative text lives on `error.message`. Prefer it,
   * falling back to a caller-supplied message when none is present.
   */
  const apiError = (err: unknown, fallbackMsg: string, summary = t('common.error')) => {
    const detail =
      (err instanceof ApiError || err instanceof Error) && err.message.trim() !== ''
        ? err.message
        : fallbackMsg
    error(detail, summary)
  }

  return { success, error, info, warn, apiError }
}
