<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ChartBarIcon, ArrowsRightLeftIcon, GlobeAltIcon, Cog6ToothIcon, PlusIcon, ArrowPathIcon } from '@heroicons/vue/24/outline'
import { Button, Card, Input, Table, StatusIndicator, IconButton } from '../components'
import Select from '../volt/Select.vue'
import PopConfirm from '../components/PopConfirm.vue'
import Popover from '../volt/Popover.vue'
import { getLabPalette, labPalettes, paletteTokens } from '../lab/palettes'

const route = useRoute()
const palette = ref(getLabPalette(route.query.palette))
const page = ref('Overview')
const query = ref('')
const protocol = ref('VLESS')
const scope = ref('All nodes')
const tag = ref('Tokyo edge')
const feedback = ref('')
const editing = ref(false)
const originalName = ref('')
const editor = ref<InstanceType<typeof Popover> | null>(null)
const loading = ref(false)
const selected = ref<string[]>([])
const rows = ref([
  { name: 'Tokyo edge', type: 'VLESS', endpoint: 'tokyo.example.com', delay: 46, ready: true },
  { name: 'Singapore relay', type: 'Trojan', endpoint: 'sg.example.com', delay: 68, ready: true },
  { name: 'Home gateway', type: 'SOCKS', endpoint: '192.0.2.10', delay: null, ready: false },
])
const visibleRows = computed(() => rows.value.filter(row => row.name.toLowerCase().includes(query.value.toLowerCase()) && (scope.value !== 'Ready only' || row.ready)))
const navigation = [
  { label: 'Overview', icon: ChartBarIcon }, { label: 'Outbounds', icon: ArrowsRightLeftIcon },
  { label: 'DNS', icon: GlobeAltIcon }, { label: 'Settings', icon: Cog6ToothIcon },
]
let timer: ReturnType<typeof setTimeout> | undefined
const previousStyles = new Map<string, string>()
const previousScheme = document.documentElement.style.colorScheme
const previousTitle = document.title
const previousMarker = document.documentElement.getAttribute('data-palette-preview')
document.documentElement.setAttribute('data-palette-preview', '')
function applyPalette() {
  for (const [name, value] of Object.entries(paletteTokens(palette.value))) {
    if (!previousStyles.has(name)) previousStyles.set(name, document.documentElement.style.getPropertyValue(name))
    document.documentElement.style.setProperty(name, value)
  }
  document.documentElement.style.colorScheme = palette.value.mode
  document.title = `${palette.value.name} — palette preview`
}
watch(palette, applyPalette, { immediate: true })
function onMessage(event: MessageEvent) {
  if (event.source !== window.parent || event.origin !== window.location.origin) return
  if (event.data?.type === 'sbe:palette' && labPalettes.some(option => option.id === event.data.id)) palette.value = getLabPalette(event.data.id)
}
function refresh() {
  loading.value = true
  clearTimeout(timer)
  timer = setTimeout(() => { loading.value = false; feedback.value = 'Sample nodes refreshed' }, 1000)
}
function save() {
  if (!tag.value.trim()) return
  if (editing.value) rows.value = rows.value.map(row => row.name === originalName.value ? { ...row, name: tag.value.trim(), type: protocol.value } : row)
  else rows.value = [...rows.value, { name: tag.value.trim(), type: protocol.value, endpoint: 'new.example.com', delay: 52, ready: true }]
  feedback.value = editing.value ? 'Sample changes saved' : 'Sample node added'
  editor.value?.hide()
}
onMounted(() => {
  window.addEventListener('message', onMessage)
  window.parent.postMessage({ type: 'sbe:palette-ready' }, window.location.origin)
})
onBeforeUnmount(() => {
  clearTimeout(timer)
  window.removeEventListener('message', onMessage)
  for (const [name, value] of previousStyles) {
    if (value) document.documentElement.style.setProperty(name, value)
    else document.documentElement.style.removeProperty(name)
  }
  document.documentElement.style.colorScheme = previousScheme
  document.title = previousTitle
  if (previousMarker === null) document.documentElement.removeAttribute('data-palette-preview')
  else document.documentElement.setAttribute('data-palette-preview', previousMarker)
})
</script>

<template>
  <div class="palette-workspace liquid-app" :data-palette="palette.id">
    <aside class="preview-navigation">
      <div class="preview-brand"><img src="/logo.svg" alt="" width="32" height="32" /><span>Sing Box Easy<small>Network workspace</small></span></div>
      <p class="preview-group-label">Workspace</p>
      <nav aria-label="Sample workspace">
        <button v-for="item in navigation" :key="item.label" type="button" :aria-current="page === item.label ? 'page' : undefined" @click="page = item.label; feedback = ''">
          <component :is="item.icon" class="h-4 w-4 shrink-0" aria-hidden="true" /><span>{{ item.label }}</span>
        </button>
      </nav>
      <div class="preview-nav-footer"><span class="preview-online-dot" /> sing-box <span class="ml-auto">Running</span></div>
    </aside>
    <main class="preview-main">
      <header class="preview-toolbar">
        <div><p class="preview-eyebrow">Network / {{ page }}</p><h1>{{ page }}</h1></div>
        <Select v-model="scope" aria-label="Node scope" :options="['All nodes', 'Ready only']" class="w-32" />
      </header>
      <div class="preview-metrics">
        <Card><span class="metric-label">Active connections</span><p class="metric-value">128 <span class="metric-trend">+12</span></p><p class="metric-caption">Across 3 sample nodes</p></Card>
        <Card><span class="metric-label">Average latency</span><p class="metric-value">46 <span class="metric-unit">ms</span></p><p class="metric-caption">Stable over the last hour</p></Card>
        <Card><span class="metric-label">Traffic today</span><p class="metric-value">8.4 <span class="metric-unit">GB</span></p><p class="metric-caption">6.2 GB down · 2.2 GB up</p></Card>
      </div>
      <Card class="traffic-card">
        <div class="traffic-heading"><h2>Traffic</h2><span><i class="chart-key" />Download <i class="chart-key secondary" />Upload</span><strong>38 MB/s</strong></div>
        <svg viewBox="0 0 800 90" preserveAspectRatio="none" class="traffic-chart" role="img" aria-label="Sample download and upload traffic over one hour">
          <path d="M0 20H800M0 45H800M0 70H800" stroke="var(--color-border)" stroke-width=".7" />
          <path d="M0 62 30 59 60 64 90 45 120 49 150 37 180 44 210 31 240 36 270 20 300 30 330 25 360 47 390 41 420 45 450 29 480 34 510 15 540 23 570 18 600 35 630 29 660 14 690 20 720 9 750 19 800 13" fill="none" stroke="var(--color-primary)" stroke-width="2.5" stroke-linejoin="round" />
          <path d="M0 78 40 74 80 77 120 65 160 71 200 59 240 65 280 51 320 60 360 67 400 59 440 64 480 51 520 56 560 47 600 61 640 48 680 55 720 43 760 51 800 45" fill="none" stroke="var(--palette-warm)" stroke-width="2" stroke-linejoin="round" />
        </svg>
        <div class="chart-times"><span>09:00</span><span>09:30</span><span>10:00</span></div>
      </Card>
      <Card class="nodes-card" padding="none">
        <div class="nodes-heading"><div><h2>{{ page === 'DNS' ? 'Resolver connections' : 'Outbounds' }}</h2><p>Routes available to this workspace</p></div>
          <Button size="sm" @click="editing = false; tag = ''; editor?.toggle($event)"><PlusIcon class="h-4 w-4" aria-hidden="true" /> Add node</Button>
        </div>
        <div class="nodes-filter"><Input v-model="query" label="Search nodes" placeholder="Name or region" /><Button variant="ghost" size="sm" :loading="loading" @click="refresh"><ArrowPathIcon class="h-4 w-4" aria-hidden="true" /><span class="sr-only">Refresh sample nodes</span></Button></div>
        <Table max-height="none" :loading="loading" :empty="visibleRows.length === 0" table-class="min-w-[38rem]">
          <template #head><th class="col-select"><label class="table-checkbox"><input type="checkbox" aria-label="Select all sample nodes" :checked="selected.length === rows.length && rows.length > 0" :indeterminate="selected.length > 0 && selected.length < rows.length" @change="selected = ($event.target as HTMLInputElement).checked ? rows.map(row => row.name) : []" /></label></th><th>Name</th><th>Status</th><th>Latency</th><th class="col-actions">Actions</th></template>
          <template #empty><p>No matching sample nodes</p><Button variant="ghost" size="sm" @click="query = ''; scope = 'All nodes'">Clear filters</Button></template>
          <tr v-for="row in visibleRows" :key="row.name" :data-selected="selected.includes(row.name)">
            <td class="col-select"><label class="table-checkbox"><input v-model="selected" type="checkbox" :value="row.name" :aria-label="`Select ${row.name}`" /></label></td>
            <td><p class="node-name">{{ row.name }}</p><span class="node-detail">{{ row.type }} · {{ row.endpoint }}</span></td>
            <td><StatusIndicator :variant="row.ready ? 'success' : 'warning'">{{ row.ready ? 'Ready' : 'Pending' }}</StatusIndicator></td>
            <td class="font-mono">{{ row.delay === null ? '—' : `${row.delay} ms` }}</td>
            <td class="col-actions"><div class="flex items-center justify-end gap-1"><IconButton action="edit" :label="`Edit ${row.name}`" @click="editing = true; originalName = row.name; tag = row.name; protocol = row.type; editor?.toggle($event)" /><PopConfirm message="Remove this sample node?" :target="row.name" confirm-label="Remove sample" @confirm="rows = rows.filter(item => item !== row); feedback = 'Sample node removed'" /></div></td>
          </tr>
        </Table>
        <div class="nodes-footer"><StatusIndicator variant="success">Configuration valid</StatusIndicator><span>Last checked just now</span></div>
      </Card>
      <div class="preview-bottom"><p role="status">{{ feedback || 'Sample workspace · Changes stay in this preview' }}</p><span>{{ palette.name }} / {{ palette.mode }}</span></div>
    </main>
    <Popover ref="editor" :aria-label="editing ? 'Edit sample node' : 'Add sample node'">
      <form class="space-y-4" @submit.prevent="save"><h2 class="font-semibold">{{ editing ? 'Edit sample node' : 'Add sample node' }}</h2><Input v-model="tag" label="Node name" required /><div><label id="palette-protocol-label" for="palette-protocol" class="mb-1 block text-sm font-medium">Protocol</label><Select v-model="protocol" input-id="palette-protocol" aria-labelledby="palette-protocol-label" :options="['VLESS', 'Trojan', 'SOCKS', 'Shadowsocks']" filter filter-placeholder="Search protocols" class="w-full" /></div><div class="flex justify-end gap-2"><Button variant="ghost" @click="editor?.hide()">Cancel</Button><Button type="submit">Save sample</Button></div></form>
    </Popover>
  </div>
</template>

<style>
/* This isolated preview maps shared gray text classes to explicit palette roles,
   so light/dark review never depends on the operating system or browser emulation.
   Important matches the existing legacy.css compatibility overrides. */
[data-palette-preview] :is(.text-gray-900, .text-gray-800, .text-gray-700, [class~="dark:text-gray-100"], [class~="dark:text-gray-200"], [class~="dark:text-gray-300"]) { color: var(--color-text-primary) !important; }
[data-palette-preview] :is(.text-gray-600, .text-gray-500, .text-gray-400, [class~="dark:text-gray-400"], [class~="dark:text-gray-500"]) { color: var(--color-text-secondary) !important; }
[data-palette-preview] .volt-select-option-focus { background: color-mix(in srgb, var(--color-primary) 10%, var(--color-bg-secondary)); }
[data-palette-preview] .volt-select-option-selected { background: color-mix(in srgb, var(--color-primary) 16%, var(--color-bg-secondary)); color: var(--color-primary); }
.palette-workspace { display: flex; min-height: 100dvh; font-size: 13px; }
.palette-workspace .bg-primary, [data-palette-preview] [data-pc-name="popover"] .bg-primary { color: var(--palette-on-primary); }
.palette-workspace input[type="checkbox"] { accent-color: var(--color-primary); }
.palette-workspace .status-indicator[data-tone="success"] { color: var(--color-success); }
.palette-workspace .status-indicator[data-tone="warning"] { color: var(--color-warning); }
.palette-workspace .status-indicator[data-tone="danger"] { color: var(--color-danger); }
.preview-navigation { width: 176px; flex-shrink: 0; display: flex; flex-direction: column; padding: 24px 12px 16px; background: var(--palette-nav); color: var(--palette-nav-ink); }
.preview-brand { display: flex; align-items: center; gap: 9px; padding: 0 4px; font-size: 12px; font-weight: 600; }
.preview-brand img { flex-shrink: 0; }
.preview-brand small { display: block; margin-top: 3px; font-size: 10px; font-weight: 400; color: var(--palette-nav-muted); }
.preview-group-label { margin: 32px 12px 8px; font-size: 10px; color: var(--palette-nav-muted); }
.preview-navigation nav { display: flex; flex-direction: column; gap: 4px; }
.preview-navigation nav button { display: flex; align-items: center; gap: 10px; min-height: 36px; padding: 8px 12px; border-radius: 6px; text-align: left; color: var(--palette-nav-muted); cursor: pointer; }
.preview-navigation nav button:hover { background: var(--palette-nav-active); color: var(--palette-nav-ink); }
.preview-navigation nav button[aria-current="page"] { background: var(--palette-nav-active); color: var(--palette-nav-ink); box-shadow: inset 2px 0 #8fbaff; }
.preview-navigation nav button:focus-visible { outline: 2px solid #8fbaff; outline-offset: 2px; }
.preview-nav-footer { display: flex; align-items: center; gap: 6px; margin-top: auto; padding: 24px 4px 0; font-size: 10px; color: var(--palette-nav-muted); }
.preview-online-dot { width: 6px; height: 6px; border-radius: 50%; background: #70cbb4; }
.preview-main { flex: 1; min-width: 0; padding: 24px; display: flex; flex-direction: column; gap: 16px; }
.preview-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.preview-eyebrow { color: var(--color-text-secondary); font-size: 10px; margin-bottom: 4px; }
.preview-toolbar h1 { font-size: 22px; font-weight: 600; }
.preview-metrics { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 12px; }
.metric-label { color: var(--color-text-secondary); font-size: 11px; }
.metric-value { font-size: 27px; font-weight: 600; line-height: 1.3; margin-top: 8px; font-variant-numeric: tabular-nums; }
.metric-unit { font-size: 14px; font-weight: 400; color: var(--color-text-secondary); }
.metric-trend { font-size: 11px; font-weight: 500; color: var(--color-success); }
.metric-caption { margin-top: 4px; font-size: 10px; color: var(--color-text-secondary); }
.traffic-heading { display: flex; align-items: center; gap: 16px; font-size: 11px; }
.traffic-heading h2, .nodes-heading h2 { font-size: 13px; font-weight: 600; }
.traffic-heading > span { display: flex; align-items: center; gap: 5px; margin-left: auto; color: var(--color-text-secondary); }
.traffic-heading strong { font-variant-numeric: tabular-nums; font-weight: 500; }
.chart-key { display: inline-block; width: 6px; height: 6px; border-radius: 50%; background: var(--color-primary); }
.chart-key.secondary { margin-left: 8px; background: var(--palette-warm); }
.traffic-chart { display: block; width: 100%; height: 90px; margin-top: 12px; }
.chart-times { display: flex; justify-content: space-between; margin-top: 4px; font-size: 10px; color: var(--color-text-secondary); }
.nodes-heading { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; padding: 16px 16px 12px; }
.nodes-heading p { margin-top: 3px; font-size: 11px; color: var(--color-text-secondary); }
.nodes-filter { display: flex; align-items: end; gap: 8px; padding: 0 16px 12px; }
.nodes-filter > div { max-width: 240px; }
.nodes-card .data-table th { color: var(--color-text-secondary); }
.nodes-card .data-table td { color: var(--color-text-primary); }
.node-name { font-size: 12px; font-weight: 500; }
.node-detail { font-size: 10px; color: var(--color-text-secondary); }
.nodes-footer { display: flex; justify-content: space-between; align-items: center; gap: 8px; padding: 12px 16px; border-top: 1px solid var(--color-border); }
.nodes-footer > span:last-child { font-size: 10px; color: var(--color-text-secondary); }
.preview-bottom { display: flex; flex-wrap: wrap; justify-content: space-between; gap: 8px; font-size: 10px; color: var(--color-text-secondary); }
@media (max-width: 680px) {
  .palette-workspace { flex-direction: column; }
  .preview-navigation { width: auto; padding: 12px; gap: 12px; }
  .preview-brand small, .preview-group-label, .preview-nav-footer { display: none; }
  .preview-navigation nav { flex-direction: row; overflow-x: auto; }
  .preview-navigation nav button { padding: 8px; min-height: 32px; gap: 6px; font-size: 11px; flex-shrink: 0; }
  .preview-main { padding: 16px 12px; gap: 12px; }
  .preview-metrics { gap: 8px; }
  .preview-metrics > div { padding: 12px 8px; }
  .metric-label { font-size: 10px; }
  .metric-value { font-size: 22px; }
  .metric-caption { display: none; }
  .traffic-heading { flex-wrap: wrap; gap: 8px; }
}
</style>
