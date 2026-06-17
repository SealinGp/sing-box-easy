<script setup lang="ts">
// Smart Routing Rule wizard — the guided "Add" flow for a route rule.
//
// Steps: 1) Match (pick ONE of domain / domain_suffix / rule_set — enforcing a
// single match type avoids the silent AND-intersection trap), 2) Action,
// 3) Outbound (only when the action needs one), 4) DNS guard (only when routing
// a domain/suffix/rule_set to a *proxied* outbound — reconciles dns.rules so the
// domain resolves cleanly and isn't GFW-poisoned).
//
// Reusable & seedable: RoutingRules opens it blank; RuleSets opens it pre-seeded
// with a freshly-added rule_set. It performs its own submit and emits `completed`
// so any host can refresh.
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useToast } from 'primevue'
import { storeToRefs } from 'pinia'
import { Dialog, Button, Select, Chips } from '../volt'
import type { RouteRule, DNSRule } from '../types/api'
import { routeService, dnsService } from '../services'
import { useOutboundsStore } from '../stores/outbounds'
import { useRouteStore } from '../stores/route'
import { useDNSStore } from '../stores/dns'
import {
  isProxiedOutbound,
  recommendDnsServer,
  reconcileDnsRule,
  type SmartMatchType,
} from '../composables/useSmartDnsSync'

const visible = defineModel<boolean>('visible', { required: true })

const props = defineProps<{
  seedMatchType?: SmartMatchType
  seedValues?: string[]
  seedAction?: string
}>()

const emit = defineEmits<{
  (e: 'completed'): void
  (e: 'advanced'): void
}>()

const { t } = useI18n()
const toast = useToast()

const outboundsStore = useOutboundsStore()
const routeStore = useRouteStore()
const dnsStore = useDNSStore()
const { outbounds } = storeToRefs(outboundsStore)
const { ruleSets } = storeToRefs(routeStore)
const { dnsServers } = storeToRefs(dnsStore)

// Wizard state
const step = ref(1)
const matchType = ref<SmartMatchType>('domain_suffix')
const domainValues = ref<string[]>([]) // for domain / domain_suffix (Chips)
const ruleSetValue = ref<string>('') // for rule_set (single Select)
const action = ref<'route' | 'reject'>('route')
const outbound = ref<string>('')
const submitting = ref(false)

// DNS guard state
const dnsEnabled = ref(true)
const dnsServer = ref<string>('')
const dnsRules = ref<DNSRule[]>([])

// --- options ---
const matchTypeOptions = computed(() => [
  { label: t('route.rules.smart.matchTypes.domain'), value: 'domain' },
  { label: t('route.rules.smart.matchTypes.domainSuffix'), value: 'domain_suffix' },
  { label: t('route.rules.smart.matchTypes.ruleSet'), value: 'rule_set' },
])

const actionOptions = computed(() => [
  { label: t('route.rules.actions.route'), value: 'route' },
  { label: t('route.rules.actions.reject'), value: 'reject' },
])

const outboundOptions = computed(() =>
  outbounds.value.map((o) => ({ label: o.tag || '', value: o.tag || '' })),
)

const ruleSetOptions = computed(() =>
  ruleSets.value.filter((r) => r.tag).map((r) => ({ label: r.tag, value: r.tag })),
)

const dnsServerOptions = computed(() =>
  dnsServers.value.map((s) => ({ label: s.tag, value: s.tag })),
)

// --- derived ---
const matchValues = computed<string[]>(() =>
  matchType.value === 'rule_set'
    ? (ruleSetValue.value ? [ruleSetValue.value] : [])
    : domainValues.value,
)

const needsOutbound = computed(() => action.value === 'route')

// DNS guard applies only when routing a name-based match to a proxied outbound.
const dnsApplies = computed(
  () =>
    action.value === 'route' &&
    matchValues.value.length > 0 &&
    isProxiedOutbound(outbound.value, outbounds.value),
)

// Last step index depends on whether the DNS guard is shown.
const lastStep = computed(() => (dnsApplies.value ? 4 : 3))

const matchValid = computed(() => matchValues.value.length > 0)
const canNext = computed(() => {
  if (step.value === 1) return matchValid.value
  if (step.value === 2) return !!action.value
  if (step.value === 3) return !needsOutbound.value || !!outbound.value
  return true
})

// --- lifecycle: (re)seed each time the dialog opens ---
function resetState() {
  step.value = 1
  matchType.value = props.seedMatchType ?? 'domain_suffix'
  domainValues.value =
    props.seedMatchType && props.seedMatchType !== 'rule_set' ? [...(props.seedValues ?? [])] : []
  ruleSetValue.value =
    props.seedMatchType === 'rule_set' ? (props.seedValues?.[0] ?? '') : ''
  action.value = (props.seedAction as 'route' | 'reject') ?? 'route'
  outbound.value = ''
  dnsEnabled.value = true
  dnsServer.value = ''
  submitting.value = false
  // When seeded (e.g. from RuleSets), the match is pre-filled → jump to Action.
  if (props.seedMatchType && (props.seedValues?.length ?? 0) > 0) step.value = 2
}

async function loadData() {
  // Each fetch self-contains its failure so one outage doesn't block the rest.
  await Promise.all([
    outboundsStore.fetchOutbounds().catch(() => {}),
    routeStore.fetchRuleSets().catch(() => {}),
    dnsStore.fetchDNSServers().catch(() => {}),
    refreshDnsRules(),
  ])
}

async function refreshDnsRules() {
  try {
    const { data } = await dnsService.getDNSRules()
    dnsRules.value = data.rules || []
  } catch {
    dnsRules.value = []
  }
}

watch(visible, (open) => {
  if (open) {
    resetState()
    loadData()
  }
})

// Reset the value inputs when the user switches match type (keeps union-only).
watch(matchType, () => {
  domainValues.value = []
  ruleSetValue.value = ''
})

// When the DNS step becomes relevant, pre-select the recommended clean server.
watch(
  () => step.value,
  (s) => {
    if (s === 4 && !dnsServer.value) {
      dnsServer.value =
        recommendDnsServer(matchType.value, dnsRules.value, dnsServers.value) ?? ''
    }
  },
)

// --- navigation ---
function next() {
  if (!canNext.value) {
    toast.add({ severity: 'warn', summary: t('common.error'), detail: t('route.rules.smart.toast.validation'), life: 2500 })
    return
  }
  // Skip the outbound step for actions that don't need one (e.g. reject).
  if (step.value === 2 && !needsOutbound.value) {
    submit()
    return
  }
  if (step.value >= lastStep.value) {
    submit()
    return
  }
  step.value += 1
}

function back() {
  if (step.value > 1) step.value -= 1
}

function openAdvanced() {
  emit('advanced')
  visible.value = false
}

// --- submit ---
function buildRouteRule(): RouteRule {
  const rule: Record<string, unknown> = { action: action.value }
  if (action.value === 'route') rule.outbound = outbound.value
  rule[matchType.value] = [...matchValues.value]
  return rule as RouteRule
}

async function syncDns() {
  if (!dnsApplies.value || !dnsEnabled.value || !dnsServer.value) return
  await refreshDnsRules()
  const result = reconcileDnsRule(matchType.value, matchValues.value, dnsServer.value, dnsRules.value)
  if (result.op === 'noop') return
  if (result.op === 'append' && result.index !== undefined) {
    await dnsService.updateDNSRule(result.index, result.rule)
    toast.add({ severity: 'success', summary: t('common.success'), detail: t('route.rules.smart.dns.appended'), life: 3000 })
  } else if (result.op === 'create') {
    await dnsService.addDNSRule(result.rule)
    toast.add({ severity: 'success', summary: t('common.success'), detail: t('route.rules.smart.dns.created'), life: 3000 })
  }
}

async function submit() {
  if (!matchValid.value) {
    step.value = 1
    return
  }
  submitting.value = true
  try {
    await routeService.addRouteRule(buildRouteRule())
    toast.add({ severity: 'success', summary: t('common.success'), detail: t('route.rules.smart.toast.added'), life: 3000 })

    // DNS guard runs after the route rule is committed, with its own toast so a
    // DNS hiccup never looks like the route add failed.
    if (dnsApplies.value && dnsEnabled.value) {
      try {
        await syncDns()
      } catch (err: any) {
        toast.add({ severity: 'warn', summary: t('common.error'), detail: err?.message || t('route.rules.smart.toast.dnsFailed'), life: 4000 })
      }
    }

    emit('completed')
    visible.value = false
  } catch (err: any) {
    toast.add({ severity: 'error', summary: t('common.error'), detail: err?.message || t('route.rules.smart.toast.addFailed'), life: 3000 })
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Dialog
    v-model:visible="visible"
    modal
    :header="t('route.rules.smart.title')"
    class="w-full max-w-xl"
  >
    <!-- Step indicator -->
    <div class="flex items-center gap-2 mb-5 text-xs">
      <span :class="['px-2 py-1 rounded', step >= 1 ? 'bg-violet-600 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-300']">1 · {{ t('route.rules.smart.steps.match') }}</span>
      <span :class="['px-2 py-1 rounded', step >= 2 ? 'bg-violet-600 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-300']">2 · {{ t('route.rules.smart.steps.action') }}</span>
      <span v-if="needsOutbound" :class="['px-2 py-1 rounded', step >= 3 ? 'bg-violet-600 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-300']">3 · {{ t('route.rules.smart.steps.outbound') }}</span>
      <span v-if="dnsApplies" :class="['px-2 py-1 rounded', step >= 4 ? 'bg-violet-600 text-white' : 'bg-gray-200 dark:bg-gray-700 text-gray-600 dark:text-gray-300']">4 · {{ t('route.rules.smart.steps.dns') }}</span>
    </div>

    <!-- Step 1 — Match -->
    <div v-if="step === 1" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.smart.matchType') }}</label>
        <Select
          v-model="matchType"
          :options="matchTypeOptions"
          optionLabel="label"
          optionValue="value"
          class="w-full"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('route.rules.smart.matchTypeHint') }}</p>
      </div>

      <div v-if="matchType === 'rule_set'">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.smart.values.ruleSet') }}</label>
        <Select
          v-model="ruleSetValue"
          :options="ruleSetOptions"
          optionLabel="label"
          optionValue="value"
          filter
          :placeholder="t('route.rules.smart.valuesPlaceholder.ruleSet')"
          class="w-full"
        />
      </div>

      <div v-else>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
          {{ matchType === 'domain' ? t('route.rules.smart.values.domain') : t('route.rules.smart.values.domainSuffix') }}
        </label>
        <Chips
          v-model="domainValues"
          :placeholder="matchType === 'domain' ? t('route.rules.smart.valuesPlaceholder.domain') : t('route.rules.smart.valuesPlaceholder.domainSuffix')"
          class="w-full"
        />
      </div>
    </div>

    <!-- Step 2 — Action -->
    <div v-else-if="step === 2" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.smart.action') }}</label>
        <Select
          v-model="action"
          :options="actionOptions"
          optionLabel="label"
          optionValue="value"
          class="w-full"
        />
      </div>
    </div>

    <!-- Step 3 — Outbound -->
    <div v-else-if="step === 3" class="space-y-4">
      <div>
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.smart.outbound') }}</label>
        <Select
          editable
          v-model="outbound"
          :options="outboundOptions"
          optionLabel="label"
          optionValue="value"
          :placeholder="t('route.rules.placeholders.outbound')"
          class="w-full"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('route.rules.smart.outboundHint') }}</p>
      </div>
    </div>

    <!-- Step 4 — DNS guard -->
    <div v-else-if="step === 4" class="space-y-4">
      <div class="rounded-md bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-800 p-3">
        <h4 class="text-sm font-semibold text-amber-800 dark:text-amber-300">{{ t('route.rules.smart.dns.heading') }}</h4>
        <p class="mt-1 text-xs text-amber-700 dark:text-amber-400">
          {{ t('route.rules.smart.dns.intro', { outbound }) }}
        </p>
      </div>

      <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
        <input type="checkbox" v-model="dnsEnabled" class="rounded border-gray-300 text-violet-600 focus:ring-violet-500" />
        {{ t('route.rules.smart.dns.enable') }}
      </label>

      <div v-if="dnsEnabled">
        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ t('route.rules.smart.dns.server') }}</label>
        <Select
          v-model="dnsServer"
          :options="dnsServerOptions"
          optionLabel="label"
          optionValue="value"
          class="w-full"
        />
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ t('route.rules.smart.dns.serverHint') }}</p>
      </div>
    </div>

    <template #footer>
      <div class="flex w-full items-center justify-between">
        <button
          type="button"
          class="text-xs text-gray-500 hover:text-violet-600 dark:text-gray-400"
          @click="openAdvanced"
        >
          {{ t('route.rules.smart.advanced') }}
        </button>
        <div class="flex gap-2">
          <Button v-if="step > 1" :label="t('route.rules.smart.back')" severity="secondary" @click="back" />
          <Button
            :label="step >= lastStep || (step === 2 && !needsOutbound) ? t('route.rules.smart.finish') : t('route.rules.smart.next')"
            :disabled="!canNext || submitting"
            @click="next"
          />
        </div>
      </div>
    </template>
  </Dialog>
</template>
