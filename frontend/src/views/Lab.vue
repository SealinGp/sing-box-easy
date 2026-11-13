<script setup lang="ts">
import PaletteStudio from '../components/PaletteStudio.vue'
import IconButton from '../components/IconButton.vue'
import StatusIndicator from '../components/StatusIndicator.vue'
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useToast } from 'primevue/usetoast'
import {
  ArrowPathIcon, ArrowRightIcon, BeakerIcon, CheckIcon,
  CubeTransparentIcon, PlusIcon, SwatchIcon,
} from '@heroicons/vue/24/outline'
import { Alert, Badge, Button, Card, CopyIcon, Input, SegmentedProgress, Skeleton, Table, Textarea } from '../components'
import type { TableColumn } from '../components/Table.vue'
import PopConfirm from '../components/PopConfirm.vue'
import Select from '../volt/Select.vue'
import Chips from '../volt/Chips.vue'
import MultiSelect from '../volt/MultiSelect.vue'
import Popover from '../volt/Popover.vue'
import Toast from '../volt/Toast.vue'
import SearchPicker from '../components/SearchPicker.vue'
import { provideInterfaceMotion } from '../composables/useInterfaceMotion'
import { useNavIndicator } from '../composables/useNavIndicator'

provideInterfaceMotion()

// A living reference: import production primitives and read production tokens.
// No API services, persistent storage, or second set of component styles.
// Design direction informed by unslop, Damir Sokolovsky (@heydamir), CC BY 4.0:
// https://github.com/heydamir/unslop
interface SampleOutbound {
  tag: string
  server: string
  port: number
  protocol: string
  status: 'Ready' | 'Pending'
}
const fixtures: SampleOutbound[] = [
  { tag: 'Tokyo edge', server: 'tokyo.example.com', port: 443, protocol: 'VLESS', status: 'Ready' },
  { tag: 'Singapore relay', server: 'sg.example.com', port: 8443, protocol: 'Trojan', status: 'Ready' },
  { tag: 'Home gateway', server: '192.0.2.10', port: 1080, protocol: 'SOCKS', status: 'Pending' },
]
const rows = ref(fixtures.map(row => ({ ...row })))
const query = ref('')
const tableState = ref('data')
const tableStates = [
  { label: 'Populated', value: 'data' },
  { label: 'Empty', value: 'empty' },
  { label: 'Loading', value: 'loading' },
]
const columns: TableColumn[] = [
  { key: 'tag', label: 'Outbound' },
  { key: 'protocol', label: 'Protocol' },
  { key: 'status', label: 'Status' },
  { key: 'actions', label: 'Actions', align: 'right' },
]
const filteredRows = computed(() => rows.value.filter(row =>
  `${row.tag} ${row.server} ${row.protocol}`.toLowerCase().includes(query.value.toLowerCase()),
))
const protocols = ['VLESS', 'Trojan', 'SOCKS', 'Shadowsocks', 'Hysteria2', 'TUIC']
const protocol = ref('VLESS')
const tag = ref('Tokyo edge')
const server = ref('tokyo.example.com')
const port = ref<string | number>(443)
const domains = ref(['example.com', 'example.org'])
const networks = ref(['TCP'])
const notes = ref('Primary route for this workspace')
const submitted = ref(false)
const enabled = ref(true)
const listenerPanel = ref<InstanceType<typeof Popover> | null>(null)
const listenerOpen = ref(false)
const listenerName = ref('Local listener')
const listenerProtocol = ref('Mixed')
const saveFeedback = ref('')
const overlayFeedback = ref('')
const actionFeedback = ref({ summary: 'Choose a button to preview its feedback', severity: 'info' })
const alertVisible = ref(true)
const sampleLoading = ref(false)
let sampleRequest: ReturnType<typeof setTimeout> | undefined
function simulateRequest() {
  sampleLoading.value = true
  sampleRequest = setTimeout(() => { sampleLoading.value = false }, 1600)
}
const toast = useToast()
const tagError = computed(() => submitted.value && !tag.value.trim() ? 'Enter an outbound name' : '')
const serverError = computed(() => submitted.value && !server.value.trim() ? 'Enter a server address' : '')
const portError = computed(() => submitted.value && (!Number.isInteger(Number(port.value)) || Number(port.value) < 1 || Number(port.value) > 65535)
  ? 'Use a port between 1 and 65535' : '')

function notify(summary: string, severity: 'success' | 'info' | 'warn' | 'error' = 'success') {
  actionFeedback.value = { summary, severity }
}
function saveSample() {
  submitted.value = true
  if (tagError.value || serverError.value || portError.value) return
  const sample: SampleOutbound = {
    tag: tag.value.trim(), server: server.value.trim(), port: Number(port.value),
    protocol: protocol.value, status: enabled.value ? 'Ready' : 'Pending',
  }
  const index = rows.value.findIndex(row => row.tag === sample.tag)
  if (index >= 0) rows.value[index] = sample
  else rows.value.push(sample)
  query.value = ''
  tableState.value = 'data'
  saveFeedback.value = 'Sample saved'
}
function editSample(row: SampleOutbound) {
  tag.value = row.tag
  server.value = row.server
  port.value = row.port
  protocol.value = row.protocol
  enabled.value = row.status === 'Ready'
  submitted.value = false
  document.getElementById('lab-outbound-form')?.scrollIntoView({ block: 'center' })
}
function reset() {
  clearTimeout(sampleRequest)
  sampleLoading.value = false
  rows.value = fixtures.map(row => ({ ...row }))
  query.value = ''
  tableState.value = 'data'
  protocol.value = 'VLESS'
  tag.value = 'Tokyo edge'
  server.value = 'tokyo.example.com'
  port.value = 443
  domains.value = ['example.com', 'example.org']
  networks.value = ['TCP']
  notes.value = 'Primary route for this workspace'
  enabled.value = true
  submitted.value = false
  listenerPanel.value?.hide()
  listenerName.value = 'Local listener'
  listenerProtocol.value = 'Mixed'
  saveFeedback.value = ''
  overlayFeedback.value = ''
  actionFeedback.value = { summary: 'Choose a button to preview its feedback', severity: 'info' }
  alertVisible.value = true
  activeView.value = 'Outbounds'
  detailsOpen.value = false
  pickedNode.value = 'No node selected'
  toast.removeAllGroups()
}

const sections = [
  { id: 'palettes', label: 'Color directions', icon: SwatchIcon },
  { id: 'workspace', label: 'Working example', icon: CubeTransparentIcon },
  { id: 'components', label: 'Components', icon: PlusIcon },
  { id: 'motion', label: 'Motion & navigation', icon: ArrowRightIcon },
  { id: 'feedback', label: 'Feedback & overlays', icon: BeakerIcon },
  { id: 'foundations', label: 'Foundations', icon: SwatchIcon },
]
const viewTabs = ['Outbounds', 'DNS', 'Routes']
const activeView = ref('Outbounds')
const detailsOpen = ref(false)
const pickedNode = ref('No node selected')
const viewNav = ref<HTMLElement | null>(null)
const viewIndicator = ref<HTMLElement | null>(null)
const { measure: measureView } = useNavIndicator({
  scroller: viewNav, content: viewNav, indicator: viewIndicator, selector: '[aria-selected="true"]',
})
watch(activeView, measureView, { flush: 'post' })
function stepView(tab: string, direction: number) {
  activeView.value = viewTabs[(viewTabs.indexOf(tab) + viewTabs.length + direction) % viewTabs.length]!
  nextTick(() => document.getElementById(`lab-tab-${activeView.value}`)?.focus())
}
const colors = [
  { label: 'Primary', token: '--color-primary' },
  { label: 'Background', token: '--color-bg-primary' },
  { label: 'Primary text', token: '--color-text-primary' },
  { label: 'Secondary text', token: '--color-text-secondary' },
  { label: 'Border', token: '--color-border' },
  { label: 'Success', token: '--color-success' },
  { label: 'Warning', token: '--color-warning' },
  { label: 'Danger', token: '--color-danger' },
]
const metrics = [
  { label: 'Table radius', token: '--radius-table' },
  { label: 'Badge radius', token: '--radius-badge' },
  { label: 'Control radius', token: '--radius-control' },
  { label: 'Surface radius', token: '--radius-surface' },
  { label: 'Page gutter', token: '--space-gutter' },
  { label: 'Card padding', token: '--space-card' },
]
const tokenValues = ref<Record<string, string>>({})
const preview = ref<HTMLElement | null>(null)
const fontFamily = ref('')
const darkMode = ref(false)
let scheme: MediaQueryList | undefined
function readTheme() {
  if (!preview.value) return
  const style = getComputedStyle(preview.value)
  tokenValues.value = Object.fromEntries([...colors, ...metrics].map(({ token }) =>
    [token, style.getPropertyValue(token).trim()],
  ))
  fontFamily.value = getComputedStyle(document.body).fontFamily
  darkMode.value = scheme?.matches ?? false
}
onMounted(() => {
  scheme = window.matchMedia('(prefers-color-scheme: dark)')
  readTheme()
  scheme.addEventListener('change', readTheme)
})
onBeforeUnmount(() => {
  scheme?.removeEventListener('change', readTheme)
  clearTimeout(sampleRequest)
})
</script>

<template>
  <div ref="preview" class="lab-preview liquid-app min-h-screen text-sm text-text-primary">
    <a href="#lab-content" class="lab-skip">Skip to lab content</a>
    <header class="border-b border-border bg-bg-primary">
      <div class="mx-auto flex max-w-[1440px] flex-wrap items-center justify-between gap-3 px-4 py-4 sm:px-6">
        <div class="flex items-center gap-3">
          <img src="/logo.svg" alt="" class="h-8 w-8 rounded-control" />
          <div>
            <RouterLink to="/" class="font-semibold hover:text-primary">Sing Box Easy</RouterLink>
            <p class="text-xs text-text-secondary">Design lab</p>
          </div>
        </div>
        <div class="flex items-center gap-3">
          <RouterLink to="/lab/login" class="text-xs font-medium hover:text-primary">Login preview</RouterLink>
          <span class="text-xs text-text-secondary">System theme · {{ darkMode ? 'Dark' : 'Light' }}</span>
          <RouterLink to="/dashboard" class="inline-flex items-center gap-2 rounded-control px-2 py-1.5 font-medium hover:bg-bg-tertiary focus-visible:outline-2 focus-visible:outline-primary">
            Open dashboard <ArrowRightIcon class="h-4 w-4" aria-hidden="true" />
          </RouterLink>
        </div>
      </div>
    </header>

    <div class="mx-auto grid max-w-[1440px] items-start lg:grid-cols-[200px_minmax(0,1fr)]">
      <aside class="px-4 py-4 sm:px-6 lg:sticky lg:top-0 lg:py-8">
        <nav aria-label="Lab sections" class="flex flex-wrap gap-1 lg:flex-col">
          <a v-for="section in sections" :key="section.id" :href="`#${section.id}`"
            class="inline-flex items-center gap-2 rounded-control px-3 py-2 text-text-secondary transition-colors hover:bg-bg-tertiary hover:text-text-primary focus-visible:outline-2 focus-visible:outline-primary">
            <component :is="section.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
            {{ section.label }}
          </a>
        </nav>
        <p class="mt-6 hidden text-xs leading-relaxed text-text-secondary lg:block">
          Rendered by the same components and styles as the dashboard
        </p>
      </aside>

      <main id="lab-content" tabindex="-1" class="min-w-0 space-y-8 px-4 pb-12 outline-none sm:px-6 lg:pt-8">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="max-w-2xl">
            <h1 class="text-3xl font-semibold">Design lab</h1>
            <p class="mt-2 leading-relaxed text-text-secondary">
              Sing Box Easy’s components, tokens, and typography, rendered together.
              Try the controls with local sample data.
            </p>
          </div>
          <Button variant="secondary" size="sm" @click="reset">
            <ArrowPathIcon class="h-4 w-4" aria-hidden="true" /> Reset samples
          </Button>
        </div>
        <p class="rounded-control border border-border bg-bg-tertiary px-3 py-2 text-xs text-text-secondary">
          The same surfaces, contrast, loading states, and motion used by the dashboard.
          This page is the live reference for the project’s visual language.
        </p>

        <PaletteStudio />

        <section id="workspace" aria-labelledby="workspace-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="workspace-title" class="text-lg font-semibold">Working example</h2>
            <span class="text-xs text-text-secondary">01 / Composition</span>
          </div>
          <div class="grid items-start gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(280px,1fr)]">
            <Card class="min-w-0 space-y-4">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h3 class="text-lg font-semibold">Outbounds</h3>
                  <p class="mt-1 text-xs text-text-secondary">A compact table with status, identifiers, and row actions</p>
                </div>
                <div class="flex items-center gap-2">
                  <Button variant="ghost" size="sm" action @click="rows.reverse()">Reverse order</Button>
                  <Badge variant="secondary">{{ rows.length }} samples</Badge>
                </div>
              </div>
              <div class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_144px]">
                <Input v-model="query" label="Search outbounds" placeholder="Name, server, or protocol" />
                <div>
                  <label for="lab-table-state" class="mb-1 block font-medium">Table state</label>
                  <Select v-model="tableState" input-id="lab-table-state" aria-label="Table state" :options="tableStates"
                    option-label="label" option-value="value" class="w-full" />
                </div>
              </div>
              <Table :columns="columns" :loading="tableState === 'loading'"
                :empty="tableState === 'empty' || !filteredRows.length" max-height="20rem">
                <template #empty>
                  <p class="font-medium text-text-primary">No outbounds to show</p>
                  <p class="mt-1 text-xs">Change the search or reset the samples</p>
                </template>
                <tr v-for="row in filteredRows" :key="row.tag">
                  <td>
                    <p class="font-medium">{{ row.tag }}</p>
                    <span class="inline-flex items-center gap-1 font-mono text-xs text-text-secondary">
                      {{ row.server }}:{{ row.port }}
                      <CopyIcon :text="`${row.server}:${row.port}`" :label="`Copy ${row.tag} address`" />
                    </span>
                  </td>
                  <td>{{ row.protocol }}</td>
                  <td><StatusIndicator :variant="row.status === 'Ready' ? 'success' : 'warning'" >{{ row.status }}</StatusIndicator></td>
                  <td class="col-actions">
                    <IconButton action="edit" @click="editSample(row)" :label="`Edit ${row.tag}`" />
                    <IconButton action="delete" @click="rows = rows.filter(item => item !== row)" :label="`Remove sample ${row.tag}`" />
                  </td>
                </tr>
              </Table>
              <div class="flex flex-wrap items-center justify-between gap-3 border-t border-border pt-3 text-xs text-text-secondary">
                <span>Sample node availability</span>
                <SegmentedProgress :percent="rows.length ? rows.filter(row => row.status === 'Ready').length / rows.length * 100 : 0"
                  :steps="6" show-info aria-label="Sample node availability" />
              </div>
            </Card>

            <Card>
              <form id="lab-outbound-form" class="space-y-4" novalidate @submit.prevent="saveSample">
                <div>
                  <h3 class="text-lg font-semibold">Outbound editor</h3>
                  <p class="mt-1 text-xs text-text-secondary">Save a sample to see it appear in the table</p>
                </div>
                <Input v-model="tag" label="Name" required :error="tagError" />
                <Input v-model="server" label="Server" required :error="serverError" />
                <div class="grid grid-cols-2 gap-3">
                  <Input v-model="port" label="Port" type="number" :min="1" :max="65535" required :error="portError" />
                  <div>
                    <label for="lab-protocol" class="mb-1 block font-medium">Protocol</label>
                    <Select v-model="protocol" input-id="lab-protocol" aria-label="Protocol" :options="protocols" filter
                      filter-placeholder="Search protocols" class="w-full" />
                  </div>
                </div>
                <label class="flex items-center gap-2">
                  <input v-model="enabled" type="checkbox" class="h-4 w-4 accent-primary" />
                  Mark this sample as ready
                </label>
                <div class="flex items-center justify-between gap-3 border-t border-border pt-3">
                  <span class="text-xs text-text-secondary" role="status">{{ saveFeedback || 'Local sample only' }}</span>
                  <Button type="submit" size="sm"><CheckIcon class="h-4 w-4" aria-hidden="true" /> Save sample</Button>
                </div>
              </form>
            </Card>
          </div>
        </section>

        <section id="components" aria-labelledby="components-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="components-title" class="text-lg font-semibold">Components</h2>
            <span class="text-xs text-text-secondary">02 / Controls</span>
          </div>
          <div class="grid items-start gap-4 xl:grid-cols-2">
            <Card class="space-y-5">
              <div>
                <h3 class="font-semibold">Buttons</h3>
                <p class="mt-1 text-xs text-text-secondary">Variants, sizes, and interactive states from Button.vue</p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <Button @click="notify('Primary action', 'info')">Primary</Button>
                <Button variant="secondary" @click="notify('Secondary action', 'info')">Secondary</Button>
                <Button variant="ghost" @click="notify('Ghost action', 'info')">Ghost</Button>
                <Button variant="success" @click="notify('Success action')">Success</Button>
                <Button variant="danger" @click="notify('Danger action', 'error')">Danger</Button>
              </div>
              <div class="flex flex-wrap items-center gap-2 border-t border-border pt-4">
                <Button variant="secondary" size="sm" @click="notify('Small button', 'info')">Small</Button>
                <Button variant="secondary" @click="notify('Medium button', 'info')">Medium</Button>
                <Button variant="secondary" size="lg" @click="notify('Large button', 'info')">Large</Button>
                <Button variant="secondary" pill aria-label="Preview icon button" @click="notify('Icon button', 'info')"><PlusIcon class="h-4 w-4" aria-hidden="true" /></Button>
              </div>
              <div class="flex flex-wrap items-center gap-2 border-t border-border pt-4">
                <Button :loading="sampleLoading" @click="simulateRequest">Simulate request</Button>
                <Button disabled>Disabled</Button>
                <Button variant="ghost" size="sm" action @click="notify('Table action', 'info')">Table action</Button>
              </div>
              <p class="min-h-5 text-xs text-text-secondary" :class="actionFeedback.severity === 'error' ? 'text-danger' : ''" role="status">{{ actionFeedback.summary }}</p>
            </Card>
            <Card class="space-y-4">
              <div>
                <h3 class="font-semibold">Form states</h3>
                <p class="mt-1 text-xs text-text-secondary">Shared input surfaces, validation, and keyboard behavior</p>
              </div>
              <Input label="Default" placeholder="Enter a value" hint="Helper text stays connected to its input" />
              <Input label="Invalid" model-value="70000" error="Use a port between 1 and 65535" />
              <Input label="Disabled" model-value="Managed by configuration" disabled />
            </Card>
            <Card class="space-y-4">
              <h3 class="font-semibold">Multi-value controls</h3>
              <div>
                <label id="lab-domains-label" class="mb-1 block font-medium">Domain matchers</label>
                <Chips v-model="domains" aria-labelledby="lab-domains-label" placeholder="Add a domain" />
                <p class="mt-1 text-xs text-text-secondary">Paste and edit a domain, then press Enter to add it</p>
              </div>
              <div>
                <label for="lab-networks" class="mb-1 block font-medium">Networks</label>
                <MultiSelect v-model="networks" input-id="lab-networks" :options="['TCP', 'UDP']"
                  placeholder="Select networks" class="w-full" />
              </div>
            </Card>
            <Card class="space-y-4">
              <div>
                <h3 class="font-semibold">Stable loading</h3>
                <p class="mt-1 text-xs text-text-secondary">The field keeps its dimensions while a placeholder replaces its content</p>
              </div>
              <div :aria-busy="sampleLoading" class="space-y-2">
                <div class="relative">
                  <Input label="Resolver result" model-value="192.0.2.10" :disabled="sampleLoading" :class="sampleLoading ? 'lab-loading-field' : ''" />
                  <Skeleton v-if="sampleLoading" class="absolute inset-x-0 bottom-0 h-8" />
                </div>
                <p class="text-xs text-text-secondary" role="status">{{ sampleLoading ? 'Loading sample result…' : 'Sample result ready' }}</p>
              </div>
              <Button variant="secondary" :loading="sampleLoading" @click="simulateRequest">Simulate request</Button>
            </Card>
            <Card class="space-y-4">
              <h3 class="font-semibold">Text & identifiers</h3>
              <Textarea v-model="notes" label="Notes" :rows="3" hint="The same textarea used in configuration forms" />
              <div class="flex items-center justify-between gap-3 rounded-control border border-border bg-bg-tertiary px-3 py-2">
                <code class="min-w-0 break-all text-xs">https://example.com/subscription</code>
                <CopyIcon text="https://example.com/subscription" label="Copy sample subscription URL" />
              </div>
            </Card>
          </div>
        </section>

        <section id="motion" aria-labelledby="motion-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="motion-title" class="text-lg font-semibold">Motion & navigation</h2>
            <span class="text-xs text-text-secondary">03 / Continuity</span>
          </div>
          <div class="grid items-start gap-4 xl:grid-cols-2">
            <Card class="space-y-4">
              <h3 class="font-semibold">Switch context</h3>
              <p class="text-xs text-text-secondary">Switch tabs quickly; the interface keeps responding while the previous view exits</p>
              <div ref="viewNav" role="tablist" aria-label="Sample pages" class="lab-view-tabs">
                <span ref="viewIndicator" class="lab-view-indicator" aria-hidden="true" />
                <button v-for="tab in viewTabs" :id="`lab-tab-${tab}`" :key="tab" type="button" role="tab"
                  :aria-selected="activeView === tab" :aria-controls="`lab-view-${tab}`" :tabindex="activeView === tab ? 0 : -1"
                  class="relative z-10 rounded-control px-3 py-2 focus-visible:outline-2 focus-visible:outline-primary"
                  @click="activeView = tab"
                  @keydown.right.prevent="stepView(tab, 1)"
                  @keydown.left.prevent="stepView(tab, -1)">
                  {{ tab }}
                </button>
              </div>
              <div class="grid">
                <Transition v-for="tab in viewTabs" :key="tab" name="ui-page">
                  <div v-show="activeView === tab" :id="`lab-view-${tab}`" role="tabpanel" :aria-labelledby="`lab-tab-${tab}`"
                    :inert="activeView !== tab" class="col-start-1 row-start-1 min-w-0 rounded-control border border-border bg-bg-tertiary p-4">
                    <h4 class="font-medium">{{ tab === 'Outbounds' ? 'Choose an outbound' : tab === 'DNS' ? 'Resolve a domain' : 'Match a route' }}</h4>
                    <p class="mt-2 text-text-secondary">{{ tab === 'Outbounds' ? 'Tokyo edge · VLESS · Ready' : tab === 'DNS' ? 'dns.example.com · DNS over HTTPS' : 'example.org → Tokyo edge' }}</p>
                    <p class="mt-3 text-xs text-text-secondary">Local sample view</p>
                  </div>
                </Transition>
              </div>
            </Card>
            <Card class="space-y-4">
              <h3 class="font-semibold">Search & progressive disclosure</h3>
              <div>
                <label for="lab-node-picker" class="mb-1 block font-medium">Find a sample node</label>
                <SearchPicker input-id="lab-node-picker" input-class="w-full" :options="fixtures.map(row => ({value:row.tag, label:row.tag}))"
                  placeholder="Search sample nodes" empty-message="No matching nodes" @select="pickedNode = $event" />
                <p class="mt-2 text-xs text-text-secondary" aria-live="polite">{{ pickedNode }}</p>
              </div>
              <Button variant="secondary" size="sm" aria-controls="lab-details" :aria-expanded="detailsOpen" @click="detailsOpen = !detailsOpen">
                {{ detailsOpen ? 'Hide details' : 'Show details' }}
              </Button>
              <div id="lab-details" class="ui-disclosure" :class="{'is-open':detailsOpen}" :inert="!detailsOpen" :aria-hidden="!detailsOpen">
                <div><div class="rounded-control border border-border bg-bg-tertiary p-3">
                  <p class="font-medium">A reversible interaction</p>
                  <p class="mt-2 text-xs text-text-secondary">Open and close again mid-transition. The same element returns along the same path without a timer or input lock.</p>
                </div></div>
              </div>
            </Card>
          </div>
        </section>

        <section id="feedback" aria-labelledby="feedback-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="feedback-title" class="text-lg font-semibold">Feedback & overlays</h2>
            <span class="text-xs text-text-secondary">04 / State</span>
          </div>
          <Card class="space-y-4">
            <p class="text-xs text-text-secondary">Labels describe an item; status uses a dot and plain text</p>
            <div class="flex flex-wrap items-center gap-2">
              <Badge variant="primary">Proxy</Badge><StatusIndicator variant="success">Running</StatusIndicator>
              <StatusIndicator variant="warning">Pending</StatusIndicator><StatusIndicator variant="danger">Stopped</StatusIndicator>
              <Badge variant="info">Information</Badge><Badge variant="secondary">Metadata</Badge>
            </div>
            <div class="grid gap-3 xl:grid-cols-2">
              <Alert type="success" title="Configuration validated">The sample configuration is ready to apply</Alert>
              <Alert type="warning" title="Subscription needs attention">One sample endpoint has not been checked</Alert>
              <Alert type="error" title="Connection unavailable">Check the endpoint before trying again</Alert>
              <Alert v-if="alertVisible" type="info" title="Local preview" closable @close="alertVisible = false">These controls do not change your running configuration</Alert>
              <Button v-else variant="ghost" @click="alertVisible = true">Restore information alert</Button>
            </div>
            <div class="flex flex-wrap items-center gap-3 border-t border-border pt-4">
              <Button variant="secondary" aria-controls="lab-listener-panel" :aria-expanded="listenerOpen"
                @click="listenerPanel?.toggle($event)">Configure listener</Button>
              <Button variant="ghost" @click="toast.add({group:'lab', severity:'success', summary:'Toast preview', detail:'A notification example', life:3000})">Preview toast</Button>
              <PopConfirm
                message="Clear the sample outbounds?"
                target="Design lab samples"
                details="Only this page's sample rows will be removed. Reset samples restores them."
                align="left"
                confirm-label="Clear samples"
                cancel-label="Cancel"
                @confirm="rows = []; overlayFeedback = 'Sample outbounds cleared'"
              />
            </div>
            <p class="min-h-5 text-xs text-text-secondary" role="status">{{ overlayFeedback || 'Editors and confirmations open beside their trigger' }}</p>
          </Card>
        </section>

        <section id="foundations" aria-labelledby="foundations-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="foundations-title" class="text-lg font-semibold">Foundations</h2>
            <span class="text-xs text-text-secondary">05 / Tokens & type</span>
          </div>
          <Card class="space-y-4">
            <div>
              <h3 class="font-semibold">Color roles</h3>
              <p class="mt-1 text-xs text-text-secondary">Values read from the active stylesheet; appearance follows your system theme</p>
            </div>
            <div class="grid grid-cols-2 gap-3 sm:grid-cols-4">
              <div v-for="color in colors" :key="color.token" class="min-w-0">
                <div class="mb-2 h-12 rounded-control border border-border" :style="{ background: `var(${color.token})` }" />
                <p class="font-medium">{{ color.label }}</p>
                <code class="mt-1 block break-all text-xs text-text-secondary">{{ color.token }}</code>
                <span class="mt-1 block break-all font-mono text-xs text-text-secondary">{{ tokenValues[color.token] }}</span>
              </div>
            </div>
          </Card>
          <div class="grid items-start gap-4 xl:grid-cols-2">
            <Card class="space-y-4">
              <h3 class="font-semibold">Typography</h3>
              <div class="space-y-3">
                <div class="border-b border-border pb-3"><p class="text-3xl font-semibold">Network overview</p><code class="text-xs text-text-secondary">text-3xl · Page heading</code></div>
                <div class="border-b border-border pb-3"><p class="text-lg font-semibold">Outbound configuration</p><code class="text-xs text-text-secondary">text-lg · Section heading</code></div>
                <div><p class="text-sm">Route traffic through the selected outbound</p><code class="text-xs text-text-secondary">text-sm · Body & controls</code></div>
                <div><p class="text-xs text-text-secondary">Updated just now · 3 sample endpoints</p><code class="text-xs text-text-secondary">text-xs · Supporting text</code></div>
                <p class="font-mono text-sm tabular-nums">192.0.2.10:443 · 24.8 MB/s</p>
              </div>
              <p class="break-words border-t border-border pt-3 text-xs text-text-secondary">{{ fontFamily }}</p>
            </Card>
            <Card class="space-y-4">
              <h3 class="font-semibold">Surfaces & spacing</h3>
              <div class="grid grid-cols-2 gap-3">
                <div class="liquid-glass rounded-surface p-4"><p class="font-medium">Resting surface</p><code class="mt-1 block text-xs text-text-secondary">liquid-glass</code></div>
                <div class="liquid-glass-float rounded-surface p-4"><p class="font-medium">Floating surface</p><code class="mt-1 block text-xs text-text-secondary">liquid-glass-float</code></div>
              </div>
              <dl class="divide-y divide-border">
                <div v-for="metric in metrics" :key="metric.token" class="flex items-center justify-between gap-3 py-2">
                  <dt>{{ metric.label }}</dt><dd class="font-mono text-xs text-text-secondary">{{ tokenValues[metric.token] }}</dd>
                </div>
              </dl>
              <p class="text-xs text-text-secondary">Focus, hover, disabled states, and reduced motion come from the shared styles and components</p>
            </Card>
          </div>
        </section>
        <footer class="border-t border-border pt-4 text-xs text-text-secondary">
          Vue 3 · Tailwind CSS 4 · PrimeVue unstyled · Heroicons · System typography
        </footer>
      </main>
    </div>

    <Toast group="lab" position="top-center" class="lab-preview" />
    <Popover id="lab-listener-panel" ref="listenerPanel" class="lab-preview" aria-labelledby="lab-listener-title"
      @show="listenerOpen = true" @hide="listenerOpen = false">
      <form class="space-y-4" @submit.prevent="listenerPanel?.hide(); overlayFeedback = `${listenerName || 'Local listener'} saved locally`">
        <h3 id="lab-listener-title" class="font-semibold">Sample listener</h3>
        <p class="text-xs text-text-secondary">Edit here, without leaving the action that opened it</p>
        <Input v-model="listenerName" label="Name" />
        <div>
          <label for="lab-dialog-protocol" class="mb-1 block text-sm font-medium">Protocol</label>
          <Select v-model="listenerProtocol" input-id="lab-dialog-protocol" aria-label="Protocol" :options="['Mixed', 'HTTP', 'SOCKS']" class="w-full" />
        </div>
        <div class="flex justify-end gap-2 border-t border-border pt-3">
          <Button variant="ghost" @click="listenerPanel?.hide()">Cancel</Button>
          <Button type="submit">Save sample</Button>
        </div>
      </form>
    </Popover>
  </div>
</template>

<style scoped>
/* Lab layout only; all specimens use shared dashboard styles. */
.lab-skip {
  position: absolute;
  top: 0;
  left: 1rem;
  z-index: 100;
  padding: .75rem 1rem;
  border-radius: var(--radius-control);
  background: var(--color-bg-primary);
  color: var(--color-text-primary);
  transform: translateY(-120%);
}
.lab-skip:focus { transform: translateY(0); }
.lab-loading-field :deep(input) { opacity: 0; }
.lab-view-tabs { position: relative; display: flex; isolation: isolate; }
.lab-view-indicator {
  position: absolute;
  top: 0;
  left: 0;
  background: var(--color-bg-tertiary);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-control);
  pointer-events: none;
}
.lab-view-indicator.is-animated {
  transition: transform var(--motion-page) var(--motion-continuous),
    width var(--motion-page) var(--motion-continuous),
    height var(--motion-page) var(--motion-continuous);
}
@media (prefers-reduced-motion: reduce) {
  .lab-view-indicator.is-animated { transition: none; }
}
</style>
