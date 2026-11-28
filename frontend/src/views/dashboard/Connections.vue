<script setup lang="ts">
import TooltipTarget from "../../components/TooltipTarget.vue"
/**
 * Live connections of the running sing-box: who is talking to what, through
 * which rule and exit, and how fast — with the means to end one.
 *
 * The totals (speed, connection count, memory) are NOT here: they are a
 * summary of the system, and live on the Overview in `ConnectionsOverviewCard`
 * with their history. This page is the list.
 *
 * The Overview diagram answers "is my config doing what I meant" in aggregate;
 * this is the same data one connection at a time, for chasing a single device
 * or a single site.
 */
import Tooltip from '../../components/Tooltip.vue'
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { InformationCircleIcon, WrenchScrewdriverIcon } from '@heroicons/vue/24/outline'
import ConnectionSourcePicker from '../../components/ConnectionSourcePicker.vue'
import ConnectionTableSettings from '../../components/ConnectionTableSettings.vue'
import ConnectionsFlow from '../../components/ConnectionsFlow.vue'
import { connectionView } from '../../utils/connectionFlow'
import { useConnections } from '../../composables/useConnections'
import { useConnectionLinks } from '../../composables/useConnectionLinks'
import { useFillHeight } from '../../composables/useFillHeight'
import { useConnectionTableSettings } from '../../composables/useConnectionTableSettings'
import { useConfirm } from '../../composables/useConfirm'
import { useNotify } from '../../composables/useNotify'
import { runtimeService } from '../../services'
import { useServiceStore } from '../../stores/service'
import {
  CLOSED_LIMIT, NUMERIC_COLUMNS, poolFor, sourceOptions, viewRows,
  type ColumnKey, type ConnectionTab, type TrackedConnection,
} from '../../utils/connectionsTable'

const { t } = useI18n()
const notify = useNotify()
const { confirm } = useConfirm()
const serviceStore = useServiceStore()
const running = computed(() => serviceStore.status?.status === 'running')

const { active, closed, totals, now, error, connecting, paused, clearClosed } =
  useConnections(running, () => t('connections.streamFailed'))

// What a connection's details may link to (proxy groups, rule sets). Loaded
// once per visit here, not by each expanded row.
useConnectionLinks().refresh()
const { settings, update, reset } = useConnectionTableSettings()

const tableRegion = ref<HTMLElement | null>(null)
useFillHeight(tableRegion, ref(false))

const tab = ref<ConnectionTab>('active')
const TABS: ConnectionTab[] = ['active', 'closed', 'all']
const query = ref('')
const source = ref('')
const sortKey = ref<ColumnKey>('downRate')
const ascending = ref(false)
const selectedId = ref('')
// A connection ending must not destroy a proxy card being inspected/switched.
// Keep its final snapshot until the person closes the details or changes view.
const retainedDetail = shallowRef<TrackedConnection | null>(null)
watch([selectedId, active, closed], () => {
  if (!selectedId.value) { retainedDetail.value = null; return }
  const latest = active.value.find(row => row.id === selectedId.value)
    ?? closed.value.find(row => row.id === selectedId.value)
  if (latest) retainedDetail.value = latest
  else if (retainedDetail.value?.id === selectedId.value && !retainedDetail.value.closed) {
    retainedDetail.value = { ...retainedDetail.value, closed: true, closedAt: now.value, up_rate: 0, down_rate: 0 }
  }
})
function readView() {
  try { return connectionView(localStorage.getItem('sbe-connections-view')) } catch { return connectionView(null) }
}
const mode = ref(readView())
watch(mode, value => { try { localStorage.setItem('sbe-connections-view', value) } catch { /* Optional preference. */ } })
const settingsOpen = ref(false)
const settingsAnchor = ref<HTMLElement | null>(null)
const closing = ref<ReadonlySet<string>>(new Set())
const closingAll = ref(false)

/**
 * Rows drawn at once. A router with a few thousand connections would otherwise
 * rebuild tens of thousands of cells every second; the sort puts the rows that
 * matter first, and the footer says how many were left out.
 */
const RENDER_LIMIT = 60
const page = ref(1)

const pool = computed(() => poolFor(tab.value, active.value, closed.value))
const devices = computed(() => {
  const options = sourceOptions(pool.value, settings.value.labels)
  const retained = selectedId.value ? retainedDetail.value : null
  if (retained && !options.some(option => option.ip === retained.source_ip)) {
    options.push({ ip: retained.source_ip, label: settings.value.labels[retained.source_ip] || retained.source_ip, count: 0 })
  }
  return options
})

// A device whose connections all ended leaves the filter pointing at nothing,
// which would read as an empty table with no visible reason.
watch(devices, (options) => {
  if (source.value && !options.some((option) => option.ip === source.value)) source.value = ''
})

const rows = computed(() => viewRows(active.value, closed.value, {
  tab: tab.value,
  query: query.value,
  source: source.value,
  sortKey: sortKey.value,
  ascending: ascending.value,
  labels: settings.value.labels,
  now: now.value,
}))
/**
 * While a row's details are open, the order is HELD.
 *
 * The table re-sorts every second — by download rate, by default — so an open
 * row would otherwise slide up and down the list with its details attached,
 * and the thing being read would keep moving out from under the eye. Values
 * still update in place; only positions are frozen. Connections that arrive
 * meanwhile are appended in their sorted order, and closing the details (or
 * changing the sort, tab or filter) releases the hold. Nothing announces the
 * hold: a note appearing above the table would itself shift every row, which
 * is the very thing being prevented.
 */
const heldOrder = ref<Map<string, number> | null>(null)

const ordered = computed(() => {
  const retained = retainedDetail.value
  const visible = retained?.closed && !rows.value.some(row => row.id === retained.id)
    ? [...rows.value, retained] : rows.value
  const held = heldOrder.value
  if (!held) return visible
  const position = (id: string) => held.get(id) ?? Number.MAX_SAFE_INTEGER
  // Array.prototype.sort is stable, so newcomers keep their sorted order.
  return [...visible].sort((a, b) => position(a.id) - position(b.id))
})

const heldPage = ref<Set<string> | null>(null)
const drawn = computed(() => heldPage.value
  ? ordered.value.filter(row => heldPage.value!.has(row.id))
  : ordered.value.slice((page.value - 1) * RENDER_LIMIT, page.value * RENDER_LIMIT))

watch([() => ordered.value.length, selectedId], ([length, selected]) => {
  if (!selected) page.value = Math.min(page.value, Math.max(1, Math.ceil(length / RENDER_LIMIT)))
})
watch(page, () => { selectedId.value = ''; releaseOrder(); tableRegion.value?.scrollTo({ top: 0 }) })

const holdOrder = () => {
  heldOrder.value = new Map(ordered.value.map((row, index) => [row.id, index]))
}
const releaseOrder = () => {
  heldOrder.value = null
  heldPage.value = null
}

const counts = computed<Record<ConnectionTab, number>>(() => ({
  active: active.value.length,
  closed: closed.value.length,
  all: active.value.length + closed.value.length,
}))

/** Every source seen this session, so a device can be named after it went quiet. */
const knownSources = computed(() =>
  [...new Set([...active.value, ...closed.value].map((row) => row.source_ip))]
    .sort((a, b) => a.localeCompare(b, undefined, { numeric: true })))

const onSort = (key: ColumnKey) => {
  releaseOrder()
  if (sortKey.value === key) {
    ascending.value = !ascending.value
    return
  }
  sortKey.value = key
  // Numbers are read largest-first ("what is using the bandwidth"), text A→Z.
  ascending.value = !NUMERIC_COLUMNS.has(key)
}

const onSelect = (id: string) => {
  if (selectedId.value === id) {
    selectedId.value = ''
    releaseOrder()
    return
  }
  heldPage.value = new Set(drawn.value.map(row => row.id))
  // Capture the order as it is on screen NOW, before anything can move.
  if (!heldOrder.value) holdOrder()
  selectedId.value = id
}

// A new tab, search or device is a new list: holding the old order over it
// would be sorting by positions that no longer mean anything.
watch([tab, query, source], () => {
  page.value = 1
  selectedId.value = ''
  releaseOrder()
})

const closeOne = async (id: string) => {
  if (closing.value.has(id)) return
  closing.value = new Set([...closing.value, id])
  try {
    await runtimeService.closeConnection(id)
  } catch (err) {
    notify.apiError(err, t('connections.closeFailed'))
  } finally {
    const next = new Set(closing.value)
    next.delete(id)
    closing.value = next
  }
}

const closeAll = async () => {
  const ok = await confirm({
    title: t('connections.closeAll'),
    message: t('connections.closeAllConfirm', { n: active.value.length }),
    confirmLabel: t('connections.closeAll'),
    tone: 'danger',
  })
  if (!ok) return
  closingAll.value = true
  try {
    await runtimeService.closeAllConnections()
  } catch (err) {
    notify.apiError(err, t('connections.closeFailed'))
  } finally {
    closingAll.value = false
  }
}

type TipTone = 'info' | 'warn' | 'error'

/**
 * What the info icon says right now: the current state first, then the two
 * standing facts about this page. The standing ones are always present, so the
 * tooltip is never empty and the icon never has to appear or disappear.
 */
const tips = computed<{ text: string; tone: TipTone }[]>(() => {
  const list: { text: string; tone: TipTone }[] = []
  if (!running.value) list.push({ text: t('connections.stopped'), tone: 'warn' })
  else if (error.value) list.push({ text: error.value, tone: 'error' })
  else if (connecting.value && !totals.value) list.push({ text: t('connections.connecting'), tone: 'info' })
  else if (paused.value) list.push({ text: t('connections.pausedNote'), tone: 'info' })
  list.push({ text: t('connections.tips.refresh'), tone: 'info' })
  list.push({ text: t('connections.closedNote', { max: CLOSED_LIMIT }), tone: 'info' })
  return list
})

/** The most serious tone present; it colours the icon. */
const tipTone = computed<TipTone>(() =>
  tips.value.some((tip) => tip.tone === 'error') ? 'error' : tips.value.some((tip) => tip.tone === 'warn') ? 'warn' : 'info')
</script>

<template>
  <div class="page-shell space-y-3">
    <div class="flex flex-wrap items-center gap-2">
      <div class="inline-flex gap-0.5 rounded-control bg-gray-100 p-0.5 dark:bg-gray-700/60" role="tablist" :aria-label="$t('connections.tabs.label')">
        <button
          v-for="name in TABS"
          :key="name"
          type="button"
          role="tab"
          :aria-selected="tab === name"
          class="h-7 whitespace-nowrap rounded px-2.5 text-sm"
          :class="tab === name ? 'bg-white font-semibold text-gray-900 shadow-sm dark:bg-gray-800 dark:text-gray-100' : 'text-gray-500 dark:text-gray-400'"
          @click="tab = name"
        >
          {{ $t(`connections.tabs.${name}`) }}<b class="ml-1 font-semibold tabular-nums">{{ counts[name] }}</b>
        </button>
      </div>
      <!--
        Status and caveats live behind this icon, not on a line of their own.
        A line that appears and disappears moves the whole table each time; a
        line that is always reserved spends height on being empty. The icon is
        always here, so nothing shifts — and it changes COLOUR when there is
        something that needs reading (stopped, stream dropped), because a
        tooltip nobody thinks to open is not a warning.
      -->
      <Tooltip class="tips" :class="`is-${tipTone}`" :width="352">
        <template #default="{ describedBy }">
          <button type="button" class="tips-icon" :aria-label="$t('connections.tips.label')" :aria-describedby="describedBy">
            <InformationCircleIcon class="h-[18px] w-[18px]" />
          </button>
        </template>
        <template #content><div class="flex flex-col gap-1.5">
          <span v-for="tip in tips" :key="tip.text" class="tips-line" :class="`is-${tip.tone}`">{{ tip.text }}</span>
        </div></template>
      </Tooltip>
      <input
        v-model="query"
        type="search"
        :placeholder="$t('connections.searchPlaceholder')"
        :aria-label="$t('connections.searchPlaceholder')"
        class="h-8 min-w-0 flex-[1_1_200px] rounded-control border border-gray-300 bg-white px-2.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
      />
      <ConnectionSourcePicker v-model="source" :options="devices" :total="pool.length" />
      <TooltipTarget as="button"
        type="button"
        class="inline-flex h-8 w-8 items-center justify-center rounded-control border border-gray-300 bg-white text-gray-700 hover:border-primary-500 hover:text-primary-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300"
        :aria-label="$t('connections.settings.title')"
        :text="$t('connections.settings.title')"
        @click="(event: MouseEvent) => { settingsAnchor = event.currentTarget as HTMLElement; settingsOpen = !settingsOpen }"
      >
        <WrenchScrewdriverIcon class="h-4 w-4" />
      </TooltipTarget>
      <button
        type="button"
        class="h-8 rounded-control border px-3 text-sm font-medium"
        :class="paused
          ? 'border-primary-500 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
          : 'border-gray-300 bg-white text-gray-700 hover:border-primary-500 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300'"
        :aria-pressed="paused"
        @click="paused = !paused"
      >
        {{ paused ? $t('connections.resume') : $t('connections.pause') }}
      </button>
      <button
        v-if="tab === 'closed'"
        type="button"
        class="h-8 rounded-control border border-gray-300 bg-white px-3 text-sm font-medium text-gray-700 hover:border-primary-500 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300"
        :disabled="closed.length === 0"
        @click="clearClosed"
      >
        {{ $t('connections.clearClosed') }}
      </button>
      <button
        v-else
        type="button"
        class="h-8 rounded-control border border-red-400 bg-white px-3 text-sm font-medium text-red-600 hover:bg-red-50 disabled:cursor-not-allowed disabled:opacity-50 dark:bg-gray-800 dark:hover:bg-red-950/40"
        :disabled="active.length === 0 || closingAll"
        @click="closeAll"
      >
        {{ $t('connections.closeAll') }}
      </button>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      <div class="inline-flex rounded-control border border-border p-0.5" :aria-label="$t('connections.flow.view')" role="group">
        <button v-for="view in (['groups', 'compact'] as const)" :key="view" type="button" :aria-pressed="mode === view"
          class="rounded px-3 py-1.5 text-xs" :class="mode === view ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'text-text-secondary'" @click="mode = view">
          {{ $t(`connections.flow.${view}`) }}
        </button>
      </div>
      <label class="ml-auto flex items-center gap-2 text-xs">
        {{ $t('connections.flow.sort') }}
        <select :value="sortKey" class="max-w-36 rounded-control border border-border bg-bg-secondary px-2 py-1.5" @change="onSort(($event.target as HTMLSelectElement).value as ColumnKey)">
          <option v-for="key in ['downRate', 'upRate', 'source', 'host', 'duration', 'start']" :key="key" :value="key">{{ $t(`connections.columns.${key}`) }}</option>
        </select>
      </label>
      <TooltipTarget as="button" type="button" class="h-8 w-8 rounded-control border border-border" :aria-label="$t(ascending ? 'connections.flow.descending' : 'connections.flow.ascending')" :text="$t(ascending ? 'connections.flow.descending' : 'connections.flow.ascending')" @click="onSort(sortKey)">{{ ascending ? '↑' : '↓' }}</TooltipTarget>
    </div>
    <!--
      The table scrolls INSIDE this region, sized by `useFillHeight` to the
      space left in the window — the same behaviour as the Outbounds list, so
      the toolbar stays put and only the rows move.
    -->
    <div ref="tableRegion" class="scroll-region rounded-table border border-border bg-bg-secondary"
      @pointerenter="!selectedId && holdOrder()" @pointerleave="!selectedId && releaseOrder()">
      <ConnectionsFlow
        :mode="mode"
        :rows="drawn"
        :settings="settings"
        :selected-id="selectedId"
        :now="now"
        :closing="closing"
        @select="onSelect"
        @close="closeOne"
      />
      <p v-if="drawn.length === 0" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ pool.length === 0 ? $t(`connections.empty.${tab}`) : $t('connections.noMatch') }}
      </p>
    </div>

    <div v-if="ordered.length > RENDER_LIMIT" class="flex items-center justify-center gap-3 text-xs">
      <button type="button" :disabled="page === 1" class="rounded-control border border-border px-3 py-2 disabled:opacity-40" @click="page--">{{ $t('connections.flow.previous') }}</button>
      <span>{{ $t('connections.flow.page', { page, pages: Math.max(page, Math.ceil(ordered.length / RENDER_LIMIT)), total: rows.length }) }}</span>
      <button type="button" :disabled="page * RENDER_LIMIT >= ordered.length" class="rounded-control border border-border px-3 py-2 disabled:opacity-40" @click="page++">{{ $t('connections.flow.next') }}</button>
    </div>

    <ConnectionTableSettings
      flow-only
      v-model:visible="settingsOpen"
      :anchor="settingsAnchor"
      :settings="settings"
      :sources="knownSources"
      @update="update"
      @reset="reset"
    />
  </div>
</template>

<style scoped>
.tips {
  position: relative;
  display: inline-flex;
}
.tips-icon {
  display: inline-flex;
  width: 1.75rem;
  height: 1.75rem;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  color: var(--color-text-secondary);
  cursor: help;
  transition: color 0.15s ease;
}
.tips-icon:hover,
.tips-icon:focus-visible {
  color: var(--color-primary);
}
.tips-icon:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 1px;
}
.tips.is-warn .tips-icon {
  color: var(--color-warning);
}
.tips.is-error .tips-icon {
  color: var(--color-danger);
}

.tips-line.is-warn {
  color: var(--color-warning);
  font-weight: 600;
}
.tips-line.is-error {
  color: var(--color-danger);
  font-weight: 600;
}
</style>
