<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Card from './Card.vue'
import { Dialog, Select } from '../volt'
import Button from './Button.vue'
import RoutingRuleItem from './RoutingRuleItem.vue'
import List from './List.vue'
import RouteRuleMatchers from './RouteRuleMatchers.vue'
import SchemaFieldsEditor from './SchemaFieldsEditor.vue'
import {
  ROUTE_RULE_ACTION_TYPE_NAMES,
  applyActionDefaults,
  pruneForeignFields,
  resolveRouteRuleActionFields,
  validateRouteRuleAction,
  type RouteRuleActionTypeName,
} from '../schemas/routeRuleActionFields'
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
// Derived from the generated inventory rather than hand-listed, so it cannot
// drift from what the registry can build. The old list was missing `direct`
// entirely. Labels fall back to the action's own name.
const ACTION_LABEL_KEYS: Record<string, string> = {
  route: 'route.rules.actions.route',
  'route-options': 'route.rules.actions.routeOptions',
  direct: 'route.rules.actions.direct',
  reject: 'route.rules.actions.reject',
  'hijack-dns': 'route.rules.actions.hijackDns',
  sniff: 'route.rules.actions.sniff',
  resolve: 'route.rules.actions.resolve',
}

const actionOptions = computed(() =>
  ROUTE_RULE_ACTION_TYPE_NAMES.map((value) => ({
    value,
    label: ACTION_LABEL_KEYS[value] ? t(ACTION_LABEL_KEYS[value]!) : value,
  })),
)

// Computed properties for v-model
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

// Effective action — drives which action-specific fields are shown. Falls back
// to the "route" default so an action-less rule still renders its outbound field.
const currentAction = computed(() => (editingRule.value ? editingRule.value.rule.action : ruleForm.value.action) || ACTION_DEFAULT)

// True when the rule has no explicit `action` (i.e. it relies on the "route"
// default). Clears once the user picks an action, which hides the hint.
const actionIsDefaulted = computed(() => {
  const raw = editingRule.value ? editingRule.value.rule.action : ruleForm.value.action
  return !raw
})

/**
 * The action driving the schema. An omitted `action` means "route".
 *
 * Kept separate from `currentRuleAction` above, which is what the Select writes:
 * that one must NOT persist the default, so a rule that omitted `action` keeps
 * omitting it and `actionIsDefaulted` stays true.
 */
const actionFields = computed(() => resolveRouteRuleActionFields(currentAction.value))

/**
 * The rule as a plain record, for the schema editor.
 *
 * `SchemaFieldsEditor` replaces the whole object with an immutable spread that
 * touches only the keys it renders, so it composes with RouteRuleMatchers'
 * three editors and with the action Select without any of them stepping on
 * each other.
 */
const actionRecord = computed<Record<string, unknown>>({
  get: () => activeRule.value as Record<string, unknown>,
  set: (v) => {
    activeRule.value = v as RouteRule
  },
})

/**
 * Switching the action prunes the previous one's fields and seeds the new one's
 * defaults.
 *
 * There was no handler here at all before, and no pruning anywhere on the route
 * path — no allowlist, no denylist. The form seeds `{action:'route', outbound:''}`,
 * so picking another action shipped the previous one's fields. Reproduced
 * against a running panel:
 *
 *   {"action":"reject","outbound":"direct"}   -> 200, outbound SILENTLY DROPPED
 *   {"action":"sniff","outbound":"direct",…}  -> 400 unknown field "outbound"
 *
 * Silent data loss or a hard error depending on the combination.
 * `pruneForeignFields` removes only keys another action owns, so all 37
 * matchers — and `type`/`mode`/`rules` on a logical rule — survive untouched.
 */
function changeAction(action: RouteRuleActionTypeName) {
  const pruned = pruneForeignFields(activeRule.value as Record<string, unknown>, action)
  activeRule.value = applyActionDefaults(pruned, action) as RouteRule
}

/** Outbound tags that actually exist, for the validation `sing-box check` skips. */
const knownOutboundTags = computed(() =>
  outbounds.value.map((o) => o.tag || '').filter(Boolean),
)

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

/**
 * Constraints sing-box enforces while decoding or starting, which surface as an
 * opaque upstream string — or, for an outbound tag that does not exist, not at
 * all: `sing-box check` PASSES for that one, so the client is the only place it
 * can be caught.
 *
 * Returns false and shows the reason when the rule cannot be saved.
 */
function guardAction(rule: RouteRule): boolean {
  const invalid = validateRouteRuleAction(
    rule as Record<string, unknown>,
    knownOutboundTags.value,
  )
  if (!invalid) return true

  toast.add({
    severity: 'error',
    summary: t('common.error'),
    detail: t(invalid),
    life: 4000,
  })
  return false
}

function submitAddRule() {
  if (!guardAction(ruleForm.value)) return
  handleAddRule(ruleForm.value)
}

function submitUpdateRule() {
  if (!editingRule.value) return
  if (!guardAction(editingRule.value.rule)) return
  handleEditRule(editingRule.value.index, editingRule.value.rule)
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
            :modelValue="currentAction"
            @update:modelValue="changeAction"
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

        <!--
          Everything the action owns, from the generated inventory. Replaces six
          hand-written v-if blocks that covered 9 of the 45 fields across the
          seven actions, offered no `direct` action at all, and left
          `route-options` unable to reach seven of its ten.

          :key remounts on action change so the previous action's added-field
          state does not leak — same reason matchersKey exists below.
        -->
        <SchemaFieldsEditor
          :key="currentAction"
          v-model="actionRecord"
          :fields="actionFields"
          :empty-hint="$t('route.rules.hints.hijackDns')"
        />

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