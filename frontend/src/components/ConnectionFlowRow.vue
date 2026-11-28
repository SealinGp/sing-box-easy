<script setup lang="ts">
import { ref, watch } from 'vue'
import { ChevronRightIcon } from '@heroicons/vue/24/outline'
import ConnectionDetailFold from './ConnectionDetailFold.vue'
import ConnectionDetails from './ConnectionDetails.vue'
import { endpoint } from '../utils/connectionFlow'
import { hostOf, exitOf, type TrackedConnection } from '../utils/connectionsTable'
import { displayName } from '../utils/proxyGroups'
import { formatRate } from '../utils/flowOverlay'

const props = defineProps<{
  row: TrackedConnection
  label: string
  selected: boolean
  closing: boolean
  now: number
  fullChain: boolean
}>()
const emit = defineEmits<{ select: [id: string]; close: [id: string] }>()
const leaving = ref(false)
const trigger = ref<HTMLButtonElement | null>(null)
watch(() => props.selected, (next, previous) => { leaving.value = previous && !next })
function dismiss() {
  emit('select', props.row.id)
  trigger.value?.focus({ preventScroll: true })
}
</script>

<template>
  <article class="connection-flow-row" :class="{ selected, closed: row.closed }">
    <button ref="trigger" type="button" class="flow-line" :aria-expanded="selected" @click="emit('select', row.id)">
      <span class="flow-source" :title="endpoint(row.source_ip, row.source_port)">
        <strong>{{ label || row.source_ip || '–' }}</strong>
        <small>{{ label ? endpoint(row.source_ip, row.source_port) : `:${row.source_port}` }}</small>
      </span>
      <span class="flow-bridge" aria-hidden="true"><span>{{ row.network.toUpperCase() }}</span><i /></span>
      <span class="flow-target">
        <strong :title="endpoint(hostOf(row), row.destination_port)">{{ endpoint(hostOf(row), row.destination_port) }}</strong>
        <small :title="row.chains.join(' → ')">{{ fullChain ? row.chains.map(displayName).join(' → ') : displayName(exitOf(row)) || '–' }}</small>
      </span>
      <span class="flow-rates">
        <span v-if="row.closed" class="flow-closed">{{ $t('connections.closedBadge') }}</span>
        <template v-else>
          <strong :aria-label="`${$t('connections.columns.downRate')} ${formatRate(row.down_rate)}/s`">↓ {{ formatRate(row.down_rate) }}/s</strong>
          <small :aria-label="`${$t('connections.columns.upRate')} ${formatRate(row.up_rate)}/s`">↑ {{ formatRate(row.up_rate) }}/s</small>
        </template>
      </span>
      <ChevronRightIcon class="flow-chevron" :class="{ expanded: selected }" />
    </button>
    <ConnectionDetailFold v-if="selected || leaving" :open="selected" @collapsed="leaving = false">
      <div class="flow-detail">
        <ConnectionDetails :connection="row" :label="label" :now="now" :closing="closing" @dismiss="dismiss" @close="emit('close', row.id)" />
      </div>
    </ConnectionDetailFold>
  </article>
</template>

<style scoped>
.connection-flow-row { border-bottom: 1px solid var(--color-border); }
.flow-line { position: relative; display: grid; grid-template-columns: minmax(100px, 1fr) 76px minmax(140px, 1.6fr) 100px 16px; align-items: center; gap: 14px; width: 100%; min-height: 62px; padding: 9px 16px; text-align: left; font-size: 13px; }
.flow-line:focus-visible { outline: 2px solid var(--color-primary); outline-offset: -2px; border-radius: 6px; }
.selected > .flow-line { background: color-mix(in srgb, var(--color-primary) 8%, transparent); }
.flow-source, .flow-target, .flow-rates { display: flex; min-width: 0; flex-direction: column; gap: 2px; }
strong { font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
small { font-size: 11px; color: var(--color-text-secondary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.flow-source, .flow-rates { font-variant-numeric: tabular-nums; }
.flow-rates { text-align: right; }
.flow-bridge { display: flex; flex-direction: column; gap: 5px; align-items: center; font-size: 10px; color: var(--color-text-secondary); }
.flow-bridge i { position: relative; display: block; height: 1px; width: 100%; background: color-mix(in srgb, var(--color-primary) 40%, var(--color-border)); }
.flow-bridge i::after { content: ''; position: absolute; right: 0; top: -3px; width: 6px; height: 6px; border-top: 1px solid currentColor; border-right: 1px solid currentColor; transform: rotate(45deg); }
.flow-chevron { width: 15px; height: 15px; color: var(--color-text-secondary); transition: transform var(--motion-feedback) var(--motion-continuous); }
.flow-chevron.expanded { transform: rotate(90deg); }
.flow-detail { padding: 14px 16px; border-left: 3px solid var(--color-primary); background: color-mix(in srgb, var(--color-primary) 4%, transparent); }
.closed .flow-line { color: var(--color-text-secondary); }
:global(.flow-compact .flow-line), :global(.flow-dense .flow-line) { min-height: 48px; padding-block: 6px; }
:global(.flow-compact .flow-source) { flex-direction: row; align-items: baseline; gap: 0; }
@media (max-width: 700px) {
  .flow-line { grid-template-columns: minmax(0, 1fr) 32px minmax(0, 1.3fr) 14px; gap: 8px; padding: 10px; }
  .flow-rates { grid-column: 1 / 4; grid-row: 2; flex-direction: row; gap: 14px; text-align: left; }
  .flow-chevron { grid-column: 4; grid-row: 1; }
  .flow-source, :global(.flow-compact .flow-source) { flex-direction: column; gap: 2px; align-items: stretch; }
  .flow-detail { padding: 12px; }
}
@media (prefers-reduced-motion: reduce) { .flow-chevron { transition: none; } }
</style>
