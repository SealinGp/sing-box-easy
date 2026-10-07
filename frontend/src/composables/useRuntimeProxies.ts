import { onBeforeUnmount, ref, shallowRef, watch, type Ref } from 'vue'
import { runtimeService } from '../services'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { withGroupDelays } from '../utils/proxyGroups'
import type { RuntimeProxies } from '../types/runtime'

/**
 * A urltest group re-elects its node on its own schedule, so the page would
 * otherwise show a "current" node that stopped being current minutes ago.
 */
const REFRESH_MS = 5_000

/**
 * State and actions for the runtime Proxies page.
 *
 * `enabled` is whether sing-box is running. Everything here is answered by the
 * running process, so there is nothing to fetch while it is down — and a page
 * that polls a stopped service only produces a stream of error toasts.
 */
export function useRuntimeProxies(enabled: Ref<boolean>, fallbackError: () => string) {
  const view = shallowRef<RuntimeProxies | null>(null)
  const loading = ref(false)
  const error = ref('')
  /** Groups with a latency test in flight. */
  const testing = ref<ReadonlySet<string>>(new Set())
  const testingAll = ref(false)
  /** `group\u0000name` of a switch in flight. */
  const switching = ref('')

  let timer: ReturnType<typeof setInterval> | null = null
  // Bumped on every load so a slow reply cannot overwrite a newer one.
  let generation = 0

  const load = async (quiet = false) => {
    const current = ++generation
    if (!quiet) loading.value = true
    try {
      const { data } = await runtimeService.getProxies()
      if (current !== generation) return
      view.value = data
      error.value = ''
    } catch (err) {
      if (current !== generation) return
      error.value = apiErrorMessage(err, fallbackError())
    } finally {
      if (current === generation) loading.value = false
    }
  }

  const replaceGroup = (name: string, update: (group: RuntimeProxies['groups'][number]) => RuntimeProxies['groups'][number]) => {
    if (!view.value) return
    view.value = {
      ...view.value,
      groups: view.value.groups.map((group) => (group.name === name ? update(group) : group)),
    }
  }

  /** Switches a selector. Throws so the caller can toast the reason. */
  const select = async (group: string, name: string) => {
    if (switching.value) return
    switching.value = `${group}\u0000${name}`
    try {
      await runtimeService.select(group, name)
      // Shown at once; the reload then corrects every group that nests this one.
      replaceGroup(group, (current) => ({ ...current, now: name }))
      await load(true)
    } finally {
      switching.value = ''
    }
  }

  const setTesting = (name: string, on: boolean) => {
    const next = new Set(testing.value)
    if (on) next.add(name)
    else next.delete(name)
    testing.value = next
  }

  /** Tests one group's members. Throws so the caller can toast the reason. */
  const testGroup = async (name: string) => {
    if (testing.value.has(name)) return
    setTesting(name, true)
    try {
      const { data } = await runtimeService.testGroup(name)
      replaceGroup(name, (group) => withGroupDelays(group, data.delays))
    } finally {
      setTesting(name, false)
    }
  }

  /**
   * One group at a time, in order. Parallel would be faster and would also ask
   * a router that is busy routing traffic to dial every node it has at once.
   */
  const testAll = async () => {
    if (testingAll.value || !view.value) return
    testingAll.value = true
    try {
      for (const group of view.value.groups) {
        await testGroup(group.name)
      }
      await load(true)
    } finally {
      testingAll.value = false
    }
  }

  const stop = () => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  const start = () => {
    stop()
    void load()
    timer = setInterval(() => {
      // A background tab learns nothing from a refresh, and a test in flight
      // must not have its results overwritten by a stale list.
      if (document.hidden || testing.value.size > 0 || switching.value) return
      void load(true)
    }, REFRESH_MS)
  }

  watch(enabled, (on) => {
    if (on) {
      start()
    } else {
      stop()
      generation++
      view.value = null
      error.value = ''
      loading.value = false
    }
  }, { immediate: true })

  onBeforeUnmount(stop)

  return { view, loading, error, testing, testingAll, switching, load, select, testGroup, testAll }
}
