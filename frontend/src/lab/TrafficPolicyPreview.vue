<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

import { createDNSResponsePrefix } from './dnsResponseTag'
import { computed, nextTick, onBeforeUnmount, onDeactivated, ref, watch } from 'vue'
import { PlusIcon, ArrowRightIcon, GlobeAltIcon, ArrowsRightLeftIcon, CheckCircleIcon } from '@heroicons/vue/24/outline'
import Button from '../components/Button.vue'
import Card from '../components/Card.vue'
import Textarea from '../components/Textarea.vue'
import IconButton from '../components/IconButton.vue'
import Skeleton from '../components/Skeleton.vue'
import Select from '../volt/Select.vue'
import Popover from '../volt/Popover.vue'
import DNSGroupForm from './DNSGroupForm.vue'
import CopyIcon from '../components/CopyIcon.vue'
import { buildTrafficPolicyConfig, matchValues } from './trafficPolicyConfig'

import { previewDNSServers, type PreviewDNSGroup } from './dnsGroupPreview'
const props = defineProps<{ groups: PreviewDNSGroup[] }>()
const emit = defineEmits<{ manageGroups: []; groupCreated: [group: PreviewDNSGroup] }>()

type Condition = { id: number; kind: string; value: string }
const kinds = [{ get label() { return t('lab.domainSuffix') }, value: 'domain_suffix' }, { get label() { return t('lab.exactDomain') }, value: 'domain' }, { get label() { return t('lab.ruleSet') }, value: 'rule_set' }, { get label() { return t('lab.ipRange') }, value: 'ip_cidr' }]
const profiles = computed(() => [
  ...props.groups.filter(g => !g.drift).map(g => ({ label: t('lab.parallelGroupName', { name: g.name }), value: g.id, servers: g.servers, get path() { return t('lab.pathsConfiguredOnEachResolver') }, timeout: t('lab.seconds', { n: g.timeout }) })),
  { get label() { return t('lab.googledotSingleServer') }, value: 'google', servers: ['GoogleDoT'], get path() { return t('lab.viaAutoSelect') }, get timeout() { return t('lab.5Seconds') } },
  { get label() { return t('lab.leaveDnsUnchanged') }, value: 'unchanged', servers: [], get path() { return t('lab.existingDnsRules') }, timeout: '' },
])
const dnsSelect = ref<InstanceType<typeof Select>>()
const dnsSearch = ref('')
const groupPopover = ref<InstanceType<typeof Popover>>()
const groupPopoverOpen = ref(false)
const newGroup = ref<PreviewDNSGroup>({ id: '', name: '', servers: [], timeout: 3, fallback: 'strict' })
let groupSaved = false
async function showGroupPopover(event: Event) {
  await openGroupForm(event.currentTarget as HTMLElement, '')
}
function dnsSearchKeydown(event: KeyboardEvent) {
  if (event.key === 'Tab' && !event.shiftKey && !event.isComposing) {
    event.preventDefault()
    event.stopPropagation()
    document.getElementById('dns-create-option')?.focus()
  }
}
function dnsCreateKeydown(event: KeyboardEvent) {
  if (event.key === 'Tab' && event.shiftKey) {
    event.preventDefault()
    document.getElementById('policy-dns-search')?.focus()
  } else if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    dnsSelect.value?.hide()
    document.getElementById('policy-dns')?.focus()
  }
}
async function createFromDNSSearch() {
  const name = dnsSearch.value.trim()
  const anchor = document.getElementById('policy-dns')
  await dnsSelect.value?.hideAndWait()
  if (anchor) await openGroupForm(anchor, name)
}
async function openGroupForm(anchor: HTMLElement, name: string) {
  newGroup.value = { id: crypto.randomUUID(), name, servers: [], timeout: 3, fallback: 'strict' }
  groupSaved = false
  await nextTick()
  groupPopover.value?.showAt(anchor)
}
async function saveQuickGroup(group: PreviewDNSGroup) {
  emit('groupCreated', group)
  await nextTick()
  dns.value = group.id
  groupSaved = true
  groupPopover.value?.hide()
}
function groupPopoverHidden() {
  groupPopoverOpen.value = false
  if (groupSaved) {
    nextTick(() => document.getElementById('policy-dns')?.focus({ preventScroll: true }))
    groupSaved = false
  }
}
const previewView = ref<'summary' | 'config'>('summary')
function changePreviewView(event: KeyboardEvent) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  previewView.value = event.key === 'Home' ? 'summary' : event.key === 'End' ? 'config' : previewView.value === 'summary' ? 'config' : 'summary'
  nextTick(() => document.getElementById(`policy-tab-${previewView.value}`)?.focus())
}
const open = ref(false)
const conditions = ref<Condition[]>([{ id: 1, kind: 'domain_suffix', value: 'xx.com' }])
let sequence = 1
const mode = ref('or')
const outbound = ref('Auto-select')
const dns = ref('unchanged')
const fallback = ref('strict')
watch(dns, id => { const group = props.groups.find(g => g.id === id); if (group) fallback.value = group.fallback })
watch(profiles, options => { if (!options.some(p => p.value === dns.value)) dns.value = 'unchanged' })
async function useGroup(id: string) {
  dns.value = id
  if (!open.value) await start()
  else await nextTick()
  editor.value?.scrollIntoView({ block: 'start', behavior: 'instant' })
}
async function returnToDNS() {
  await nextTick()
  const field = document.getElementById('policy-dns')
  field?.scrollIntoView({ block: 'center', behavior: 'instant' })
  field?.focus({ preventScroll: true })
}
defineExpose({ useGroup, returnToDNS })

const attempted = ref(false)
const saving = ref(false)
const feedback = ref('')
const editor = ref<HTMLElement>()
const trigger = ref<HTMLElement>()
const saved = ref<{ id: number; match: string; outbound: string; dns: string; config: string }[]>([])
let timer: ReturnType<typeof setTimeout> | undefined
const hasIP = computed(() => conditions.value.some(c => c.kind === 'ip_cidr'))
const effectiveDNS = computed(() => hasIP.value ? 'unchanged' : dns.value)
const profile = computed(() => profiles.value.find(p => p.value === effectiveDNS.value) ?? profiles.value[profiles.value.length - 1]!)
const matchText = computed(() => conditions.value.map(c => {
  const text = `${kinds.find(k => k.value === c.kind)?.label}: ${matchValues(c.value).join(', ') || '…'}`
  return conditions.value.length > 1 ? `(${text})` : text
}).join(mode.value === 'or' ? ' OR ' : ' AND '))
const responsePrefix = ref(createDNSResponsePrefix())
const configuration = computed(() => {
  if (conditions.value.some(conditionError)) return null
  return buildTrafficPolicyConfig({ conditions: conditions.value, mode: mode.value as 'and' | 'or', outbound: outbound.value,
    servers: profile.value.servers, timeout: props.groups.find(g => g.id === effectiveDNS.value)?.timeout ?? 5,
    fallback: fallback.value as 'strict' | 'general', tagPrefix: responsePrefix.value })
})
const configurationJSON = computed(() => configuration.value ? JSON.stringify(configuration.value, null, 2) : '')
function conditionError(c: Condition) {
  const values = matchValues(c.value)
  if (!values.length) return t('lab.enterAtLeastOneMatchValue')
  for (const value of values) {
  if (c.kind === 'domain' || c.kind === 'domain_suffix') {
    if (!/^(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/i.test(value)) return t('lab.enterADomainSuchAsExampleComWithoutAUrl')
  }
  if (c.kind === 'ip_cidr' && !/^\d{1,3}(?:\.\d{1,3}){3}\/\d{1,2}$/.test(value)) return t('lab.useAnIpv4RangeForThisPreviewSuchAs192')
  if (c.kind === 'ip_cidr') {
    const [ip, prefix] = value.split('/')
    if (Number(prefix) > 32 || ip!.split('.').some(n => Number(n) > 255)) return t('lab.enterAValidIpv4Range')
  }
  }
  return ''
}
async function start() {
  responsePrefix.value = createDNSResponsePrefix()
  open.value = true
  attempted.value = false
  feedback.value = ''
  await nextTick()
  editor.value?.querySelector<HTMLTextAreaElement>('textarea')?.focus({ preventScroll: true })
}
async function close() {
  groupPopover.value?.hide()
  clearTimeout(timer)
  saving.value = false
  open.value = false
  await nextTick()
  trigger.value?.querySelector<HTMLButtonElement>('button')?.focus({ preventScroll: true })
}
function save() {
  attempted.value = true
  if (conditions.value.some(conditionError)) return
  saving.value = true
  timer = setTimeout(() => {
    saved.value.push({ id: Date.now(), match: matchText.value, outbound: outbound.value, dns: profile.value.label, config: configurationJSON.value })
    feedback.value = t('lab.trafficPolicyAddedToThisPreviewYourConfigurationHasNot')
    close()
  }, 650)
}
onBeforeUnmount(() => clearTimeout(timer))
onDeactivated(() => groupPopover.value?.hide())

</script>

<template>
  <section id="traffic-policy" aria-labelledby="traffic-policy-title" class="scroll-mt-4 space-y-4">
    <div class="flex flex-wrap items-end justify-between gap-3">
      <div><p class="text-xs font-medium uppercase tracking-wider text-text-secondary">{{ t('lab.interactionPreview') }}</p><h2 id="traffic-policy-title" class="mt-1 text-xl font-semibold">{{ t('lab.trafficPolicies') }}</h2><p class="mt-1 text-sm text-text-secondary">{{ t('lab.oneMatchATrafficDestinationAndADnsDecision') }}</p></div>
      <span class="text-xs text-text-secondary">{{ t('lab.sampleDataChangesStayInThisPage') }}</span>
    </div>
    <Card padding="none" class="overflow-hidden">
      <div class="flex flex-wrap items-center justify-between gap-3 p-4">
        <div><h3 class="font-semibold">{{ t('lab.yourPolicies') }}</h3><p class="mt-1 text-xs text-text-secondary">{{ t('lab.decideWhereTrafficGoesAndHowItsNamesAreResolved') }}</p></div>
        <div ref="trigger"><Button :disabled="open" :aria-expanded="open" aria-controls="traffic-policy-editor" @click="start"><PlusIcon class="h-4 w-4" aria-hidden="true" />{{ t('lab.addTrafficPolicy') }}</Button></div>
      </div>
      <Transition name="policy-expand">
        <div v-if="open" id="traffic-policy-editor" ref="editor" class="policy-expansion" @keydown.esc.stop="close">
          <div class="min-h-0 overflow-hidden">
            <form class="border-t border-border" @submit.prevent="save">
              <fieldset :disabled="saving" class="grid min-w-0 lg:grid-cols-[minmax(0,1.35fr)_minmax(0,1fr)]">
                <div class="min-w-0 space-y-6 p-4 sm:p-5">
                  <div>
                    <div class="mb-3 flex flex-wrap items-center justify-between gap-2"><h4 class="text-sm font-semibold">{{ t('lab.1MatchTraffic') }}</h4><Select v-model="mode" :aria-label="t('lab.conditionLogic')" :options="[{label: t('lab.matchAnyRowOr'), value: 'or'}, {label: t('lab.matchAllRowsAnd'), value: 'and'}]" option-label="label" option-value="value" /></div>
                    <div v-for="(condition, index) in conditions" :key="condition.id" class="mb-3 space-y-2">
                      <p v-if="index" class="text-xs font-semibold text-text-secondary">{{ mode === 'or' ? 'OR' : 'AND' }}</p>
                      <div class="flex items-end gap-2"><div class="min-w-0 flex-1"><label :for="`policy-kind-${condition.id}`" class="mb-1 block text-xs text-text-secondary">{{ t('lab.condition') }} {{ index + 1 }}</label><Select v-model="condition.kind" :input-id="`policy-kind-${condition.id}`" :aria-label="t('lab.numberedCondition', { n: index + 1 })" :options="kinds" option-label="label" option-value="value" class="w-full" @update:model-value="condition.value = ''" /></div><IconButton v-if="conditions.length > 1" action="delete" :label="t('lab.removeCondition', { n: index + 1 })" @click="conditions.splice(index, 1)" /></div>
                      <Textarea v-model="condition.value" :rows="3" :label="t('lab.matchValues', { n: index + 1 })" :placeholder="condition.kind === 'ip_cidr' ? '192.0.2.0/24' : condition.kind === 'rule_set' ? 'sea-rulesets-proxy' : 'example.com'" :error="attempted ? conditionError(condition) : ''" :hint="t('lab.oneValuePerLineOrSeparateWithCommasMatchesAny')" />
                    </div>
                    <Button variant="ghost" size="sm" @click="conditions.push({ id: ++sequence, kind: 'domain_suffix', value: '' })"><PlusIcon class="h-4 w-4" aria-hidden="true" />{{ t('lab.addCondition') }}</Button>
                  </div>
                  <div><label for="policy-outbound" class="mb-2 block text-sm font-semibold">{{ t('lab.2RouteTrafficThrough') }}</label><Select v-model="outbound" input-id="policy-outbound" :aria-label="t('lab.trafficOutbound')" :options="['Auto-select', 'Direct', 'Claude', 'Japan relay']" filter class="w-full" /></div>
                  <div class="space-y-3"><label for="policy-dns" class="block text-sm font-semibold">{{ t('lab.3ResolveDnsWith') }}</label><Select ref="dnsSelect" :model-value="effectiveDNS" input-id="policy-dns" :aria-label="t('lab.3ResolveDnsWith')" :options="profiles" option-label="label" option-value="value" :disabled="hasIP" filter :filter-placeholder="t('lab.searchDnsServersOrGroups')" :filter-fields="['label', 'servers']" reset-filter-on-hide filter-input-id="policy-dns-search" :filter-input-label="t('lab.searchDnsServersOrGroups')" :filter-keydown="dnsSearchKeydown" class="w-full" @filter="dnsSearch = $event.value" @hide="dnsSearch = ''" @update:model-value="dns = $event">
                      <template #emptyfilter><span class="text-sm text-text-secondary">{{ t('lab.noMatchingDnsServersOrGroups') }}</span></template>
                      <template #footer>
                        <div class="border-t border-border p-2">
                          <button id="dns-create-option" type="button" @keydown="dnsCreateKeydown" class="flex min-h-10 w-full items-center gap-2 rounded-control px-3 py-2 text-left text-sm font-medium text-primary hover:bg-bg-tertiary focus-visible:outline-2 focus-visible:outline-primary" @click="createFromDNSSearch">
                            <PlusIcon class="h-4 w-4 shrink-0" aria-hidden="true" /><span class="min-w-0 break-words">{{ dnsSearch.trim() ? t('lab.createNamedGroup', { name: dnsSearch.trim() }) : t('lab.createDnsGroup2') }}</span>
                          </button>
                        </div>
                      </template>
                    </Select>
                    <div class="flex flex-wrap gap-2"><Button variant="ghost" size="sm" aria-controls="policy-create-dns-group" :aria-expanded="groupPopoverOpen" @click="showGroupPopover"><PlusIcon class="h-4 w-4" aria-hidden="true" />{{ t('lab.createDnsGroup') }}</Button><Button variant="ghost" size="sm" @click="emit('manageGroups')">{{ t('lab.manageDnsGroups') }}</Button></div>
                    <p v-if="!groups.length" class="text-xs text-text-secondary">{{ t('lab.adoptOrCreateAResolverGroupAboveToUseIt') }}</p>
                    <p v-if="hasIP" class="rounded-control border border-border bg-bg-tertiary p-3 text-sm" role="note">{{ t('lab.ipAddressesAreKnownAfterResolutionThisPreviewKeepsDns') }}</p>
                    <template v-else-if="effectiveDNS !== 'unchanged'"><p class="text-xs text-text-secondary">{{ profile.servers.join(' + ') }} · {{ profile.timeout }} {{ t('lab.perQuery') }}</p><label for="policy-fallback" class="block text-xs font-medium">{{ t('lab.ifDnsResolutionFails') }}</label><Select v-model="fallback" input-id="policy-fallback" :aria-label="t('lab.ifDnsResolutionFails')" :options="[{label: t('lab.stopWithinThisDnsProfile'), value: 'strict'}, {label: t('lab.allowExistingDnsFallback'), value: 'general'}]" option-label="label" option-value="value" class="w-full" /><p v-if="fallback === 'general'" class="text-sm text-text-secondary" role="note">{{ t('lab.existingFallbackMaySendThisDomainToResolversOutsideThe') }}</p></template>
                  </div>
                </div>
                <aside class="min-w-0 border-t border-border bg-bg-tertiary p-4 sm:p-5 lg:border-l lg:border-t-0" :aria-label="t('lab.policyOutcome')" :aria-busy="saving">
                  <div role="tablist" :aria-label="t('lab.policyPreviewViews')" class="flex gap-1 border-b border-border pb-3" @keydown="changePreviewView">
                    <button v-for="view in [{id:'summary',label:t('lab.policyPreview')},{id:'config',label:t('lab.configJson')}]" :id="`policy-tab-${view.id}`" :key="view.id" type="button" role="tab" :aria-selected="previewView === view.id" :aria-controls="`policy-panel-${view.id}`" :tabindex="previewView === view.id ? 0 : -1" class="rounded-control px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-primary" :class="previewView === view.id ? 'bg-bg-secondary text-text-primary' : 'text-text-secondary hover:bg-bg-secondary'" @click="previewView = view.id as 'summary' | 'config'">{{ view.label }}</button>
                  </div>
                  <div v-show="previewView === 'summary'" id="policy-panel-summary" role="tabpanel" aria-labelledby="policy-tab-summary">
                  <div class="relative mt-5 min-h-64">
                    <div :class="saving ? 'invisible' : ''" class="space-y-5">
                      <p class="break-words text-sm font-medium">{{ matchText }}</p>
                      <div class="flex gap-3"><ArrowsRightLeftIcon class="mt-1 h-5 w-5 shrink-0 text-primary" aria-hidden="true" /><div><p class="text-xs text-text-secondary">{{ t('lab.trafficOutbound') }}</p><p class="mt-1 font-semibold">{{ outbound }}</p></div></div>
                      <div class="flex gap-3"><GlobeAltIcon class="mt-1 h-5 w-5 shrink-0 text-primary" aria-hidden="true" /><div class="min-w-0"><p class="text-xs text-text-secondary">{{ t('lab.dnsResolution') }}</p><p class="mt-1 font-semibold">{{ profile.servers.length > 1 ? t('lab.firstSuccessfulResponse') : profile.label }}</p><p class="mt-1 break-words text-sm">{{ profile.servers.join(' + ') }}</p><p class="mt-1 text-xs text-text-secondary">{{ profile.path }}</p></div></div>
                      <p v-if="effectiveDNS !== 'unchanged'" class="text-sm text-text-secondary">{{ fallback === 'strict' ? t('lab.stopIfThisProfileCannotResolveTheNameNoGeneral') : t('lab.continueThroughExistingDnsRulesIfThisProfileFails') }}</p>
                    </div>
                    <div v-if="saving" class="absolute inset-0 space-y-5" role="status"><Skeleton class="h-5 w-4/5" /><Skeleton class="h-14 w-full" /><Skeleton class="h-20 w-full" /><p class="text-xs text-text-secondary">{{ t('lab.addingToPreview') }}</p></div>
                  </div>
                  <div class="mt-5 border-t border-border pt-4 text-xs leading-relaxed text-text-secondary"><p class="font-semibold text-text-primary">{{ t('lab.proposedConfiguration') }}</p><p class="mt-2">{{ t('lab.generateANewRouteAndWhenSelectedADnsRule') }}</p><p class="mt-2">{{ t('lab.insertBeforeApplicableCatchAllRulesPreservingDnsInterceptionAnd') }}</p></div>
                  </div>
              <section v-show="previewView === 'config'" id="policy-panel-config" role="tabpanel" aria-labelledby="policy-tab-config" class="mt-5 min-w-0">
                <div class="flex items-center justify-between gap-3"><h4 class="font-semibold">{{ t('lab.configJsonProposedAdditions') }}</h4><CopyIcon v-if="configurationJSON" :text="configurationJSON" :label="t('lab.copyProposedConfiguration')" /></div>
                <p class="mt-2 text-xs text-text-secondary">{{ t('lab.theseAreNewRulesNotAReplacementForTheExisting') }}</p>
                <p v-if="!configurationJSON" class="mt-3 text-sm text-text-secondary" role="status">{{ t('lab.completeValidMatchConditionsToSeeTheGeneratedConfiguration') }}</p>
                <pre v-else data-config-preview tabindex="0" :aria-label="t('lab.proposedConfigurationJson')" class="mt-3 max-h-[32rem] overflow-auto rounded-control border border-border bg-bg-tertiary p-3 text-xs leading-relaxed"><code>{{ configurationJSON }}</code></pre>
                <p v-if="configurationJSON && effectiveDNS === 'unchanged'" class="mt-2 text-xs text-text-secondary">{{ t('lab.dnsIsUnchangedOnlyRouteRulesAdditionsAreShown') }}</p>
                <p v-else-if="configurationJSON" class="mt-2 text-xs text-text-secondary">{{ fallback === 'strict' ? t('lab.theFinalMatchingRejectRulePreventsGeneralFallbackIfNo') : t('lab.noRejectGuardIsAddedUnmatchedResponsesFallThroughTo') }}</p>
              </section>
                </aside>
              </fieldset>

              <div class="flex flex-wrap items-center justify-between gap-3 border-t border-border p-4"><span class="text-xs text-text-secondary">{{ t('lab.previewOnlyNoConfigurationIsWritten') }}</span><div class="flex gap-2"><Button variant="secondary" @click="close">{{ t('lab.cancel') }}</Button><Button type="submit" :disabled="saving">{{ saving ? t('lab.adding') : t('lab.addToPreview') }}<ArrowRightIcon class="h-4 w-4" aria-hidden="true" /></Button></div></div>
            </form>
          </div>
        </div>
      </Transition>
      <div v-if="!saved.length && !open" class="border-t border-border px-4 py-8 text-center"><ArrowsRightLeftIcon class="mx-auto h-6 w-6 text-text-secondary" aria-hidden="true" /><p class="mt-3 text-sm font-medium">{{ t('lab.noSamplePoliciesYet') }}</p><p class="mt-1 text-xs text-text-secondary">{{ t('lab.addOneToTryTheCombinedTrafficAndDnsWorkflow') }}</p></div>
      <div v-for="policy in saved" :key="policy.id" class="flex items-center gap-3 border-t border-border p-4"><CheckCircleIcon class="h-5 w-5 shrink-0 text-success" aria-hidden="true" /><div class="min-w-0 flex-1"><p class="break-words text-sm font-medium">{{ policy.match }}</p><p class="mt-1 text-xs text-text-secondary">{{ policy.outbound }} → {{ policy.dns }}</p><details class="mt-2 text-xs"><summary class="cursor-pointer text-primary">{{ t('lab.viewSavedConfiguration') }}</summary><pre class="mt-2 max-h-80 overflow-auto rounded-control border border-border bg-bg-tertiary p-3">{{ policy.config }}</pre></details></div><IconButton action="delete" :label="t('lab.removeSamplePolicy')" @click="saved = saved.filter(p => p.id !== policy.id); feedback = t('lab.samplePolicyRemoved')" /></div>
    </Card>
    <Popover id="policy-create-dns-group" ref="groupPopover" :aria-label="t('lab.createDnsGroup')" panel-class="w-[28rem]" @show="groupPopoverOpen = true" @hide="groupPopoverHidden">
      <DNSGroupForm :key="newGroup.id" :initial="newGroup" :groups="groups" :servers="previewDNSServers" :policy-context="{ conditions, mode: mode as 'and' | 'or', outbound, tagPrefix: responsePrefix }" :policy-match-valid="!hasIP && !conditions.some(conditionError)" compact @save="saveQuickGroup" @cancel="groupPopover?.hide()" />
    </Popover>
    <p class="min-h-5 text-sm text-text-secondary" role="status">{{ feedback }}</p>
  </section>
</template>

<style scoped>
.policy-expansion { display: grid; grid-template-rows: 1fr; opacity: 1; }
.policy-expand-enter-active { transition: grid-template-rows 240ms ease-out, opacity 240ms ease-out; }
.policy-expand-leave-active { transition: grid-template-rows 240ms ease-in, opacity 240ms ease-in; }
.policy-expand-enter-from, .policy-expand-leave-to { grid-template-rows: 0fr; opacity: 0; }
@media (prefers-reduced-motion: reduce) { .policy-expand-enter-active, .policy-expand-leave-active { transition: none; } }
</style>
