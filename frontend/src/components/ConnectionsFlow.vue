<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ChevronDownIcon, ComputerDesktopIcon } from '@heroicons/vue/24/outline'
import ConnectionFlowRow from './ConnectionFlowRow.vue'
import { groupConnections, type ConnectionView } from '../utils/connectionFlow'
import { type TableSettings, type TrackedConnection } from '../utils/connectionsTable'
import { formatRate } from '../utils/flowOverlay'
import { useRowHighlight } from '../composables/useRowHighlight'

const props = defineProps<{
  rows: TrackedConnection[]
  mode: ConnectionView
  settings: TableSettings
  selectedId: string
  closing: ReadonlySet<string>
  now: number
}>()
const emit = defineEmits<{ select: [id: string]; close: [id: string] }>()
const root = ref<HTMLElement | null>(null)
useRowHighlight(root, '.flow-line')
const collapsed = ref(new Set<string>())
const groups = computed(() => props.mode === 'groups'
  ? groupConnections(props.rows)
  : [{ ip: '', rows: props.rows, down: 0, up: 0 }])
function toggle(ip: string) {
  if (collapsed.value.has(ip)) collapsed.value.delete(ip)
  else collapsed.value.add(ip)
}
watch(() => props.rows, rows => {
  const present = new Set(rows.map(row => row.source_ip))
  for (const ip of collapsed.value) if (!present.has(ip)) collapsed.value.delete(ip)
})
</script>

<template>
  <div ref="root" class="connections-flow" :class="{ 'flow-compact': mode === 'compact', 'flow-dense': settings.dense }">
    <section v-for="group in groups" :key="group.ip" class="flow-group">
      <button v-if="mode === 'groups'" type="button" class="flow-group-heading" :aria-expanded="!collapsed.has(group.ip)" @click="toggle(group.ip)">
        <ComputerDesktopIcon class="h-4 w-4 shrink-0" />
        <strong>{{ settings.labels[group.ip] || group.ip || '–' }}</strong>
        <span v-if="settings.labels[group.ip]" class="group-ip">{{ group.ip }}</span>
        <span class="group-count">{{ $t('connections.flow.count', { n: group.rows.length }) }}</span>
        <span class="group-speed">↓ {{ formatRate(group.down) }}/s <span>↑ {{ formatRate(group.up) }}/s</span></span>
        <ChevronDownIcon class="h-4 w-4 shrink-0 group-chevron" :class="{ collapsed: collapsed.has(group.ip) }" />
      </button>
      <div v-show="mode === 'compact' || !collapsed.has(group.ip)">
        <ConnectionFlowRow v-for="row in group.rows" :key="row.id" :row="row" :label="settings.labels[row.source_ip] || ''"
          :selected="row.id === selectedId" :closing="closing.has(row.id)" :now="now" :full-chain="settings.fullChain"
          @select="emit('select', $event)" @close="emit('close', $event)" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.connections-flow { position: relative; isolation: isolate; }
.flow-group-heading { position: relative; display: flex; align-items: center; gap: 9px; width: 100%; min-height: 42px; padding: 9px 16px; text-align: left; background: var(--color-bg-secondary); border-bottom: 1px solid var(--color-border); font-size: 12px; }
.flow-group-heading:focus-visible { outline: 2px solid var(--color-primary); outline-offset: -2px; }
.flow-group-heading strong { font-weight: 600; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.group-ip, .group-count, .group-speed { color: var(--color-text-secondary); font-variant-numeric: tabular-nums; white-space: nowrap; }
.group-count { font-size: 11px; }
.group-speed { margin-left: auto; font-size: 11px; }
.group-speed span { margin-left: 8px; }
.group-chevron { transition: transform var(--motion-feedback) var(--motion-continuous); }
.group-chevron.collapsed { transform: rotate(-90deg); }
@media (max-width: 700px) { .flow-group-heading { padding-inline: 10px; flex-wrap: wrap; } .group-ip { display: none; } .group-speed { flex: 1; text-align: right; } .group-speed span { display: none; } }
@media (prefers-reduced-motion: reduce) { .group-chevron { transition: none; } }
</style>
