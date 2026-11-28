<script setup lang="ts">
import Table from './Table.vue'
import StatusIndicator from './StatusIndicator.vue'
/**
 * The connection rows. Markup only: what is shown, searched and sorted is
 * decided by utils/connectionsTable.ts and handed in.
 *
 * Rows select on MOUSEDOWN, not click. The table re-sorts once a second, and a
 * click needs press and release to land on the same element: when a re-sort
 * falls between the two, the row slides out from under the pointer, the
 * release lands on a different row, and the browser reports a click on neither
 * — the press is silently lost. Press alone cannot be lost that way.
 *
 * A clicked row opens its details as a second row DIRECTLY BENEATH IT — the
 * answer appears where the question was asked. The table is wider than the
 * viewport on most screens, so the detail's content is pinned to the visible
 * left edge (`.detail-pin`) rather than stretched across the scrolled width,
 * where half of it would sit off-screen.
 */
import { ref, watch } from 'vue'
import ConnectionDetailFold from './ConnectionDetailFold.vue'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import ConnectionDetails from './ConnectionDetails.vue'
import { displayName } from '../utils/proxyGroups'
import { formatBytes } from '../utils/formatBytes'
import { formatRate } from '../utils/flowOverlay'
import {
  NUMERIC_COLUMNS, durationOf, exitOf, formatDuration, hostOf, leafOf,
  type ColumnKey, type TableSettings, type TrackedConnection,
} from '../utils/connectionsTable'

const props = defineProps<{
  rows: TrackedConnection[]
  settings: TableSettings
  sortKey: ColumnKey
  ascending: boolean
  selectedId: string
  now: number
  /** Ids with a close request in flight. */
  closing: ReadonlySet<string>
}>()

const emit = defineEmits<{
  (e: 'sort', key: ColumnKey): void
  (e: 'select', id: string): void
  (e: 'close', id: string): void
}>()

// Keep only closing details alive until their actual transition finishes.
const leavingIds = ref(new Set<string>())
watch(() => props.selectedId, (next, previous) => {
  if (previous) leavingIds.value.add(previous)
  leavingIds.value.delete(next)
})
function collapsed(id: string) {
  if (id !== props.selectedId) leavingIds.value.delete(id)
}

/** Above this a download is worth the eye; the same 100 KB/s the preview used. */
const HOT_RATE = 100 * 1024

const rate = (value: number) => (value > 0 ? `${formatRate(value)}/s` : '–')
const clock = (ms: number) => (ms ? new Date(ms).toLocaleTimeString(undefined, { hour12: false }) : '–')
</script>

<template>
  <Table :scroll="false" :column-storage-key="`connections:${settings.columns.join(',')}`" :table-class="settings.dense ? 'dense' : ''">
    <template #head>
        <th class="w-px">{{ $t('connections.columns.close') }}</th>
        <th v-for="key in settings.columns" :key="key" :class="{ 'text-right': NUMERIC_COLUMNS.has(key) }" :aria-sort="sortKey === key ? (ascending ? 'ascending' : 'descending') : undefined">
          <button type="button" class="font-semibold hover:text-primary-700 dark:hover:text-primary-300" @click="emit('sort', key)">
            {{ $t(`connections.columns.${key}`) }}<span v-if="sortKey === key" class="text-primary-600 dark:text-primary-400" aria-hidden="true">{{ ascending ? ' ↑' : ' ↓' }}</span>
          </button>
        </th>
    </template>

      <template v-for="row in rows" :key="row.id">
      <tr
        class="cursor-pointer"
        :class="{ selected: row.id === selectedId, gone: row.closed }"
        :aria-expanded="row.id === selectedId"
        tabindex="0"
        @keydown.enter.self.prevent="emit('select', row.id)"
        @keydown.space.self.prevent="emit('select', row.id)"
        @mousedown.left="emit('select', row.id)"
      >
        <td class="align-middle">
          <StatusIndicator v-if="row.closed" variant="secondary">
            {{ $t('connections.closedBadge') }}
          </StatusIndicator>
          <button
            v-else
            type="button"
            class="inline-flex h-7 w-7 items-center justify-center rounded-pill border border-gray-300 text-gray-500 hover:border-red-500 hover:text-red-600 disabled:cursor-wait disabled:opacity-40 dark:border-gray-600"
            :disabled="closing.has(row.id)"
            :aria-label="$t('connections.closeOne', { host: hostOf(row) })"
            @mousedown.stop
            @click.stop="emit('close', row.id)"
          >
            <!-- An icon, not the "×" character: a glyph sits on the text
                 baseline with its own side bearings, so it is never quite in
                 the middle of a round button. An SVG is centred by its box. -->
            <XMarkIcon class="h-3 w-3" aria-hidden="true" />
          </button>
        </td>
        <template v-for="key in settings.columns" :key="key">
          <td v-if="key === 'source'" class="tabular-nums" :title="row.source_ip">{{ settings.labels[row.source_ip] || row.source_ip }}</td>
          <td v-else-if="key === 'sourcePort'" class="text-right tabular-nums">{{ row.source_port }}</td>
          <td v-else-if="key === 'host'" class="max-w-[240px] truncate font-medium" :title="`${hostOf(row)}:${row.destination_port}`">
            {{ hostOf(row) }}<span class="font-normal text-gray-500 dark:text-gray-400">:{{ row.destination_port }}</span>
          </td>
          <td v-else-if="key === 'destinationIp'" class="tabular-nums">{{ row.destination_ip || '–' }}</td>
          <td v-else-if="key === 'network'">
            <span class="rounded bg-gray-100 px-1.5 py-px text-[11px] text-gray-600 dark:bg-gray-700 dark:text-gray-300">{{ row.network }}</span>
          </td>
          <td v-else-if="key === 'inbound'">{{ row.inbound }}</td>
          <td v-else-if="key === 'rule'" class="max-w-[300px] truncate" :title="row.rule">{{ row.rule }}</td>
          <td v-else-if="key === 'chain'">
            <template v-if="settings.fullChain">
              <template v-for="(hop, index) in row.chains" :key="index">
                <span v-if="index" class="text-gray-400"> → </span>
                <span :class="index === 0 ? 'font-semibold' : 'text-gray-500 dark:text-gray-400'" :title="hop">{{ displayName(hop) }}</span>
              </template>
            </template>
            <template v-else>
              <span class="font-semibold" :title="exitOf(row)">{{ displayName(exitOf(row)) || '–' }}</span>
              <span v-if="leafOf(row)" class="ml-1.5 text-gray-500 dark:text-gray-400" :title="leafOf(row)">{{ $t('connections.via', { node: displayName(leafOf(row)) }) }}</span>
            </template>
          </td>
          <td v-else-if="key === 'downRate'" class="text-right tabular-nums" :class="{ 'font-semibold text-primary-700 dark:text-primary-300': row.down_rate >= HOT_RATE }">{{ rate(row.down_rate) }}</td>
          <td v-else-if="key === 'upRate'" class="text-right tabular-nums">{{ rate(row.up_rate) }}</td>
          <td v-else-if="key === 'download'" class="text-right tabular-nums">{{ formatBytes(row.download) }}</td>
          <td v-else-if="key === 'upload'" class="text-right tabular-nums">{{ formatBytes(row.upload) }}</td>
          <td v-else-if="key === 'duration'" class="text-right tabular-nums">{{ formatDuration(durationOf(row, now)) }}</td>
          <td v-else-if="key === 'start'" class="text-right tabular-nums">{{ clock(row.start) }}</td>
        </template>
      </tr>
      <tr v-if="row.id === selectedId || leavingIds.has(row.id)" class="detail-row" :class="{ 'is-leaving': row.id !== selectedId }">
        <td :colspan="settings.columns.length + 1">
          <ConnectionDetailFold :open="row.id === selectedId" @collapsed="collapsed(row.id)">
          <div class="detail-pin">
            <ConnectionDetails
              :connection="row"
              :label="settings.labels[row.source_ip] ?? ''"
              :now="now"
              :closing="closing.has(row.id)"
              @dismiss="emit('select', row.id)"
              @close="emit('close', row.id)"
            />
          </div>
          </ConnectionDetailFold>
        </td>
      </tr>
      </template>

  </Table>
</template>

<style scoped>
th,
td {
  white-space: nowrap;
  border-bottom: 1px solid var(--color-border);
  padding: 7px 10px;
  text-align: left;
}
th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--connections-head-bg, #f4f7fb);
  font-size: 11px;
  color: var(--color-text-secondary);
}
th.text-right,
td.text-right {
  text-align: right;
}
table.dense th,
table.dense td {
  padding: 4px 8px;
  font-size: 12px;
}
tbody tr.detail-row,
tbody tr.detail-row:hover {
  cursor: default;
  background: color-mix(in srgb, var(--color-primary) 5%, transparent);
}
tbody tr.detail-row > td {
  /* The padding lives on the panel inside, so a folded row is truly 0 tall. */
  padding: 0;
  /* Reads as the lower half of the selected row, not as a row of its own. */
  box-shadow: inset 3px 0 0 var(--color-primary);
}
.detail-pin {
  position: sticky;
  left: 12px;
  width: min(100%, calc(100vw - 6rem));
  max-width: 980px;
  padding: 10px 12px 12px;
}
tbody tr:hover,
tbody tr.selected {
  background: color-mix(in srgb, var(--color-primary) 9%, transparent);
}
tbody tr.gone td {
  color: var(--color-text-secondary);
}
tbody tr:last-child td {
  border-bottom: 0;
}
@media (prefers-color-scheme: dark) {
  th {
    background: var(--connections-head-bg, #1b2430);
  }
}
</style>
