<script setup lang="ts">
/**
 * Runtime proxies: which node each group is using right now, how fast each
 * member last tested, and switching a selector by hand.
 *
 * This is the running process, not the config — the Outbounds page edits what
 * groups ARE; this page shows and steers what sing-box is DOING with them. A
 * switch here is never written to config.json.
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ProxyGroupCard from '../../components/ProxyGroupCard.vue'
import { useRuntimeProxies } from '../../composables/useRuntimeProxies'
import { useNotify } from '../../composables/useNotify'
import { useServiceStore } from '../../stores/service'
import { columnCount, dealIntoColumns, visibleGroups, type MemberSort } from '../../utils/proxyGroups'

const { t } = useI18n()
const notify = useNotify()
const serviceStore = useServiceStore()
const running = computed(() => serviceStore.status?.status === 'running')

const { view, loading, error, testing, testingAll, switching, load, select, testGroup, testAll } =
  useRuntimeProxies(running, () => t('proxies.loadFailed'))

const query = ref('')
const sort = ref<MemberSort>('default')
const SORTS: { value: MemberSort; labelKey: string }[] = [
  { value: 'default', labelKey: 'proxies.sort.default' },
  { value: 'latency', labelKey: 'proxies.sort.latency' },
  { value: 'name', labelKey: 'proxies.sort.name' },
]

const visible = computed(() => visibleGroups(view.value?.groups ?? [], query.value, sort.value))

/**
 * Cards are laid out in independent COLUMNS, not grid rows.
 *
 * In a row-based grid every card in a row shares the row's height, so one
 * expanded card left a tall empty band under each collapsed neighbour — and
 * expanding it pushed down every card on the rows below, in all columns. With
 * columns, a card's height belongs to its own column: opening one moves only
 * the cards beneath it, and the others stay exactly where they were.
 *
 * The column count comes from the measured width, and cards are dealt
 * round-robin (see `dealIntoColumns` for why not shortest-first).
 */
const layout = ref<HTMLElement | null>(null)
const layoutWidth = ref(0)
let layoutObserver: ResizeObserver | null = null

// A watcher rather than onMounted: the element is behind a v-if and only
// exists once the groups have loaded.
watch(layout, (el) => {
  layoutObserver?.disconnect()
  layoutObserver = null
  if (!el) return
  layoutWidth.value = el.clientWidth
  layoutObserver = new ResizeObserver((entries) => {
    layoutWidth.value = entries[0]?.contentRect.width ?? 0
  })
  layoutObserver.observe(el)
})
onBeforeUnmount(() => layoutObserver?.disconnect())

const columns = computed(() => dealIntoColumns(visible.value, columnCount(layoutWidth.value)))

/**
 * Which cards are expanded, remembered per browser.
 *
 * `null` means "never chosen": selectors open (they are the ones with something
 * to click) and automatic groups closed. Once the operator touches any card the
 * explicit set takes over, so a card they closed does not reopen on reload.
 */
const OPEN_KEY = 'sbe-proxies-open'
const readOpen = (): Set<string> | null => {
  try {
    const raw = localStorage.getItem(OPEN_KEY)
    if (raw === null) return null
    const parsed: unknown = JSON.parse(raw)
    return Array.isArray(parsed) ? new Set(parsed.filter((item): item is string => typeof item === 'string')) : null
  } catch {
    return null
  }
}
const openSet = ref<Set<string> | null>(readOpen())

const isOpen = (name: string, switchable: boolean) => (openSet.value ? openSet.value.has(name) : switchable)

const currentOpen = (): Set<string> =>
  new Set((view.value?.groups ?? []).filter((group) => isOpen(group.name, group.switchable)).map((group) => group.name))

watch(openSet, (value) => {
  if (!value) return
  try {
    localStorage.setItem(OPEN_KEY, JSON.stringify([...value]))
  } catch {
    // Private mode or a full quota: the page works, it just forgets.
  }
})

const toggle = (name: string) => {
  const next = currentOpen()
  if (next.has(name)) next.delete(name)
  else next.add(name)
  openSet.value = next
}

const allOpen = computed(() => {
  const groups = view.value?.groups ?? []
  return groups.length > 0 && groups.every((group) => isOpen(group.name, group.switchable))
})

const toggleAll = () => {
  openSet.value = allOpen.value ? new Set() : new Set((view.value?.groups ?? []).map((group) => group.name))
}

const switchingTo = (group: string) => {
  const [g, name] = switching.value.split('\u0000')
  return g === group ? name ?? '' : ''
}

const onSelect = async (group: string, name: string) => {
  try {
    await select(group, name)
  } catch (err) {
    notify.apiError(err, t('proxies.selectFailed'))
    void load(true)
  }
}

const onTest = async (group: string) => {
  try {
    await testGroup(group)
    void load(true)
  } catch (err) {
    notify.apiError(err, t('proxies.testFailed'))
  }
}

const onTestAll = async () => {
  try {
    await testAll()
  } catch (err) {
    notify.apiError(err, t('proxies.testFailed'))
  }
}
</script>

<template>
  <div class="page-shell space-y-3">
    <div class="flex flex-wrap items-center gap-2">
      <input
        v-model="query"
        type="search"
        :placeholder="$t('proxies.searchPlaceholder')"
        :aria-label="$t('proxies.searchPlaceholder')"
        class="h-8 min-w-0 flex-[1_1_200px] rounded-control border border-gray-300 bg-white px-2.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
      />
      <select
        v-model="sort"
        :aria-label="$t('proxies.sort.label')"
        class="h-8 rounded-control border border-gray-300 bg-white px-2 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
      >
        <option v-for="option in SORTS" :key="option.value" :value="option.value">{{ $t(option.labelKey) }}</option>
      </select>
      <button
        type="button"
        class="h-8 rounded-control border border-gray-300 bg-white px-3 text-sm font-medium text-gray-700 transition-colors hover:border-primary-500 hover:text-primary-700 disabled:cursor-wait disabled:opacity-60 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300"
        :disabled="!view || testingAll"
        @click="onTestAll"
      >
        {{ testingAll ? $t('proxies.testing') : $t('proxies.testAll') }}
      </button>
      <button
        type="button"
        class="h-8 rounded-control border border-gray-300 bg-white px-3 text-sm font-medium text-gray-700 transition-colors hover:border-primary-500 hover:text-primary-700 disabled:opacity-60 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300"
        :disabled="!view"
        @click="toggleAll"
      >
        {{ allOpen ? $t('proxies.collapseAll') : $t('proxies.expandAll') }}
      </button>
    </div>

    <p v-if="view" class="text-xs" :class="view.selection_persisted ? 'text-gray-500 dark:text-gray-400' : 'text-amber-600 dark:text-amber-400'">
      {{ view.selection_persisted ? $t('proxies.persisted') : $t('proxies.notPersisted') }}
    </p>

    <div
      v-if="!running"
      class="rounded-surface border border-dashed border-gray-300 px-4 py-10 text-center text-sm text-gray-500 dark:border-gray-600 dark:text-gray-400"
    >
      {{ $t('proxies.stopped') }}
    </div>

    <div
      v-else-if="error && !view"
      class="rounded-surface border border-red-300 bg-red-50 px-4 py-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/40 dark:text-red-300"
    >
      <p>{{ error }}</p>
      <button type="button" class="mt-1 text-xs underline" @click="load()">{{ $t('proxies.retry') }}</button>
    </div>

    <div v-else-if="loading && !view" class="flex justify-center py-10">
      <div class="h-7 w-7 animate-spin rounded-pill border-b-2 border-primary-500"></div>
    </div>

    <template v-else-if="view">
      <p v-if="error" class="text-xs text-red-600 dark:text-red-400">{{ error }}</p>
      <div
        v-if="visible.length === 0"
        class="rounded-surface border border-dashed border-gray-300 px-4 py-10 text-center text-sm text-gray-500 dark:border-gray-600 dark:text-gray-400"
      >
        {{ view.groups.length === 0 ? $t('proxies.noGroups') : $t('proxies.noMatch') }}
      </div>
      <div v-else ref="layout" class="flex items-start gap-3">
        <div v-for="(column, index) in columns" :key="index" class="flex min-w-0 flex-1 basis-0 flex-col gap-3">
        <ProxyGroupCard
          v-for="entry in column"
          :key="entry.group.name"
          :group="entry.group"
          :members="entry.members"
          :open="isOpen(entry.group.name, entry.group.switchable)"
          :testing="testing.has(entry.group.name)"
          :switching-to="switchingTo(entry.group.name)"
          @toggle="toggle(entry.group.name)"
          @test="onTest(entry.group.name)"
          @select="(name) => onSelect(entry.group.name, name)"
        />
        </div>
      </div>
    </template>
  </div>
</template>
