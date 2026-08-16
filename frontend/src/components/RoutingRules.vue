<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Card from './Card.vue'
import { Dialog, Select } from '../volt'
import ChipsField from './ChipsField.vue'
import Button from './Button.vue'
import RoutingRuleItem from './RoutingRuleItem.vue'
import List from './List.vue'
import RouteRuleMatchers from './RouteRuleMatchers.vue'
import SmartRoutingRuleWizard from './SmartRoutingRuleWizard.vue'
import type { RouteRule, Outbound } from '../types/api'
import { routeService, outboundService } from '../services'
import { useToast } from 'primevue'

// sing-box accepts scalar OR array on the wire for every list-like matcher
// (e.g. "inbound": "dns-in" is equivalent to ["dns-in"]). The backend
// round-trips whatever shape lives in config.json, so coerce to arrays at the
// boundary. Without this: `.join()` crashes on strings, and `[...str]`
// silently splits "dns-in" into characters in startEditRule.
function toArray<T>(v: T | T[] | undefined | null): T[] | undefined {
  if (v === undefined || v === null) return undefined
  return Array.isArray(v) ? v : [v]
}

// Input is the raw wire payload (scalar-or-array), output matches RouteRule
// (post-normalization, arrays only). Cast on entry because the wire shape is
// intentionally not captured in the TS contract — see the note on RouteRule.
function normalizeRouteRule(rule: RouteRule): RouteRule {
  const raw = rule as Record<string, unknown>
  return {
    ...rule,
    inbound: toArray(raw.inbound as string | string[] | undefined),
    protocol: toArray(raw.protocol as string | string[] | undefined),
    network: toArray(raw.network as string | string[] | undefined),
    domain: toArray(raw.domain as string | string[] | undefined),
    domain_suffix: toArray(raw.domain_suffix as string | string[] | undefined),
    domain_keyword: toArray(raw.domain_keyword as string | string[] | undefined),
    domain_regex: toArray(raw.domain_regex as string | string[] | undefined),
    geosite: toArray(raw.geosite as string | string[] | undefined),
    source_geoip: toArray(raw.source_geoip as string | string[] | undefined),
    geoip: toArray(raw.geoip as string | string[] | undefined),
    ip_cidr: toArray(raw.ip_cidr as string | string[] | undefined),
    source_ip_cidr: toArray(raw.source_ip_cidr as string | string[] | undefined),
    source_port: toArray(raw.source_port as number | number[] | undefined),
    port: toArray(raw.port as number | number[] | undefined),
    rule_set: toArray(raw.rule_set as string | string[] | undefined),
    sniffer: toArray(raw.sniffer as string | string[] | undefined),
  }
}

const toast = useToast()
const { t } = useI18n()

// Local state
const loading = ref(false)
const rules = ref<RouteRule[]>([])
const outbounds = ref<Outbound[]>([])

// State for dialog
const showAddRuleDialog = ref(false)
const editingRule = ref<{ index: number; rule: RouteRule } | null>(null)

// Guided "Smart Routing Rule" wizard — the default Add flow. The legacy
// full-form dialog (showAddRuleDialog) is kept for Edit and for the wizard's
// "Advanced options" escape hatch.
const showWizard = ref(false)
function openLegacyAdd() {
  showWizard.value = false
  matchersKey.value++
  showAddRuleDialog.value = true
}

// Bumped on every dialog open; used as the matchers block's :key so it remounts
// and re-derives which matching style this rule uses. Without it, reopening the
// Add dialog would inherit the collapse state left behind by the previous rule.
const matchersKey = ref(0)

// Form data
const ruleForm = ref<RouteRule>({ action: 'route', outbound: '' })

// Fetch data
const fetchRouteRules = async () => {
  loading.value = true
  try {
    const { data } = await routeService.getRouteRules()
    rules.value = (data.rules || []).map(normalizeRouteRule)
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.rules.toast.fetchFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const fetchOutbounds = async () => {
  try {
    const { data } = await outboundService.getOutbounds()
    outbounds.value = data.outbounds || []
  } catch (err: any) {
    console.error('Failed to fetch outbounds:', err)
  }
}

// Handlers
const handleAddRule = async (rule: RouteRule) => {
  loading.value = true
  try {
    await routeService.addRouteRule(rule)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('route.rules.toast.added'),
      life: 3000
    })
    await fetchRouteRules()
    showAddRuleDialog.value = false
    ruleForm.value = { action: 'route', outbound: '' }
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.rules.toast.addFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const handleEditRule = async (index: number, rule: RouteRule) => {
  loading.value = true
  try {
    await routeService.updateRouteRule(index, rule)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('route.rules.toast.updated'),
      life: 3000
    })
    await fetchRouteRules()
    editingRule.value = null
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.rules.toast.updateFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Confirmation happens inline, in the <PopConfirm> inside RoutingRuleItem —
// anchored to the row so the user can see which rule they are deleting. By the
// time this fires the user has already confirmed.
const handleDeleteRule = async (index: number) => {
  loading.value = true
  try {
    await routeService.deleteRouteRule(index)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('route.rules.toast.deleted'),
      life: 3000
    })
    await fetchRouteRules()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.rules.toast.deleteFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Options
const outboundOptions = computed(() =>
  outbounds.value.map(o => ({ label: o.tag || '', value: o.tag || '' }))
)

const actionOptions = computed(() => [
  { label: t('route.rules.actions.route'), value: 'route' },
  { label: t('route.rules.actions.reject'), value: 'reject' },
  { label: t('route.rules.actions.routeOptions'), value: 'route-options' },
  { label: t('route.rules.actions.sniff'), value: 'sniff' },
  { label: t('route.rules.actions.resolve'), value: 'resolve' },
  { label: t('route.rules.actions.hijackDns'), value: 'hijack-dns' },
])

const rejectMethodOptions = computed(() => [
  { label: t('route.rules.rejectMethods.default'), value: 'default' },
  { label: t('route.rules.rejectMethods.drop'), value: 'drop' },
])

const networkStrategyOptions = computed(() => [
  { label: t('route.rules.networkStrategies.preferIpv4'), value: 'prefer_ipv4' },
  { label: t('route.rules.networkStrategies.preferIpv6'), value: 'prefer_ipv6' },
  { label: t('route.rules.networkStrategies.ipv4Only'), value: 'ipv4_only' },
  { label: t('route.rules.networkStrategies.ipv6Only'), value: 'ipv6_only' },
])

const dnsStrategyOptions = computed(() => [
  { label: t('route.rules.networkStrategies.preferIpv4'), value: 'prefer_ipv4' },
  { label: t('route.rules.networkStrategies.preferIpv6'), value: 'prefer_ipv6' },
  { label: t('route.rules.networkStrategies.ipv4Only'), value: 'ipv4_only' },
  { label: t('route.rules.networkStrategies.ipv6Only'), value: 'ipv6_only' },
])

// Computed properties for v-model
const currentRuleOutbound = computed({
  get: () => editingRule.value ? editingRule.value.rule.outbound : ruleForm.value.outbound,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.outbound = val
    } else {
      ruleForm.value.outbound = val
    }
  }
})

// The rule currently being edited (existing rule) or composed (add form). The
// common-matcher fields are delegated to <RouteRuleMatchers v-model="activeRule">,
// which updates immutably; the setter reassigns the underlying object so the
// parent's action-specific computeds keep reading fresh state.
const activeRule = computed<RouteRule>({
  get: () => editingRule.value ? editingRule.value.rule : ruleForm.value,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule = val
    } else {
      ruleForm.value = val
    }
  }
})

// Per the sing-box docs, `action` defaults to "route" when omitted. We surface
// that default in the UI (the Select shows "Route" and the route/outbound field
// renders) but DO NOT write it back into the rule: a rule that omitted `action`
// keeps omitting it on save, and `actionIsDefaulted` stays true so the hint shows.
const ACTION_DEFAULT = 'route'

const currentRuleAction = computed({
  get: () => (editingRule.value ? editingRule.value.rule.action : ruleForm.value.action) || ACTION_DEFAULT,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.action = val
    } else {
      ruleForm.value.action = val
    }
  }
})

// Effective action — drives which action-specific fields are shown. Falls back
// to the "route" default so an action-less rule still renders its outbound field.
const currentAction = computed(() => (editingRule.value ? editingRule.value.rule.action : ruleForm.value.action) || ACTION_DEFAULT)

// True when the rule has no explicit `action` (i.e. it relies on the "route"
// default). Clears once the user picks an action, which hides the hint.
const actionIsDefaulted = computed(() => {
  const raw = editingRule.value ? editingRule.value.rule.action : ruleForm.value.action
  return !raw
})

// Action-specific computed properties

// Reject action
const currentRuleMethod = computed({
  get: () => editingRule.value ? editingRule.value.rule.method : ruleForm.value.method,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.method = val
    } else {
      ruleForm.value.method = val
    }
  }
})

// Route-options action
const currentRuleOverrideAddress = computed({
  get: () => editingRule.value ? editingRule.value.rule.override_address : ruleForm.value.override_address,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.override_address = val
    } else {
      ruleForm.value.override_address = val
    }
  }
})

const currentRuleOverridePort = computed({
  get: () => editingRule.value ? editingRule.value.rule.override_port : ruleForm.value.override_port,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.override_port = val
    } else {
      ruleForm.value.override_port = val
    }
  }
})

const currentRuleNetworkStrategy = computed({
  get: () => editingRule.value ? editingRule.value.rule.network_strategy : ruleForm.value.network_strategy,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.network_strategy = val
    } else {
      ruleForm.value.network_strategy = val
    }
  }
})

// Sniff action
const currentRuleSniffer = computed({
  get: () => {
    const sniffer = editingRule.value ? editingRule.value.rule.sniffer : ruleForm.value.sniffer
    return Array.isArray(sniffer) ? sniffer : (sniffer ? [sniffer] : [])
  },
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.sniffer = val
    } else {
      ruleForm.value.sniffer = val
    }
  }
})

const currentRuleTimeout = computed({
  get: () => editingRule.value ? editingRule.value.rule.timeout : ruleForm.value.timeout,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.timeout = val
    } else {
      ruleForm.value.timeout = val
    }
  }
})

// Resolve action
const currentRuleServer = computed({
  get: () => editingRule.value ? editingRule.value.rule.server : ruleForm.value.server,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.server = val
    } else {
      ruleForm.value.server = val
    }
  }
})

const currentRuleStrategy = computed({
  get: () => editingRule.value ? editingRule.value.rule.strategy : ruleForm.value.strategy,
  set: (val) => {
    if (editingRule.value) {
      editingRule.value.rule.strategy = val
    } else {
      ruleForm.value.strategy = val
    }
  }
})

// Dialog visibility
const dialogVisible = computed({
  get: () => showAddRuleDialog.value || !!editingRule.value,
  set: (val) => {
    if (!val) {
      showAddRuleDialog.value = false
      editingRule.value = null
      ruleForm.value = { action: 'route', outbound: '' }
    }
  }
})

// Functions
function startEditRule(index: number, rule: RouteRule) {
  // Ensure add dialog is closed
  showAddRuleDialog.value = false

  // Deep copy the rule to avoid mutations
  const ruleCopy: RouteRule = {
    ...rule,
    // Common matching criteria
    inbound: rule.inbound ? [...rule.inbound] : undefined,
    protocol: rule.protocol ? [...rule.protocol] : undefined,
    network: rule.network ? [...rule.network] : undefined,
    domain: rule.domain ? [...rule.domain] : undefined,
    domain_suffix: rule.domain_suffix ? [...rule.domain_suffix] : undefined,
    domain_keyword: rule.domain_keyword ? [...rule.domain_keyword] : undefined,
    domain_regex: rule.domain_regex ? [...rule.domain_regex] : undefined,
    geosite: rule.geosite ? (Array.isArray(rule.geosite) ? [...rule.geosite] : rule.geosite) : undefined,
    source_geoip: rule.source_geoip ? [...rule.source_geoip] : undefined,
    geoip: rule.geoip ? (Array.isArray(rule.geoip) ? [...rule.geoip] : rule.geoip) : undefined,
    ip_cidr: rule.ip_cidr ? [...rule.ip_cidr] : undefined,
    source_ip_cidr: rule.source_ip_cidr ? [...rule.source_ip_cidr] : undefined,
    source_port: rule.source_port ? [...rule.source_port] : undefined,
    port: rule.port ? [...rule.port] : undefined,
    rule_set: rule.rule_set ? (Array.isArray(rule.rule_set) ? [...rule.rule_set] : rule.rule_set) : undefined,
    // Sniffer field can be string[] or string
    sniffer: rule.sniffer ? (Array.isArray(rule.sniffer) ? [...rule.sniffer] : rule.sniffer) : undefined,
  }

  matchersKey.value++
  editingRule.value = { index, rule: ruleCopy }
}

function submitAddRule() {
  handleAddRule(ruleForm.value)
}

function submitUpdateRule() {
  if (editingRule.value) {
    handleEditRule(editingRule.value.index, editingRule.value.rule)
  }
}

function submitDeleteRule(index: number) {
  handleDeleteRule(index)
}

// Load data on mount
onMounted(() => {
  fetchRouteRules()
  fetchOutbounds()
})
</script>

<template>
  <div class="space-y-4">
    <Card>
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
          {{ $t('route.rules.title') }}
        </h3>
        <button
          @click="showWizard = true"
          class="px-3 py-1.5 text-sm font-medium bg-primary-600 text-white rounded-control hover:bg-primary-700 transition-colors"
        >
          {{ $t('route.rules.add') }}
        </button>
      </div>

      <List :loading="loading" :empty="rules.length === 0">
        <template #empty>{{ $t('route.rules.empty') }}</template>

        <RoutingRuleItem
          v-for="(rule, index) in rules"
          :key="index"
          :rule="rule"
          :index="index"
          @edit="startEditRule"
          @delete="submitDeleteRule"
        />
      </List>
    </Card>

    <!-- Guided add wizard (default Add flow) -->
    <SmartRoutingRuleWizard
      v-model:visible="showWizard"
      @completed="fetchRouteRules"
      @advanced="openLegacyAdd"
    />

    <!-- Add/Edit Rule Dialog -->
    <Dialog
      v-model:visible="dialogVisible"
      modal
      :header="editingRule ? $t('route.rules.modal.edit') : $t('route.rules.modal.add')"
      class="w-full max-w-2xl"
    >
      <div :key="editingRule ? `edit-${editingRule.index}` : 'add'" class="space-y-4">
        <!-- Action Selection -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.action') }}</label>
          <Select
            v-model="currentRuleAction"
            :options="actionOptions"
            optionLabel="label"
            optionValue="value"
            :placeholder="$t('route.rules.placeholders.action')"
            class="w-full"
          />
          <p v-if="actionIsDefaulted" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
            {{ $t('route.rules.actionDefaultHint') }}
          </p>
        </div>

        <!-- Action-specific fields -->

        <!-- Route Action -->
        <div v-if="currentAction === 'route'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.outbound') }}</label>
          <Select
            editable
            v-model="currentRuleOutbound"
            :options="outboundOptions"
            optionLabel="label"
            optionValue="value"
            :placeholder="$t('route.rules.placeholders.outbound')"
            class="w-full"
          />
        </div>

        <!-- Reject Action -->
        <template v-if="currentAction === 'reject'">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.method') }}</label>
            <Select
              v-model="currentRuleMethod"
              :options="rejectMethodOptions"
              optionLabel="label"
              optionValue="value"
              :placeholder="$t('route.rules.placeholders.method')"
              class="w-full"
            />
          </div>
        </template>

        <!-- Route Options Action -->
        <template v-if="currentAction === 'route-options'">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.overrideAddress') }}</label>
            <input
              v-model="currentRuleOverrideAddress"
              type="text"
              :placeholder="$t('route.rules.placeholders.overrideAddress')"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-surface bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.overridePort') }}</label>
            <input
              v-model.number="currentRuleOverridePort"
              type="number"
              :placeholder="$t('route.rules.placeholders.overridePort')"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-surface bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.networkStrategy') }}</label>
            <Select
              v-model="currentRuleNetworkStrategy"
              :options="networkStrategyOptions"
              optionLabel="label"
              optionValue="value"
              :placeholder="$t('route.rules.placeholders.networkStrategy')"
              class="w-full"
            />
          </div>
        </template>

        <!-- Sniff Action -->
        <template v-if="currentAction === 'sniff'">
          <ChipsField
            v-model="currentRuleSniffer"
            :label="$t('route.rules.fields.sniffer')"
            :placeholder="$t('route.rules.placeholders.sniffer')"
          />
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.timeout') }}</label>
            <input
              v-model="currentRuleTimeout"
              type="text"
              :placeholder="$t('route.rules.placeholders.timeout')"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-surface bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>
        </template>

        <!-- Resolve Action -->
        <template v-if="currentAction === 'resolve'">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.dnsServer') }}</label>
            <input
              v-model="currentRuleServer"
              type="text"
              :placeholder="$t('route.rules.placeholders.dnsServer')"
              class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-surface bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('route.rules.fields.strategy') }}</label>
            <Select
              v-model="currentRuleStrategy"
              :options="dnsStrategyOptions"
              optionLabel="label"
              optionValue="value"
              :placeholder="$t('route.rules.placeholders.strategy')"
              class="w-full"
            />
          </div>
        </template>

        <!-- Common matching criteria fields (reusable, renders anywhere).
             Remounted per dialog open (:key) so the collapse state is decided
             from the rule as loaded, not from the previous edit. -->
        <RouteRuleMatchers :key="matchersKey" v-model="activeRule" />
      </div>

      <template #footer>
        <Button
          :label="$t('common.cancel')"
          severity="secondary"
          @click="dialogVisible = false"
        />
        <Button
          :label="editingRule ? $t('common.update') : $t('common.add')"
          @click="editingRule ? submitUpdateRule() : submitAddRule()"
        />
      </template>
    </Dialog>
  </div>
</template>