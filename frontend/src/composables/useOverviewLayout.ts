import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { userService } from '../services'
import { useDeployment } from './useDeployment'

export function useOverviewLayout(defaults: string[]) {
  const { t } = useI18n()
  const { authEnabled } = useDeployment()
  const order = ref([...defaults])
  const hidden = ref<string[]>([])
  const enabled = ref(false)
  const saving = ref(false)
  const loading = ref(true)
  const error = ref('')
  const message = ref('')
  const key = 'sbe-overview-layout:anonymous'
  let saved = { overview_order: [...defaults], overview_hidden: [] as string[] }
  const ids = (value: unknown): string[] => Array.isArray(value)
    ? [...new Set(value.filter((id): id is string => typeof id === 'string' && defaults.includes(id)))] : []
  const snapshot = () => ({ overview_order: [...order.value], overview_hidden: [...hidden.value] })
  const dirty = computed(() => JSON.stringify(snapshot()) !== JSON.stringify(saved))
  async function load() {
    try {
      const value = authEnabled.value ? await userService.getPreferences()
        : JSON.parse(localStorage.getItem(key) ?? 'null') ?? {
          overview_order: JSON.parse(localStorage.getItem('sbe-overview-order:anonymous') ?? '[]'),
        }
      order.value = [...new Set([...ids(value.overview_order), ...defaults])]
      hidden.value = ids(value.overview_hidden)
      saved = snapshot()
    } catch { error.value = t('overview.layout.loadError') }
    finally { loading.value = false }
  }
  function start() { enabled.value = true; error.value = ''; message.value = '' }
  function cancel() {
    if (saving.value) return
    order.value = [...saved.overview_order]
    hidden.value = [...saved.overview_hidden]
    enabled.value = false
    error.value = ''
  }
  async function save() {
    if (saving.value) return
    if (!dirty.value) { enabled.value = false; return }
    saving.value = true
    error.value = ''
    try {
      const value = snapshot()
      if (authEnabled.value) await userService.updatePreferences(value)
      else localStorage.setItem(key, JSON.stringify(value))
      saved = value
      enabled.value = false
      message.value = t('overview.layout.saved')
    } catch { error.value = t('overview.layout.saveError') }
    finally { saving.value = false }
  }
  function hide(id: string) {
    if (saving.value || hidden.value.includes(id)) return
    hidden.value = [...hidden.value, id]
  }
  function add(slotId: string, id: string) {
    if (saving.value || !hidden.value.includes(id) || !hidden.value.includes(slotId)) return
    const next = [...order.value]
    const target = next.indexOf(slotId), from = next.indexOf(id)
    if (target < 0 || from < 0) return
    ;[next[target], next[from]] = [next[from]!, next[target]!]
    order.value = next
    hidden.value = hidden.value.filter(item => item !== id)
  }
  function nudge(from: number, delta: number) {
    if (saving.value) return
    const visible = order.value
    const id = visible[from], target = visible[from + delta]
    if (!id || !target) return
    const next = [...order.value]
    next.splice(next.indexOf(id), 1)
    next.splice(next.indexOf(target) + (delta > 0 ? 1 : 0), 0, id)
    order.value = next
  }
  function reset() { if (!saving.value) { order.value = [...defaults]; hidden.value = [] } }
  return { order, hidden, enabled, saving, loading, error, message, dirty, authEnabled, load, start, cancel, save, hide, add, nudge, reset }
}
