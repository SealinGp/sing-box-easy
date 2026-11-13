<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

import { createDNSResponsePrefix } from './dnsResponseTag'
import { computed, nextTick, ref, useId } from 'vue'
import Input from '../components/Input.vue'
import Button from '../components/Button.vue'
import Select from '../volt/Select.vue'
import MultiSelect from '../volt/MultiSelect.vue'
import CopyIcon from '../components/CopyIcon.vue'
import { buildTrafficPolicyConfig, type TrafficPolicyConfigSpec } from './trafficPolicyConfig'
import type { PreviewDNSGroup } from './dnsGroupPreview'
// Shared form only: callers own presentation, persistence and selection.
const props = withDefaults(defineProps<{
  live?: boolean
  busy?: boolean
  initial: PreviewDNSGroup
  groups: PreviewDNSGroup[]
  servers: string[]
  locked?: boolean
  compact?: boolean
  mode?: 'create' | 'edit'
  policyContext?: Pick<TrafficPolicyConfigSpec, 'conditions' | 'mode' | 'outbound' | 'tagPrefix'>
  policyMatchValid?: boolean
}>(), { locked: false, compact: false, mode: 'create', policyMatchValid: true })
const emit = defineEmits<{ save: [group: PreviewDNSGroup]; cancel: [] }>()
const draft = ref({ ...props.initial, servers: [...props.initial.servers] })
const serverInputId = useId()
const attempted = ref(false)
const activeView = ref<'settings' | 'config'>('settings')
const tabsId = useId()
const examplePrefix = createDNSResponsePrefix()
const configJSON = computed(() => {
  if (error.value || !props.policyMatchValid) return ''
  const context = props.policyContext ?? { conditions: [{kind:'domain_suffix', value:'example.com'}], mode:'or' as const, outbound:'example-outbound', tagPrefix:examplePrefix }
  const config = buildTrafficPolicyConfig({ ...context, servers:draft.value.servers, timeout:Number(draft.value.timeout), fallback:draft.value.fallback })
  return config.dns ? JSON.stringify({dns:config.dns}, null, 2) : ''
})
function switchView(event: KeyboardEvent) {
  if (!['ArrowLeft','ArrowRight','Home','End'].includes(event.key)) return
  event.preventDefault()
  activeView.value = event.key === 'Home' ? 'settings' : event.key === 'End' ? 'config' : activeView.value === 'settings' ? 'config' : 'settings'
  nextTick(() => document.getElementById(`${tabsId}-${activeView.value}`)?.focus())
}
const error = computed(() => !draft.value.name.trim() ? t('lab.giveThisGroupAName') : props.groups.some(g => g.id !== draft.value.id && g.name.toLowerCase() === draft.value.name.trim().toLowerCase()) ? t('lab.chooseAUniqueGroupName') : draft.value.servers.length < 2 ? t('lab.chooseAtLeastTwoResolversForAParallelGroup') : !Number.isInteger(Number(draft.value.timeout)) || Number(draft.value.timeout) < 1 || Number(draft.value.timeout) > 30 ? t('lab.setATimeoutBetween1And30Seconds') : '')
function save() {
  attempted.value = true
  if (error.value) { activeView.value = 'settings'; return }
  emit('save', { ...draft.value, name: draft.value.name.trim(), timeout: Number(draft.value.timeout), servers: [...draft.value.servers] })
}

</script>
<template>
        <form @submit.prevent="save">
          <div class="mb-4"><h3 class="font-semibold">{{ locked ? t('lab.nameThisDiscoveredGroup') : mode === 'create' ? t('lab.createDnsGroup') : t('lab.editDnsGroup') }}</h3><p class="mt-1 text-xs text-text-secondary">{{ locked ? t('lab.keepTheDiscoveredSettingsIntactYouCanEditThemAfter') : t('lab.groupSettingsApplyToEveryTrafficPolicyThatUsesThis') }}</p></div>
          <div role="tablist" :aria-label="t('lab.dnsGroupViews')" class="mb-4 flex gap-1 border-b border-border pb-3" @keydown="switchView">
            <button v-for="view in [{id:'settings',label:t('lab.groupSettings')},{id:'config',label:t('lab.configJson')}]" :id="`${tabsId}-${view.id}`" :key="view.id" type="button" role="tab" :aria-selected="activeView === view.id" :aria-controls="`${tabsId}-${view.id}-panel`" :tabindex="activeView === view.id ? 0 : -1" class="rounded-control px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-primary" :class="activeView === view.id ? 'bg-bg-tertiary text-text-primary' : 'text-text-secondary'" @click="activeView = view.id as 'settings' | 'config'">{{ view.label }}</button>
          </div>
          <div v-show="activeView === 'settings'" :id="`${tabsId}-settings-panel`" role="tabpanel" :aria-labelledby="`${tabsId}-settings`">
          <div :class="['grid gap-5', compact ? '' : 'md:grid-cols-2']"><div class="space-y-4"><Input v-model="draft.name" :label="t('lab.groupName')" :placeholder="t('lab.eGOverseas')" /><div><label :for="serverInputId" class="mb-2 block text-sm font-medium">{{ t('lab.dnsServers') }}</label><MultiSelect v-model="draft.servers" :input-id="serverInputId" :aria-label="t('lab.dnsServers')" :options="servers" :disabled="locked" filter :filter-placeholder="t('lab.searchDnsServers')" :empty-filter-message="t('lab.noMatchingDnsServers')" display="chip" :placeholder="t('lab.selectDnsServers')" :show-toggle-all="false" reset-filter-on-hide class="w-full" /><p class="mt-2 text-xs text-text-secondary">{{ locked ? t('lab.discoveredServersArePreservedDuringAdoption') : t('lab.chooseAtLeastTwoServersSearchAndSelectMultipleResolvers') }}</p></div></div><div class="space-y-4"><p class="text-sm"><strong>{{ t('lab.parallelResolution') }}</strong><br /><span class="text-text-secondary">{{ t('lab.firstResponseWithNoerrorWinsThisCanIncludeAnEmpty') }}</span></p><Input v-model="draft.timeout" type="number" :label="t('lab.timeoutSeconds')" :min="1" :max="30" :disabled="locked" /><div><label class="mb-1 block text-sm font-medium">{{ t('lab.ifResolutionFails') }}</label><Select v-model="draft.fallback" :aria-label="t('lab.groupFailureBehavior')" :disabled="locked" :options="[{label:t('lab.stopWithinThisProfile'),value:'strict'},{label:t('lab.allowExistingDnsFallback'),value:'general'}]" option-label="label" option-value="value" class="w-full" /></div></div></div>
          </div>
          <section v-show="activeView === 'config'" :id="`${tabsId}-config-panel`" role="tabpanel" :aria-labelledby="`${tabsId}-config`" class="min-w-0 space-y-3">
            <div class="flex items-start justify-between gap-3"><h4 class="text-sm font-semibold">{{ t('lab.configJsonDnsRulePreview') }}</h4><CopyIcon v-if="configJSON" :text="configJSON" :label="t('lab.copyDnsGroupConfiguration')" /></div>
            <p class="text-xs leading-relaxed text-text-secondary">{{ locked ? t('lab.adoptionNamesTheExistingGroupWithoutRewritingItsRulesThe') : mode === 'edit' ? t('lab.thisPreviewsTheRevisedDnsBehaviorChangesToExistingPolicies') : t('lab.creatingAReusableGroupAloneAddsNoSingBoxRules') }}</p>
            <p class="text-xs font-medium">{{ policyContext ? t('lab.usesTheCurrentTrafficPolicyMatchTheseRulesAreProposed') : t('lab.exampleOnlyDomainSuffixExampleComThisDomainIsNot') }}</p>
            <p v-if="!configJSON" role="status" class="text-sm text-text-secondary">{{ error || t('lab.theCurrentTrafficPolicyNeedsAValidDnsCompatibleMatch') }}</p>
            <pre v-else data-dns-group-config tabindex="0" :aria-label="t('lab.dnsGroupConfigurationJson')" class="max-h-80 overflow-auto rounded-control border border-border bg-bg-tertiary p-3 text-xs leading-relaxed"><code>{{ configJSON }}</code></pre>
            <p v-if="configJSON" class="text-xs text-text-secondary">{{ draft.fallback === 'strict' ? t('lab.evaluateAllSelectedServersAcceptTheFirstNoerrorResponseThen') : t('lab.evaluateAllSelectedServersAndRaceAcceptedResponsesIfNone') }}</p>
            <p class="text-xs text-text-secondary">{{ t('lab.existingServerDefinitionsAreReferencedNotDuplicatedThisFragmentDoes') }}</p>
          </section>
          <p v-if="attempted && error" role="alert" class="mt-3 text-sm text-danger">{{ error }}</p><div class="mt-5 flex flex-wrap items-center justify-between gap-3 border-t border-border pt-4"><p class="text-xs text-text-secondary">{{ live ? t('managed.groupMetadata') : t('lab.previewOnlyNoDatabaseOrConfigChanges') }}</p><div class="flex gap-2"><Button variant="secondary" @click="emit('cancel')">{{ t('lab.cancel') }}</Button><Button type="submit" :disabled="busy">{{ live ? t('managed.preview') : locked ? t('lab.adoptInPreview') : t('lab.saveGroupInPreview') }}</Button></div></div>
        </form>
</template>
