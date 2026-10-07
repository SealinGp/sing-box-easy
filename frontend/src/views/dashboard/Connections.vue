<script setup lang="ts">
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
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { InformationCircleIcon, WrenchScrewdriverIcon } from '@heroicons/vue/24/outline'
import ConnectionSourcePicker from '../../components/ConnectionSourcePicker.vue'
import ConnectionTableSettings from '../../components/ConnectionTableSettings.vue'
import ConnectionsTable from '../../components/ConnectionsTable.vue'
import { useConnections } from '../../composables/useConnections'
import { useFillHeight } from '../../composables/useFillHeight'
import { useConnectionTableSettings } from '../../composables/useConnectionTableSettings'
import { useConfirm } from '../../composables/useConfirm'
import { useNotify } from '../../composables/useNotify'
import { runtimeService } from '../../services'
import { useServiceStore } from '../../stores/service'
import {
  CLOSED_LIMIT, NUMERIC_COLUMNS, poolFor, sourceOptions, viewRows,
  type ColumnKey, type ConnectionTab,
} from '../../utils/connectionsTable'

const { t } = useI18n()
const notify = useNotify()
const { confirm } = useConfirm()
const serviceStore = useServiceStore()
const running = computed(() => serviceStore.status?.status === 'running')

const { active, closed, totals, now, error, connecting, paused, clearClosed } =
  useConnections(running, () => t('connections.streamFailed'))
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
const settingsOpen = ref(false)
const closing = ref<ReadonlySet<string>>(new Set())
const closingAll = ref(false)

/**
 * Rows drawn at once. A router with a few thousand connections would otherwise
 * rebuild tens of thousands of cells every second; the sort puts the rows that
 * matter first, and the footer says how many were left out.
 */
const RENDER_LIMIT = 300

const pool = computed(() => poolFor(tab.value, active.value, closed.value))
const devices = computed(() => sourceOptions(pool.value, settings.value.labels))

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
  const held = heldOrder.value
  if (!held) return rows.value
  const position = (id: string) => held.get(id) ?? Number.MAX_SAFE_INTEGER
  // Array.prototype.sort is stable, so newcomers keep their sorted order.
  return [...rows.value].sort((a, b) => position(a.id) - position(b.id))
})

const drawn = computed(() => ordered.value.slice(0, RENDER_LIMIT))

const holdOrder = () => {
  heldOrder.value = new Map(ordered.value.map((row, index) => [row.id, index]))
}
const releaseOrder = () => {
  heldOrder.value = null
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
  // Capture the order as it is on screen NOW, before anything can move.
  if (!heldOrder.value) holdOrder()
  selectedId.value = id
}

// A new tab, search or device is a new list: holding the old order over it
// would be sorting by positions that no longer mean anything.
watch([tab, query, source], () => {
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
      <span class="tips" :class="`is-${tipTone}`">
        <button type="button" class="tips-icon" :aria-label="$t('connections.tips.label')" aria-describedby="connections-tips">
          <InformationCircleIcon class="h-[18px] w-[18px]" />
        </button>
        <span id="connections-tips" role="tooltip" class="tips-panel">
          <span v-for="tip in tips" :key="tip.text" class="tips-line" :class="`is-${tip.tone}`">{{ tip.text }}</span>
        </span>
      </span>
      <input
        v-model="query"
        type="search"
        :placeholder="$t('connections.searchPlaceholder')"
        :aria-label="$t('connections.searchPlaceholder')"
        class="h-8 min-w-0 flex-[1_1_200px] rounded-control border border-gray-300 bg-white px-2.5 text-sm text-gray-900 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100"
      />
      <ConnectionSourcePicker v-model="source" :options="devices" :total="pool.length" />
      <button
        type="button"
        class="inline-flex h-8 w-8 items-center justify-center rounded-control border border-gray-300 bg-white text-gray-700 hover:border-primary-500 hover:text-primary-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300"
        :aria-label="$t('connections.settings.title')"
        :title="$t('connections.settings.title')"
        @click="settingsOpen = true"
      >
        <WrenchScrewdriverIcon class="h-4 w-4" />
      </button>
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

    <!--
      The table scrolls INSIDE this region, sized by `useFillHeight` to the
      space left in the window — the same behaviour as the Outbounds list, so
      the toolbar stays put and only the rows move.
    -->
    <div ref="tableRegion" class="scroll-region rounded-surface border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800">
      <ConnectionsTable
        :rows="drawn"
        :settings="settings"
        :sort-key="sortKey"
        :ascending="ascending"
        :selected-id="selectedId"
        :now="now"
        :closing="closing"
        @sort="onSort"
        @select="onSelect"
        @close="closeOne"
      />
      <p v-if="rows.length === 0" class="px-4 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
        {{ pool.length === 0 ? $t(`connections.empty.${tab}`) : $t('connections.noMatch') }}
      </p>
      <p v-else-if="rows.length > drawn.length" class="border-t border-gray-200 px-4 py-2 text-center text-xs text-gray-500 dark:border-gray-700 dark:text-gray-400">
        {{ $t('connections.truncated', { shown: drawn.length, total: rows.length }) }}
      </p>
    </div>

    <ConnectionTableSettings
      v-model:visible="settingsOpen"
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

/*
 * Opens on hover AND on keyboard focus. Opaque on purpose: it floats over the
 * table, and the app's translucent glass would let the rows show through.
 */
.tips-panel {
  position: absolute;
  top: calc(100% + 6px);
  left: 0;
  z-index: 20;
  display: flex;
  width: max-content;
  max-width: min(22rem, calc(100vw - 3rem));
  flex-direction: column;
  gap: 6px;
  border: 1px solid var(--color-border-dark);
  border-radius: 10px;
  background: #ffffff;
  padding: 8px 10px;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.16);
  font-size: 12px;
  line-height: 1.45;
  color: var(--color-text-primary);
  opacity: 0;
  visibility: hidden;
  transform: translateY(-2px);
  pointer-events: none;
  transition:
    opacity 0.15s ease,
    transform 0.15s ease,
    visibility 0s linear 0.15s;
}
.tips:hover .tips-panel,
.tips:focus-within .tips-panel {
  opacity: 1;
  visibility: visible;
  transform: none;
  transition-delay: 0s;
}
.tips-line.is-warn {
  color: var(--color-warning);
  font-weight: 600;
}
.tips-line.is-error {
  color: var(--color-danger);
  font-weight: 600;
}
@media (prefers-color-scheme: dark) {
  .tips-panel {
    background: #131a23;
  }
}
@media (prefers-reduced-motion: reduce) {
  .tips-panel {
    transition: none;
  }
}
</style>
