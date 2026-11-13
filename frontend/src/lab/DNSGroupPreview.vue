<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

import { computed, nextTick, ref } from 'vue'
import { PlusIcon, GlobeAltIcon, ArrowRightIcon } from '@heroicons/vue/24/outline'
import Card from '../components/Card.vue'
import Button from '../components/Button.vue'
import DNSGroupForm from './DNSGroupForm.vue'
import IconButton from '../components/IconButton.vue'
import { previewDNSServers, type PreviewDNSGroup } from './dnsGroupPreview'
const groups = defineModel<PreviewDNSGroup[]>({ required: true })
const emit = defineEmits<{ use: [id: string]; saved: [id: string]; cancel: [] }>()
const candidates = [
  { id: 'proxy-domain', servers: ['GoogleDoT', 'dns_google'], timeout: 5, get rules() { return t('lab.2Evaluate2RespondRules') }, get match() { return t('lab.domainSuffixes9Domains') }, get path() { return t('lab.bothResolversViaAutoSelect') } },
  { id: 'direct-domain', servers: ['dns_router', 'aliyun', 'tencent'], timeout: 3, get rules() { return t('lab.3Evaluate3RespondRules') }, get match() { return t('lab.domainSuffixes6Domains') }, get path() { return t('lab.defaultResolverPathsReviewBeforeAdoption') } },
]
const discovered = computed(() => candidates.filter(c => !groups.value.some(g => g.source === c.id)))
const servers = previewDNSServers
const editing = ref<string | null>(null)
const draft = ref<PreviewDNSGroup>({ id: '', name: '', servers: [], timeout: 3, fallback: 'strict' })
const message = ref('')
const review = ref<string | null>(null)
let trigger: HTMLElement | null = null
const root = ref<HTMLElement>()

async function start(key: string, event: Event | undefined, existing?: PreviewDNSGroup) {
  trigger = event?.currentTarget as HTMLElement | null
  const candidate = candidates.find(c => c.id === key)
  draft.value = existing ? { ...existing, servers: [...existing.servers] } : { id: crypto.randomUUID(), name: '', servers: candidate ? [...candidate.servers] : [], timeout: candidate?.timeout ?? 3, fallback: candidate ? 'general' : 'strict', source: candidate?.id }
  editing.value = key
  message.value = ''
  await nextTick()
  root.value?.querySelector<HTMLInputElement>('[data-group-editor] input')?.focus({ preventScroll: true })
}
async function cancel() { editing.value = null; emit('cancel'); await nextTick(); trigger?.isConnected && trigger.focus({ preventScroll: true }) }
function save(group: PreviewDNSGroup) {
  const index = groups.value.findIndex(g => g.id === group.id)
  groups.value = index < 0 ? [...groups.value, group] : groups.value.map(g => g.id === group.id ? group : g)
  editing.value = null
  message.value = t(index < 0 ? 'lab.groupAdded' : 'lab.groupUpdated', { name: group.name })
  emit('saved', group.id)
  nextTick(() => root.value?.querySelector<HTMLButtonElement>(`[data-use-group="${group.id}"]`)?.focus({ preventScroll: true }))
}
function resolve(group: PreviewDNSGroup, adopt: boolean) {
  groups.value = groups.value.map(g => g.id === group.id ? { ...g, timeout: adopt ? (g.timeout === 3 ? 5 : 3) : g.timeout, drift: false } : g)
  review.value = null
  message.value = adopt ? t('lab.configChangeAdoptedInThisPreview') : t('lab.savedSettingsRestoredInThisPreviewTheRealConfigIs')
}

</script>
<template>
  <section id="dns-groups" ref="root" class="scroll-mt-4 space-y-4" aria-labelledby="dns-groups-title">
    <div class="flex flex-wrap items-end justify-between gap-3"><div><p class="text-xs uppercase tracking-wider text-text-secondary">{{ t('lab.interactionPreviewDnsServers') }}</p><h2 id="dns-groups-title" class="mt-1 text-xl font-semibold">{{ t('lab.resolverGroups') }}</h2><p class="mt-1 text-sm text-text-secondary">{{ t('lab.defineHowNamesResolveReuseAGroupInAnyTraffic') }}</p></div><Button :disabled="editing !== null" @click="start('new', $event)"><PlusIcon class="h-4 w-4" aria-hidden="true" />{{ t('lab.createDnsGroup') }}</Button></div>
    <p class="text-xs text-text-secondary">{{ t('lab.sampleDiscoveryOnlyChangesResetWhenYouLeaveTheLab') }}</p>
    <div id="group-editor-new" />
    <Card padding="none">
      <div class="border-b border-border p-4"><h3 class="font-semibold">{{ t('lab.managedGroups') }} <span class="ml-1 font-normal text-text-secondary">{{ groups.length }}</span></h3><p class="mt-1 text-xs text-text-secondary">{{ t('lab.namedGroupsAvailableToTheTrafficPolicyEditor') }}</p></div>
      <p v-if="!groups.length" class="p-5 text-sm text-text-secondary">{{ t('lab.noManagedGroupsYetAdoptADiscoveredGroupBelowOr') }}</p>
      <div v-for="group in groups" :key="group.id" class="border-b border-border p-4 last:border-b-0">
        <div class="flex flex-wrap items-start justify-between gap-3"><div class="min-w-0"><h4 class="break-words font-semibold">{{ group.name }}</h4><p class="mt-1 text-sm text-text-secondary">{{ group.servers.join(' + ') }} · {{ group.timeout }}{{ t('lab.sFirstNoerrorResponse') }}</p><p class="mt-2 flex items-center gap-2 text-xs"><span class="h-1.5 w-1.5 rounded-full" :class="group.drift ? 'bg-warning' : 'bg-success'" aria-hidden="true" />{{ group.drift ? t('lab.configDiffersFromSavedSettings') : t('lab.inSyncSampleConfiguration') }}</p></div><div class="flex items-center gap-2"><IconButton action="edit" :label="t('lab.editNamed', { name: group.name })" :disabled="editing !== null || group.drift" @click="start(group.id, $event, group)" /><Button size="sm" variant="secondary" :data-use-group="group.id" :disabled="group.drift" @click="emit('use', group.id)">{{ t('lab.useInPolicy') }}<ArrowRightIcon class="h-4 w-4" aria-hidden="true" /></Button></div></div>
        <p class="mt-2 text-xs text-text-secondary">{{ group.fallback === 'strict' ? t('lab.stopWithinThisProfileIfResolutionFails') : t('lab.existingFallbackIsAllowedQueriesMayReachOtherResolvers') }}</p>
        <div v-if="group.drift" class="mt-3"><Button size="sm" variant="secondary" :aria-expanded="review === group.id" @click="review = review === group.id ? null : group.id">{{ t('lab.reviewDifference') }}</Button><div v-if="review === group.id" class="mt-3 border-l-2 border-primary pl-4"><p class="text-sm">{{ t('lab.theTimeoutWasChangedOutsideTheGroupEditor') }}</p><dl class="my-3 grid grid-cols-2 gap-2 text-sm"><div><dt class="text-text-secondary">{{ t('lab.savedGroup') }}</dt><dd>{{ group.timeout }} {{ t('lab.seconds') }}</dd></div><div><dt class="text-text-secondary">{{ t('lab.sampleConfig') }}</dt><dd>{{ group.timeout === 3 ? 5 : 3 }} {{ t('lab.seconds') }}</dd></div></dl><p class="mb-3 text-xs text-text-secondary">{{ t('lab.adoptKeepsTheConfigSTimeoutRestorePutsTheSaved') }}</p><div class="flex flex-wrap gap-2"><Button size="sm" @click="resolve(group, true)">{{ t('lab.adoptConfigChanges') }}</Button><Button size="sm" variant="secondary" @click="resolve(group, false)">{{ t('lab.restoreSavedSettings') }}</Button><Button size="sm" variant="ghost" @click="review = null">{{ t('lab.later') }}</Button></div></div></div>
        <Button v-else size="sm" variant="ghost" class="mt-2" :disabled="editing !== null" @click="groups = groups.map(g => g.id === group.id ? {...g, drift: true} : g)">{{ t('lab.simulateManualConfigChange') }}</Button>
        <div :id="`group-editor-${group.id}`" />
      </div>
    </Card>
    <Card padding="none">
      <div class="border-b border-border p-4"><h3 class="font-semibold">{{ t('lab.discoveredInConfiguration') }}</h3><p class="mt-1 text-xs text-text-secondary">{{ t('lab.compatibleRuleBlocksNotYetNamedOrManagedAdoptionLeaves') }}</p></div>
      <p v-if="!discovered.length" class="p-5 text-sm text-text-secondary">{{ t('lab.allSampleGroupsHaveBeenAdopted') }}</p>
      <div v-for="candidate in discovered" :key="candidate.id"><div class="flex flex-wrap items-center justify-between gap-3 border-b border-border p-4 last:border-b-0"><div class="min-w-0"><p class="flex items-center gap-2 font-medium"><GlobeAltIcon class="h-4 w-4 shrink-0" aria-hidden="true" />{{ candidate.servers.join(' + ') }}</p><p class="mt-2 text-xs text-text-secondary">{{ candidate.match }} · {{ candidate.rules }} · {{ candidate.timeout }}s</p><p class="mt-1 text-xs text-text-secondary">{{ candidate.path }}</p></div><Button variant="secondary" size="sm" :disabled="editing !== null" @click="start(candidate.id, $event)">{{ t('lab.adoptGroup') }}</Button></div><div :id="`group-editor-${candidate.id}`" /></div>
    </Card>
    <Teleport v-if="editing" defer :to="`#group-editor-${editing}`">
    <Transition name="group-editor" appear>
      <div v-if="editing" data-group-editor class="scroll-mt-4 border-t border-border bg-bg-tertiary p-4 sm:p-5" @keydown.esc.stop="cancel">
        <DNSGroupForm :key="draft.id" :initial="draft" :groups="groups" :servers="servers" :locked="!!draft.source && !groups.some(g => g.id === draft.id)" :mode="editing === 'new' ? 'create' : 'edit'" @save="save" @cancel="cancel" />
      </div>
    </Transition>
    </Teleport>
    <p role="status" class="min-h-5 text-sm text-text-secondary">{{ message }}</p>
  </section>
</template>
<style scoped>
.group-editor-enter-active { transition: opacity 180ms ease-out, transform 180ms ease-out; }
.group-editor-leave-active { transition: opacity 180ms ease-in, transform 180ms ease-in; }
.group-editor-enter-from, .group-editor-leave-to { opacity: 0; transform: translateY(-4px); }
@media (prefers-reduced-motion: reduce) { .group-editor-enter-active, .group-editor-leave-active { transition: none; } }
</style>
