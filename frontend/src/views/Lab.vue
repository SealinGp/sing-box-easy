<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

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
  { get label() { return t('lab.populated') }, value: 'data' },
  { get label() { return t('lab.empty') }, value: 'empty' },
  { get label() { return t('lab.loading') }, value: 'loading' },
]
const columns: TableColumn[] = [
  { key: 'tag', get label() { return t('lab.outbound') } },
  { key: 'protocol', get label() { return t('lab.protocol') } },
  { key: 'status', get label() { return t('lab.status') } },
  { key: 'actions', get label() { return t('lab.actions') }, align: 'right' },
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
const actionFeedback = ref({ summary: t('lab.chooseAButtonToPreviewItsFeedback'), severity: 'info' })
const alertVisible = ref(true)
const sampleLoading = ref(false)
let sampleRequest: ReturnType<typeof setTimeout> | undefined
function simulateRequest() {
  sampleLoading.value = true
  sampleRequest = setTimeout(() => { sampleLoading.value = false }, 1600)
}
const toast = useToast()
const tagError = computed(() => submitted.value && !tag.value.trim() ? t('lab.enterAnOutboundName') : '')
const serverError = computed(() => submitted.value && !server.value.trim() ? t('lab.enterAServerAddress') : '')
const portError = computed(() => submitted.value && (!Number.isInteger(Number(port.value)) || Number(port.value) < 1 || Number(port.value) > 65535)
  ? t('lab.useAPortBetween1And65535') : '')

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
  saveFeedback.value = t('lab.sampleSaved')
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
  actionFeedback.value = { summary: t('lab.chooseAButtonToPreviewItsFeedback'), severity: 'info' }
  alertVisible.value = true
  activeView.value = 'Outbounds'
  detailsOpen.value = false
  pickedNode.value = 'No node selected'
  toast.removeAllGroups()
}

const sections = [
  { id: 'palettes', get label() { return t('lab.colorDirections') }, icon: SwatchIcon },
  { id: 'workspace', get label() { return t('lab.workingExample') }, icon: CubeTransparentIcon },
  { id: 'components', get label() { return t('lab.components') }, icon: PlusIcon },
  { id: 'motion', get label() { return t('lab.motionNavigation') }, icon: ArrowRightIcon },
  { id: 'feedback', get label() { return t('lab.feedbackOverlays') }, icon: BeakerIcon },
  { id: 'foundations', get label() { return t('lab.foundations') }, icon: SwatchIcon },
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
  { get label() { return t('lab.primary') }, token: '--color-primary' },
  { get label() { return t('lab.background') }, token: '--color-bg-primary' },
  { get label() { return t('lab.primaryText') }, token: '--color-text-primary' },
  { get label() { return t('lab.secondaryText') }, token: '--color-text-secondary' },
  { get label() { return t('lab.border') }, token: '--color-border' },
  { get label() { return t('lab.success') }, token: '--color-success' },
  { get label() { return t('lab.warning') }, token: '--color-warning' },
  { get label() { return t('lab.danger') }, token: '--color-danger' },
]
const metrics = [
  { get label() { return t('lab.tableRadius') }, token: '--radius-table' },
  { get label() { return t('lab.badgeRadius') }, token: '--radius-badge' },
  { get label() { return t('lab.controlRadius') }, token: '--radius-control' },
  { get label() { return t('lab.surfaceRadius') }, token: '--radius-surface' },
  { get label() { return t('lab.pageGutter') }, token: '--space-gutter' },
  { get label() { return t('lab.cardPadding') }, token: '--space-card' },
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
  <div ref="preview" class="lab-preview liquid-app min-w-0 text-sm text-text-primary">
    <div class="space-y-5">
      <div class="border-b border-border pb-3">
        <nav :aria-label="t('lab.labSections')" class="flex flex-wrap gap-1">
          <a v-for="section in sections" :key="section.id" :href="`#${section.id}`"
            class="inline-flex items-center gap-2 rounded-control px-3 py-2 text-text-secondary transition-colors hover:bg-bg-tertiary hover:text-text-primary focus-visible:outline-2 focus-visible:outline-primary">
            <component :is="section.icon" class="h-4 w-4 shrink-0" aria-hidden="true" />
            {{ section.label }}
          </a>
        </nav>
      </div>

      <div id="lab-content" tabindex="-1" class="min-w-0 space-y-8 pb-8 outline-none">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div class="max-w-2xl">
            <h1 class="text-xl font-semibold">{{ t('lab.themeStyle') }}</h1>
            <p class="mt-2 leading-relaxed text-text-secondary"> {{ t('lab.singBoxEasySComponentsTokensAndTypographyRenderedTogether') }} </p>
          </div>
          <div class="flex flex-wrap items-center gap-3">
          <RouterLink to="/lab/login" class="text-xs font-medium hover:text-primary">{{ t('lab.loginPreview') }}</RouterLink>
          <span class="text-xs text-text-secondary">{{ darkMode ? t('lab.dark') : t('lab.light') }} {{ t('lab.theme') }}</span>
          <Button variant="secondary" size="sm" @click="reset">
            <ArrowPathIcon class="h-4 w-4" aria-hidden="true" /> {{ t('lab.resetSamples') }} </Button>
          </div>
        </div>
        <p class="rounded-control border border-border bg-bg-tertiary px-3 py-2 text-xs text-text-secondary"> {{ t('lab.theSameSurfacesContrastLoadingStatesAndMotionUsedBy') }} </p>

        <PaletteStudio />

        <section id="workspace" aria-labelledby="workspace-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="workspace-title" class="text-lg font-semibold">{{ t('lab.workingExample') }}</h2>
            <span class="text-xs text-text-secondary">{{ t('lab.01Composition') }}</span>
          </div>
          <div class="grid items-start gap-4 xl:grid-cols-[minmax(0,1.4fr)_minmax(280px,1fr)]">
            <Card class="min-w-0 space-y-4">
              <div class="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <h3 class="text-lg font-semibold">{{ t('lab.outbounds') }}</h3>
                  <p class="mt-1 text-xs text-text-secondary">{{ t('lab.aCompactTableWithStatusIdentifiersAndRowActions') }}</p>
                </div>
                <div class="flex items-center gap-2">
                  <Button variant="ghost" size="sm" action @click="rows.reverse()">{{ t('lab.reverseOrder') }}</Button>
                  <Badge variant="secondary">{{ rows.length }} {{ t('lab.samples') }}</Badge>
                </div>
              </div>
              <div class="grid gap-3 sm:grid-cols-[minmax(0,1fr)_144px]">
                <Input v-model="query" :label="t('lab.searchOutbounds')" :placeholder="t('lab.nameServerOrProtocol')" />
                <div>
                  <label for="lab-table-state" class="mb-1 block font-medium">{{ t('lab.tableState') }}</label>
                  <Select v-model="tableState" input-id="lab-table-state" :aria-label="t('lab.tableState')" :options="tableStates"
                    option-label="label" option-value="value" class="w-full" />
                </div>
              </div>
              <Table :columns="columns" :loading="tableState === 'loading'"
                :empty="tableState === 'empty' || !filteredRows.length" max-height="20rem">
                <template #empty>
                  <p class="font-medium text-text-primary">{{ t('lab.noOutboundsToShow') }}</p>
                  <p class="mt-1 text-xs">{{ t('lab.changeTheSearchOrResetTheSamples') }}</p>
                </template>
                <tr v-for="row in filteredRows" :key="row.tag">
                  <td>
                    <p class="font-medium">{{ row.tag }}</p>
                    <span class="inline-flex items-center gap-1 font-mono text-xs text-text-secondary">
                      {{ row.server }}:{{ row.port }}
                      <CopyIcon :text="`${row.server}:${row.port}`" :label="t('lab.copyAddress', { name: row.tag })" />
                    </span>
                  </td>
                  <td>{{ row.protocol }}</td>
                  <td><StatusIndicator :variant="row.status === 'Ready' ? 'success' : 'warning'" >{{ row.status === 'Ready' ? t('lab.ready') : t('lab.pending') }}</StatusIndicator></td>
                  <td class="col-actions">
                    <IconButton action="edit" @click="editSample(row)" :label="t('lab.editNamed', { name: row.tag })" />
                    <IconButton action="delete" @click="rows = rows.filter(item => item !== row)" :label="t('lab.removeNamed', { name: row.tag })" />
                  </td>
                </tr>
              </Table>
              <div class="flex flex-wrap items-center justify-between gap-3 border-t border-border pt-3 text-xs text-text-secondary">
                <span>{{ t('lab.sampleNodeAvailability') }}</span>
                <SegmentedProgress :percent="rows.length ? rows.filter(row => row.status === 'Ready').length / rows.length * 100 : 0"
                  :steps="6" show-info :aria-label="t('lab.sampleNodeAvailability')" />
              </div>
            </Card>

            <Card>
              <form id="lab-outbound-form" class="space-y-4" novalidate @submit.prevent="saveSample">
                <div>
                  <h3 class="text-lg font-semibold">{{ t('lab.outboundEditor') }}</h3>
                  <p class="mt-1 text-xs text-text-secondary">{{ t('lab.saveASampleToSeeItAppearInTheTable') }}</p>
                </div>
                <Input v-model="tag" :label="t('lab.name')" required :error="tagError" />
                <Input v-model="server" :label="t('lab.server')" required :error="serverError" />
                <div class="grid grid-cols-2 gap-3">
                  <Input v-model="port" :label="t('lab.port')" type="number" :min="1" :max="65535" required :error="portError" />
                  <div>
                    <label for="lab-protocol" class="mb-1 block font-medium">{{ t('lab.protocol') }}</label>
                    <Select v-model="protocol" input-id="lab-protocol" :aria-label="t('lab.protocol')" :options="protocols" filter
                      :filter-placeholder="t('lab.searchProtocols')" class="w-full" />
                  </div>
                </div>
                <label class="flex items-center gap-2">
                  <input v-model="enabled" type="checkbox" class="h-4 w-4 accent-primary" /> {{ t('lab.markThisSampleAsReady') }} </label>
                <div class="flex items-center justify-between gap-3 border-t border-border pt-3">
                  <span class="text-xs text-text-secondary" role="status">{{ saveFeedback || t('lab.localSampleOnly') }}</span>
                  <Button type="submit" size="sm"><CheckIcon class="h-4 w-4" aria-hidden="true" /> {{ t('lab.saveSample') }}</Button>
                </div>
              </form>
            </Card>
          </div>
        </section>

        <section id="components" aria-labelledby="components-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="components-title" class="text-lg font-semibold">{{ t('lab.components') }}</h2>
            <span class="text-xs text-text-secondary">{{ t('lab.02Controls') }}</span>
          </div>
          <div class="grid items-start gap-4 xl:grid-cols-2">
            <Card class="space-y-5">
              <div>
                <h3 class="font-semibold">{{ t('lab.buttons') }}</h3>
                <p class="mt-1 text-xs text-text-secondary">{{ t('lab.variantsSizesAndInteractiveStatesFromButtonVue') }}</p>
              </div>
              <div class="flex flex-wrap items-center gap-2">
                <Button @click="notify(t('lab.primaryAction'), 'info')">{{ t('lab.primary') }}</Button>
                <Button variant="secondary" @click="notify(t('lab.secondaryAction'), 'info')">{{ t('lab.secondary') }}</Button>
                <Button variant="ghost" @click="notify(t('lab.ghostAction'), 'info')">{{ t('lab.ghost') }}</Button>
                <Button variant="success" @click="notify(t('lab.successAction'))">{{ t('lab.success') }}</Button>
                <Button variant="danger" @click="notify(t('lab.dangerAction'), 'error')">{{ t('lab.danger') }}</Button>
              </div>
              <div class="flex flex-wrap items-center gap-2 border-t border-border pt-4">
                <Button variant="secondary" size="sm" @click="notify(t('lab.smallButton'), 'info')">{{ t('lab.small') }}</Button>
                <Button variant="secondary" @click="notify(t('lab.mediumButton'), 'info')">{{ t('lab.medium') }}</Button>
                <Button variant="secondary" size="lg" @click="notify(t('lab.largeButton'), 'info')">{{ t('lab.large') }}</Button>
                <Button variant="secondary" pill :aria-label="t('lab.previewIconButton')" @click="notify(t('lab.iconButton'), 'info')"><PlusIcon class="h-4 w-4" aria-hidden="true" /></Button>
              </div>
              <div class="flex flex-wrap items-center gap-2 border-t border-border pt-4">
                <Button :loading="sampleLoading" @click="simulateRequest">{{ t('lab.simulateRequest') }}</Button>
                <Button disabled>{{ t('lab.disabled') }}</Button>
                <Button variant="ghost" size="sm" action @click="notify(t('lab.tableAction'), 'info')">{{ t('lab.tableAction') }}</Button>
              </div>
              <p class="min-h-5 text-xs text-text-secondary" :class="actionFeedback.severity === 'error' ? 'text-danger' : ''" role="status">{{ actionFeedback.summary }}</p>
            </Card>
            <Card class="space-y-4">
              <div>
                <h3 class="font-semibold">{{ t('lab.formStates') }}</h3>
                <p class="mt-1 text-xs text-text-secondary">{{ t('lab.sharedInputSurfacesValidationAndKeyboardBehavior') }}</p>
              </div>
              <Input :label="t('lab.default')" :placeholder="t('lab.enterAValue')" :hint="t('lab.helperTextStaysConnectedToItsInput')" />
              <Input :label="t('lab.invalid')" model-value="70000" error="Use a port between 1 and 65535" />
              <Input :label="t('lab.disabled')" model-value="Managed by configuration" disabled />
            </Card>
            <Card class="space-y-4">
              <h3 class="font-semibold">{{ t('lab.multiValueControls') }}</h3>
              <div>
                <label id="lab-domains-label" class="mb-1 block font-medium">{{ t('lab.domainMatchers') }}</label>
                <Chips v-model="domains" aria-labelledby="lab-domains-label" :placeholder="t('lab.addADomain')" />
                <p class="mt-1 text-xs text-text-secondary">{{ t('lab.pasteAndEditADomainThenPressEnterToAdd') }}</p>
              </div>
              <div>
                <label for="lab-networks" class="mb-1 block font-medium">{{ t('lab.networks') }}</label>
                <MultiSelect v-model="networks" input-id="lab-networks" :options="['TCP', 'UDP']"
                  :placeholder="t('lab.selectNetworks')" class="w-full" />
              </div>
            </Card>
            <Card class="space-y-4">
              <div>
                <h3 class="font-semibold">{{ t('lab.stableLoading') }}</h3>
                <p class="mt-1 text-xs text-text-secondary">{{ t('lab.theFieldKeepsItsDimensionsWhileAPlaceholderReplacesIts') }}</p>
              </div>
              <div :aria-busy="sampleLoading" class="space-y-2">
                <div class="relative">
                  <Input :label="t('lab.resolverResult')" model-value="192.0.2.10" :disabled="sampleLoading" :class="sampleLoading ? 'lab-loading-field' : ''" />
                  <Skeleton v-if="sampleLoading" class="absolute inset-x-0 bottom-0 h-8" />
                </div>
                <p class="text-xs text-text-secondary" role="status">{{ sampleLoading ? t('lab.loadingSampleResult') : t('lab.sampleResultReady') }}</p>
              </div>
              <Button variant="secondary" :loading="sampleLoading" @click="simulateRequest">{{ t('lab.simulateRequest') }}</Button>
            </Card>
            <Card class="space-y-4">
              <h3 class="font-semibold">{{ t('lab.textIdentifiers') }}</h3>
              <Textarea v-model="notes" :label="t('lab.notes')" :rows="3" :hint="t('lab.theSameTextareaUsedInConfigurationForms')" />
              <div class="flex items-center justify-between gap-3 rounded-control border border-border bg-bg-tertiary px-3 py-2">
                <code class="min-w-0 break-all text-xs">https://example.com/subscription</code>
                <CopyIcon text="https://example.com/subscription" :label="t('lab.copySampleSubscriptionUrl')" />
              </div>
            </Card>
          </div>
        </section>

        <section id="motion" aria-labelledby="motion-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="motion-title" class="text-lg font-semibold">{{ t('lab.motionNavigation') }}</h2>
            <span class="text-xs text-text-secondary">{{ t('lab.03Continuity') }}</span>
          </div>
          <div class="grid items-start gap-4 xl:grid-cols-2">
            <Card class="space-y-4">
              <h3 class="font-semibold">{{ t('lab.switchContext') }}</h3>
              <p class="text-xs text-text-secondary">{{ t('lab.switchTabsQuicklyTheInterfaceKeepsRespondingWhileThePrevious') }}</p>
              <div ref="viewNav" role="tablist" :aria-label="t('lab.samplePages')" class="lab-view-tabs">
                <span ref="viewIndicator" class="lab-view-indicator" aria-hidden="true" />
                <button v-for="tab in viewTabs" :id="`lab-tab-${tab}`" :key="tab" type="button" role="tab"
                  :aria-selected="activeView === tab" :aria-controls="`lab-view-${tab}`" :tabindex="activeView === tab ? 0 : -1"
                  class="relative z-10 rounded-control px-3 py-2 focus-visible:outline-2 focus-visible:outline-primary"
                  @click="activeView = tab"
                  @keydown.right.prevent="stepView(tab, 1)"
                  @keydown.left.prevent="stepView(tab, -1)">
                  {{ tab === 'DNS' ? 'DNS' : t('lab.' + tab.toLowerCase()) }}
                </button>
              </div>
              <div class="grid">
                <Transition v-for="tab in viewTabs" :key="tab" name="ui-page">
                  <div v-show="activeView === tab" :id="`lab-view-${tab}`" role="tabpanel" :aria-labelledby="`lab-tab-${tab}`"
                    :inert="activeView !== tab" class="col-start-1 row-start-1 min-w-0 rounded-control border border-border bg-bg-tertiary p-4">
                    <h4 class="font-medium">{{ tab === 'Outbounds' ? t('lab.chooseAnOutbound') : tab === 'DNS' ? t('lab.resolveADomain') : t('lab.matchARoute') }}</h4>
                    <p class="mt-2 text-text-secondary">{{ tab === 'Outbounds' ? t('lab.tokyoEdgeVlessReady') : tab === 'DNS' ? 'dns.example.com · DNS over HTTPS' : t('lab.exampleOrgTokyoEdge') }}</p>
                    <p class="mt-3 text-xs text-text-secondary">{{ t('lab.localSampleView') }}</p>
                  </div>
                </Transition>
              </div>
            </Card>
            <Card class="space-y-4">
              <h3 class="font-semibold">{{ t('lab.searchProgressiveDisclosure') }}</h3>
              <div>
                <label for="lab-node-picker" class="mb-1 block font-medium">{{ t('lab.findASampleNode') }}</label>
                <SearchPicker input-id="lab-node-picker" input-class="w-full" :options="fixtures.map(row => ({value:row.tag, label:row.tag}))"
                  :placeholder="t('lab.searchSampleNodes')" empty-message="No matching nodes" @select="pickedNode = $event" />
                <p class="mt-2 text-xs text-text-secondary" aria-live="polite">{{ pickedNode }}</p>
              </div>
              <Button variant="secondary" size="sm" aria-controls="lab-details" :aria-expanded="detailsOpen" @click="detailsOpen = !detailsOpen">
                {{ detailsOpen ? t('lab.hideDetails') : t('lab.showDetails') }}
              </Button>
              <div id="lab-details" class="ui-disclosure" :class="{'is-open':detailsOpen}" :inert="!detailsOpen" :aria-hidden="!detailsOpen">
                <div><div class="rounded-control border border-border bg-bg-tertiary p-3">
                  <p class="font-medium">{{ t('lab.aReversibleInteraction') }}</p>
                  <p class="mt-2 text-xs text-text-secondary">{{ t('lab.openAndCloseAgainMidTransitionTheSameElementReturns') }}</p>
                </div></div>
              </div>
            </Card>
          </div>
        </section>

        <section id="feedback" aria-labelledby="feedback-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="feedback-title" class="text-lg font-semibold">{{ t('lab.feedbackOverlays') }}</h2>
            <span class="text-xs text-text-secondary">{{ t('lab.04State') }}</span>
          </div>
          <Card class="space-y-4">
            <p class="text-xs text-text-secondary">{{ t('lab.labelsDescribeAnItemStatusUsesADotAndPlain') }}</p>
            <div class="flex flex-wrap items-center gap-2">
              <Badge variant="primary">{{ t('lab.proxy') }}</Badge><StatusIndicator variant="success">{{ t('lab.running') }}</StatusIndicator>
              <StatusIndicator variant="warning">{{ t('lab.pending') }}</StatusIndicator><StatusIndicator variant="danger">{{ t('lab.stopped') }}</StatusIndicator>
              <Badge variant="info">{{ t('lab.information') }}</Badge><Badge variant="secondary">{{ t('lab.metadata') }}</Badge>
            </div>
            <div class="grid gap-3 xl:grid-cols-2">
              <Alert type="success" :title="t('lab.configurationValidated')">{{ t('lab.theSampleConfigurationIsReadyToApply') }}</Alert>
              <Alert type="warning" :title="t('lab.subscriptionNeedsAttention')">{{ t('lab.oneSampleEndpointHasNotBeenChecked') }}</Alert>
              <Alert type="error" :title="t('lab.connectionUnavailable')">{{ t('lab.checkTheEndpointBeforeTryingAgain') }}</Alert>
              <Alert v-if="alertVisible" type="info" :title="t('lab.localPreview')" closable @close="alertVisible = false">{{ t('lab.theseControlsDoNotChangeYourRunningConfiguration') }}</Alert>
              <Button v-else variant="ghost" @click="alertVisible = true">{{ t('lab.restoreInformationAlert') }}</Button>
            </div>
            <div class="flex flex-wrap items-center gap-3 border-t border-border pt-4">
              <Button variant="secondary" aria-controls="lab-listener-panel" :aria-expanded="listenerOpen"
                @click="listenerPanel?.toggle($event)">{{ t('lab.configureListener') }}</Button>
              <Button variant="ghost" @click="toast.add({group:'lab', severity:'success', summary:t('lab.toastPreview'), detail:t('lab.aNotificationExample'), life:3000})">{{ t('lab.previewToast') }}</Button>
              <PopConfirm
                message="Clear the sample outbounds?"
                :target="t('lab.designLabSamples')"
                :details="t('lab.onlyThisPageSSampleRowsWillBeRemovedReset')"
                align="left"
                :confirm-label="t('lab.clearSamples')"
                :cancel-label="t('lab.cancel')"
                @confirm="rows = []; overlayFeedback = t('lab.sampleOutboundsCleared')"
              />
            </div>
            <p class="min-h-5 text-xs text-text-secondary" role="status">{{ overlayFeedback || t('lab.editorsAndConfirmationsOpenBesideTheirTrigger') }}</p>
          </Card>
        </section>

        <section id="foundations" aria-labelledby="foundations-title" class="scroll-mt-4 space-y-4">
          <div class="flex items-baseline justify-between gap-3">
            <h2 id="foundations-title" class="text-lg font-semibold">{{ t('lab.foundations') }}</h2>
            <span class="text-xs text-text-secondary">{{ t('lab.05TokensType') }}</span>
          </div>
          <Card class="space-y-4">
            <div>
              <h3 class="font-semibold">{{ t('lab.colorRoles') }}</h3>
              <p class="mt-1 text-xs text-text-secondary">{{ t('lab.valuesReadFromTheActiveStylesheetAppearanceFollowsYourSystem') }}</p>
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
              <h3 class="font-semibold">{{ t('lab.typography') }}</h3>
              <div class="space-y-3">
                <div class="border-b border-border pb-3"><p class="text-3xl font-semibold">{{ t('lab.networkOverview') }}</p><code class="text-xs text-text-secondary">{{ t('lab.text3xlPageHeading') }}</code></div>
                <div class="border-b border-border pb-3"><p class="text-lg font-semibold">{{ t('lab.outboundConfiguration') }}</p><code class="text-xs text-text-secondary">{{ t('lab.textLgSectionHeading') }}</code></div>
                <div><p class="text-sm">{{ t('lab.routeTrafficThroughTheSelectedOutbound') }}</p><code class="text-xs text-text-secondary">{{ t('lab.textSmBodyControls') }}</code></div>
                <div><p class="text-xs text-text-secondary">{{ t('lab.updatedJustNow3SampleEndpoints') }}</p><code class="text-xs text-text-secondary">{{ t('lab.textXsSupportingText') }}</code></div>
                <p class="font-mono text-sm tabular-nums">192.0.2.10:443 · 24.8 MB/s</p>
              </div>
              <p class="break-words border-t border-border pt-3 text-xs text-text-secondary">{{ fontFamily }}</p>
            </Card>
            <Card class="space-y-4">
              <h3 class="font-semibold">{{ t('lab.surfacesSpacing') }}</h3>
              <div class="grid grid-cols-2 gap-3">
                <div class="liquid-glass rounded-surface p-4"><p class="font-medium">{{ t('lab.restingSurface') }}</p><code class="mt-1 block text-xs text-text-secondary">liquid-glass</code></div>
                <div class="liquid-glass-float rounded-surface p-4"><p class="font-medium">{{ t('lab.floatingSurface') }}</p><code class="mt-1 block text-xs text-text-secondary">liquid-glass-float</code></div>
              </div>
              <dl class="divide-y divide-border">
                <div v-for="metric in metrics" :key="metric.token" class="flex items-center justify-between gap-3 py-2">
                  <dt>{{ metric.label }}</dt><dd class="font-mono text-xs text-text-secondary">{{ tokenValues[metric.token] }}</dd>
                </div>
              </dl>
              <p class="text-xs text-text-secondary">{{ t('lab.focusHoverDisabledStatesAndReducedMotionComeFromThe') }}</p>
            </Card>
          </div>
        </section>
        <footer class="border-t border-border pt-4 text-xs text-text-secondary"> {{ t('lab.vue3TailwindCss4PrimevueUnstyledHeroiconsSystemTypography') }} </footer>
      </div>
    </div>

    <Toast group="lab" position="top-center" class="lab-preview" />
    <Popover id="lab-listener-panel" ref="listenerPanel" class="lab-preview" aria-labelledby="lab-listener-title"
      @show="listenerOpen = true" @hide="listenerOpen = false">
      <form class="space-y-4" @submit.prevent="listenerPanel?.hide(); overlayFeedback = t('lab.savedLocally', { name: listenerName })">
        <h3 id="lab-listener-title" class="font-semibold">{{ t('lab.sampleListener') }}</h3>
        <p class="text-xs text-text-secondary">{{ t('lab.editHereWithoutLeavingTheActionThatOpenedIt') }}</p>
        <Input v-model="listenerName" :label="t('lab.name')" />
        <div>
          <label for="lab-dialog-protocol" class="mb-1 block text-sm font-medium">{{ t('lab.protocol') }}</label>
          <Select v-model="listenerProtocol" input-id="lab-dialog-protocol" :aria-label="t('lab.protocol')" :options="['Mixed', 'HTTP', 'SOCKS']" class="w-full" />
        </div>
        <div class="flex justify-end gap-2 border-t border-border pt-3">
          <Button variant="ghost" @click="listenerPanel?.hide()">{{ t('lab.cancel') }}</Button>
          <Button type="submit">{{ t('lab.saveSample') }}</Button>
        </div>
      </form>
    </Popover>
  </div>
</template>

<style scoped>
/* Lab layout only; all specimens use shared dashboard styles. */
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
