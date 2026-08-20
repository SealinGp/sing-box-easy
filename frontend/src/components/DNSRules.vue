<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DNSRule } from '../types/api'
import Button from './Button.vue'
import Modal from './Modal.vue'
import { Select } from '../volt'
import Badge from './Badge.vue'
import Table from './Table.vue'
import DNSRuleConditions from './DNSRuleConditions.vue'
import SchemaFieldsEditor from './SchemaFieldsEditor.vue'
import RuleFlowPreview from './RuleFlowPreview.vue'
import { PlusIcon, PencilIcon, TrashIcon, Bars3Icon, ArrowsUpDownIcon } from '@heroicons/vue/24/outline'
import {  dnsService } from '../services'
import { useToast } from 'primevue'
import { useDragReorder } from '../composables/useDragReorder'
import { useDNSStore } from '../stores/dns'
import { useRouteStore } from '../stores/route'
import { storeToRefs } from 'pinia'
import {
  actionOf,
  applyActionDefaults,
  pruneForeignFields,
  resolveDNSRuleActionFields,
  validateDNSRuleAction,
  isTerminalAction,
  ALL_ACTION_KEYS,
  type DNSRuleActionTypeName,
} from '../schemas/dnsRuleActionFields'

const toast = useToast()
const { t } = useI18n()
const dnsStore = useDNSStore()
const routeStore = useRouteStore()
// The server list is no longer read here — the `dns-server` control in
// SchemaFieldControl.vue owns that picker now. The store is still prefetched on
// mount (below) so the select is populated before it is first opened.
const { ruleSets } = storeToRefs(routeStore)

// Local state for DNS rules
const loading = ref(false)
const dnsRules = ref<DNSRule[]>([])

// Modal state
const showRuleModal = ref(false)
// Bumped on every modal open; used as the conditions block's :key so it
// remounts and re-derives which matcher style this rule uses.
const conditionsKey = ref(0)
const isEditMode = ref(false)
const editingIndex = ref(-1)
/**
 * The rule being edited, as a whole.
 *
 * This used to be a fixed 9-key object rebuilt from the loaded rule, with
 * everything else stashed in a `preservedFields` side-channel and re-merged on
 * save — but only while the action was unchanged. That allowlist caused three
 * separate data-loss bugs: `no_drop` was listed but never rendered, so it was
 * read into nothing and written back as nothing; `strategy` was neither listed
 * nor rendered, so a real rule in this repo's test config had
 * `"strategy": "ipv4_only"` erased just by opening it; and a logical rule lost
 * its `type`/`mode`/`rules` the moment its action changed.
 *
 * Now the whole rule is the model. `SchemaFieldsEditor` only touches keys it
 * renders, and `pruneForeignFields` removes only keys some OTHER action owns.
 * Anything neither knows about — every match condition, and any field a future
 * sing-box adds — survives by construction.
 */
const currentRule = ref<Record<string, any>>(emptyRuleForm())

/**
 * Wire fields sing-box accepts as either a scalar or an array.
 *
 * The five conditions have always needed this. `answer`/`ns`/`extra` need it too
 * and are new here: badoption.Listable collapses a single-entry list to a bare
 * string on marshal, so a predefined rule with one answer reads back as
 * `"answer": "a.example. 3600 IN A 192.0.2.1"`. Left uncoerced, the chips box
 * would render one chip per CHARACTER.
 */
/**
 * Conditions, which DNSRuleConditions v-models as arrays. Always materialized,
 * present in the rule or not — its ChipsFields cannot bind to undefined.
 */
const CONDITION_FIELDS = ['rule_set', 'domain', 'domain_suffix', 'domain_keyword', 'geosite'] as const

/**
 * The predefined action's record lists. Coerced only when present, so opening a
 * route rule does not add three empty keys to it.
 */
const ACTION_LIST_FIELDS = ['answer', 'ns', 'extra'] as const

/** The action driving the schema. Absent means route — see `actionOf`. */
const currentAction = computed<DNSRuleActionTypeName>(() => actionOf(currentRule.value))

/** The fields this action owns, resolved from the generated inventory. */
const actionFields = computed(() => resolveDNSRuleActionFields(currentAction.value))

/**
 * Everything on the rule that is NOT the action — i.e. its conditions.
 *
 * Derived by subtraction rather than from a list, because there is no generated
 * matcher inventory for DNS rules (the conditions half stayed hand-written).
 * Subtracting means a condition the form has no control for still SHOWS in the
 * preview, which is the honest behaviour: the rule is matching on it either way.
 */
const STRUCTURAL_KEYS = ['action', 'type', 'mode', 'rules', 'invert']
const conditionKeys = computed(() =>
  Object.keys(currentRule.value).filter(
    (key) => !ALL_ACTION_KEYS.includes(key) && !STRUCTURAL_KEYS.includes(key),
  ),
)

/**
 * The outcome, as a phrase, for the flow preview. Each action names the field
 * that carries its result.
 */
const flowOutcome = computed(() => {
  const r = currentRule.value
  switch (currentAction.value) {
    case 'route':
      return r.server
        ? t('dns.rules.flow.then.route', { server: String(r.server) })
        : t('dns.rules.flow.then.routeIncomplete')
    case 'route-options':
      return t('dns.rules.flow.then.routeOptions')
    case 'reject':
      return r.method === 'drop'
        ? t('dns.rules.flow.then.rejectDrop')
        : t('dns.rules.flow.then.reject')
    case 'predefined':
      return r.rcode
        ? t('dns.rules.flow.then.predefinedRcode', { rcode: String(r.rcode) })
        : t('dns.rules.flow.then.predefined')
    default:
      return currentAction.value
  }
})

/**
 * Only `route-options` keeps matching — dns/router.go:147-195, where route,
 * reject and predefined each return and route-options falls through.
 *
 * Note the RouteRule list is different: there route-options, direct, resolve
 * and sniff are the non-terminal ones. Same words, different families.
 */
const flowContinues = computed(() => !isTerminalAction(currentAction.value))

/**
 * Switching the action prunes the previous one's fields and seeds the new one's
 * defaults — the analogue of `changeType()` in DNSServers.vue.
 *
 * There was no handler here at all before, which is why switching `route` to
 * `route-options` carried `server` along: the save-time `switch` deleted
 * method/no_drop/rcode but not server, and DNSRouteOptionsActionOptions has no
 * field for it, so the save failed on a field the form had stopped showing.
 */
function changeAction(action: DNSRuleActionTypeName) {
  const pruned = pruneForeignFields(currentRule.value, action)
  // Defaults are for NEW records only. On an existing rule the operator is
  // changing the action deliberately, so the new action's core fields do need
  // seeding — but nothing already set is overwritten.
  currentRule.value = applyActionDefaults(pruned, action)
}

/**
 * What a `predefined` rule answers with, for the server column.
 *
 * The rcode defaults to NOERROR, not NXDOMAIN. The column used to hardcode
 * `rcode || 'NXDOMAIN'`, which reported the opposite of the truth for a rule
 * that omits it — and every rule that answers with records omits it, because
 * NOERROR is what "here is your answer" means. The same wrong assumption used to
 * live in the edit form, where it did not merely display wrongly but rewrote the
 * rule on save.
 *
 * Records win over the rcode when both are present: a rule returning a CNAME is
 * described by that CNAME, not by the "no error" that accompanies it.
 */
function predefinedSummary(rule: unknown): string {
  const raw = rule as Record<string, unknown>
  const answers = toArrayField(raw.answer)
  if (answers.length === 1) return answers[0]!
  if (answers.length > 1) return t('dns.rules.table.answerCount', { count: answers.length })
  return (raw.rcode as string) || 'NOERROR'
}

function toArrayField(v: unknown): string[] {
  if (v === undefined || v === null) return []
  return Array.isArray(v) ? (v as string[]) : [String(v)]
}

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingIndex = ref(-1)

// Fetch DNS rules
const fetchDNSRules = async () => {
  loading.value = true
  try {
    const { data } = await dnsService.getDNSRules()
    dnsRules.value = data.rules || []
    reorder.syncKeys(dnsRules.value.length)
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('dns.rules.toast.fetchFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}


const actionTypes = computed(() => [
  { value: 'route', label: t('dns.rules.actionTypes.route') },
  { value: 'route-options', label: t('dns.rules.actionTypes.routeOptions') },
  { value: 'reject', label: t('dns.rules.actionTypes.reject') },
  // `predefined` is the only way to answer a query yourself in 1.12 — there is
  // no "block" DNS server type. Without it, blocking a domain at the DNS layer
  // is unreachable from this UI.
  { value: 'predefined', label: t('dns.rules.actionTypes.predefined') },
])

// The rcode vocabulary, the reject methods and the DNS server picker all used to
// be built here. They now live in schemas/dnsRuleActionFields.ts (the first two,
// as curated `select` options) and in SchemaFieldControl.vue (the third, as the
// `dns-server` control) — next to the field definitions they belong to, and
// reachable from any future form that needs them.

const ruleSetOptions = computed(() => {
  const options: { value: string; label: string }[] = []
  if (ruleSets.value) {
    ruleSets.value.forEach(ruleSet => {
      if (ruleSet.tag) {
        const type = (ruleSet as any).type || 'local'
        const format = (ruleSet as any).format || 'source'
        options.push({
          value: ruleSet.tag,
          label: t('dns.rules.ruleSetLabel', { tag: ruleSet.tag, type, format })
        })
      }
    })
  }
  return options
})


/**
 * A blank rule. Only the conditions are seeded, as empty arrays for the Chips
 * v-model; the action's own fields come from `applyActionDefaults`, so their
 * defaults live in the curation file rather than being restated here.
 */
function emptyRuleForm(): Record<string, any> {
  return applyActionDefaults(
    {
      rule_set: [] as string[],
      domain: [] as string[],
      domain_suffix: [] as string[],
      domain_keyword: [] as string[],
      geosite: [] as string[],
    },
    'route',
  )
}

const openAddRuleModal = () => {
  isEditMode.value = false
  currentRule.value = emptyRuleForm()
  conditionsKey.value++
  showRuleModal.value = true
}

const openEditRuleModal = (index: number, rule: DNSRule) => {
  isEditMode.value = true
  editingIndex.value = index

  const raw = rule as Record<string, unknown>

  // The whole rule, not a fixed subset. Every key survives — including ones this
  // form has no control for, and `type`/`mode`/`rules` on a logical rule.
  //
  // Note what is NOT done here: no default is seeded. `applyActionDefaults` runs
  // on create and on an action change, never on open. Seeding one here is
  // exactly how a predefined rule with no `rcode` — which means NOERROR — used
  // to be silently rewritten to NXDOMAIN by opening it and pressing Update.
  const loaded: Record<string, any> = { ...raw }

  // sing-box accepts a scalar or an array for these; normalize to an array. This
  // once kept only the first entry, so opening and saving a rule that matched two
  // rule sets silently dropped one of them.
  for (const key of CONDITION_FIELDS) {
    loaded[key] = toArrayField(loaded[key])
  }
  for (const key of ACTION_LIST_FIELDS) {
    if (key in loaded) loaded[key] = toArrayField(loaded[key])
  }

  currentRule.value = loaded
  conditionsKey.value++
  showRuleModal.value = true
}

const closeRuleModal = () => {
  showRuleModal.value = false
  currentRule.value = emptyRuleForm()
}

const handleSaveRule = async () => {
  // Keep only the fields valid for the selected action. sing-box strict-parses
  // DNS rules and rejects any field that does not belong to the action —
  // `method` is reject-only ("unknown field method" on a route rule).
  //
  // This replaces a hand-written switch that had to be updated by hand for every
  // action, and which missed `server` on the route -> route-options edge.
  // pruneForeignFields derives the same answer from the generated inventory, and
  // — unlike the switch and the EDITED_FIELDS allowlist it worked with — keeps
  // every key no action owns: all the match conditions, and `type`/`mode`/`rules`
  // on a logical rule.
  const processedRule: Record<string, unknown> = pruneForeignFields(
    currentRule.value,
    currentAction.value,
  )

  // An empty value is the absence of a setting, not a setting of "". Writing it
  // through would persist it into config.json as an explicit one — and for a
  // `false` switch, sing-box's omitempty means absent and false already
  // serialize identically.
  Object.keys(processedRule).forEach((key) => {
    const v = processedRule[key]
    if (v === undefined || v === null || v === '' || (Array.isArray(v) && v.length === 0)) {
      delete processedRule[key]
    }
  })

  // Three constraints sing-box only enforces while decoding, which `sing-box
  // check` reports as an opaque upstream string. Caught here so the operator
  // sees which field is wrong. See validateDNSRuleAction for each one's source.
  const invalid = validateDNSRuleAction(processedRule)
  if (invalid) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t(invalid),
      life: 4000,
    })
    return
  }

  loading.value = true
  try {
    if (isEditMode.value) {
      await dnsService.updateDNSRule(editingIndex.value, processedRule)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('dns.rules.toast.updatedOk'),
        life: 3000
      })
    } else {
      await dnsService.addDNSRule(processedRule)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('dns.rules.toast.addedOk'),
        life: 3000
      })
    }
    await fetchDNSRules()
    closeRuleModal()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('dns.rules.toast.saveFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const openDeleteConfirm = (index: number) => {
  deletingIndex.value = index
  showDeleteConfirm.value = true
}

const closeDeleteConfirm = () => {
  showDeleteConfirm.value = false
  deletingIndex.value = -1
}

const handleDeleteRule = async () => {
  if (deletingIndex.value === -1) return
  loading.value = true
  try {
    await dnsService.deleteDNSRule(deletingIndex.value)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('dns.rules.toast.deletedOk'),
      life: 3000
    })
    await fetchDNSRules()
    closeDeleteConfirm()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('dns.rules.toast.deleteFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const getRuleConditionsSummary = (rule: any) => {
  const conditions = []

  // Handle rule_set - could be array, string, or undefined
  if (rule.rule_set) {
    if (Array.isArray(rule.rule_set) && rule.rule_set.length > 0) {
      conditions.push(t('dns.rules.summary.ruleSet', { value: rule.rule_set.join(', ') }))
    } else if (typeof rule.rule_set === 'string' && rule.rule_set.trim()) {
      conditions.push(t('dns.rules.summary.ruleSet', { value: rule.rule_set }))
    }
  }

  // Handle arrays for other fields
  if (Array.isArray(rule.domain) && rule.domain.length) {
    conditions.push(t('dns.rules.summary.domain', { value: rule.domain.join(', ') }))
  }
  if (Array.isArray(rule.domain_suffix) && rule.domain_suffix.length) {
    conditions.push(t('dns.rules.summary.suffix', { value: rule.domain_suffix.join(', ') }))
  }
  if (Array.isArray(rule.domain_keyword) && rule.domain_keyword.length) {
    conditions.push(t('dns.rules.summary.keyword', { value: rule.domain_keyword.join(', ') }))
  }
  if (Array.isArray(rule.geosite) && rule.geosite.length) {
    conditions.push(t('dns.rules.summary.geosite', { value: rule.geosite.join(', ') }))
  }

  return conditions.length > 0 ? conditions.join(' | ') : t('dns.rules.summary.none')
}

/* ── Reorder ────────────────────────────────────────────────────────────────
 *
 * DNS rules match top-down exactly as route rules do, so their order is policy
 * too — a `geosite-cn -> local` rule under a catch-all never runs. Same
 * composable, same live-sorting drag; only the endpoint differs.
 */
const reorder = useDragReorder(dnsRules, persistOrder)

/**
 * Sends the permutation. The table already shows the result, so this only has
 * to handle the failure: refetch, which restores whatever the config actually
 * says rather than leaving the rows lying about it.
 */
async function persistOrder(order: number[]) {
  try {
    await dnsService.reorderDNSRules(order)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('dns.rules.toast.reordered'),
      life: 2000,
    })
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('dns.rules.toast.reorderFailed'),
      life: 3000,
    })
    await fetchDNSRules()
    throw err
  }
}

// Load data on mount
onMounted(() => {
  fetchDNSRules()
  routeStore.fetchRuleSets() // Fetch shared rule sets
  dnsStore.fetchDNSServers() // Fetch shared DNS servers
})
</script>

<template>
  <div>
    <div class="flex justify-end items-center gap-2 mb-2">
      <!-- Reorder is a MODE, and a BATCHED one: turning it on swaps the index
           column for a drag handle and hides edit/delete, and nothing is
           written until Save — a five-row cleanup is one config write, not
           five. -->
      <template v-if="reorder.enabled.value">
        <Button @click="reorder.cancel()" variant="secondary" :disabled="reorder.saving.value">
          {{ $t('dns.rules.reorder.cancel') }}
        </Button>
        <Button @click="reorder.save()" variant="primary" :disabled="reorder.saving.value">
          <ArrowsUpDownIcon class="h-5 w-5 mr-2" />
          {{ reorder.dirty.value ? $t('dns.rules.reorder.save') : $t('dns.rules.reorder.done') }}
        </Button>
      </template>

      <Button
        v-else-if="dnsRules.length > 1"
        @click="reorder.start()"
        variant="secondary"
      >
        <ArrowsUpDownIcon class="h-5 w-5 mr-2" />
        {{ $t('dns.rules.reorder.start') }}
      </Button>

      <Button @click="openAddRuleModal" variant="primary">
        <PlusIcon class="h-5 w-5 mr-2" />
        {{ $t('dns.rules.add') }}
      </Button>
    </div>

    <p v-if="reorder.enabled.value" class="mb-2 text-xs text-gray-500 dark:text-gray-400">
      {{ $t('dns.rules.reorder.hint') }}
      <span v-if="reorder.dirty.value" class="text-amber-600 dark:text-amber-400 font-medium">
        {{ $t('dns.rules.reorder.dirtyHint') }}
      </span>
    </p>

    <!-- DNS Rules Table -->
    <div class="bg-white dark:bg-slate-800 rounded-surface shadow-surface overflow-hidden">
      <Table :loading="loading && dnsRules.length === 0" :empty="dnsRules.length === 0" transition>
        <template #empty>
          <p class="text-gray-500 dark:text-gray-500 mb-3">{{ $t('dns.rules.empty') }}</p>
          <Button @click="openAddRuleModal" variant="primary" size="sm">
            <PlusIcon class="h-4 w-4 mr-1.5" />
            {{ $t('dns.rules.addFirst') }}
          </Button>
        </template>

        <template #head>
          <th>{{ $t('dns.rules.table.index') }}</th>
          <th>{{ $t('dns.rules.table.action') }}</th>
          <th>{{ $t('dns.rules.table.server') }}</th>
          <th>{{ $t('dns.rules.table.conditions') }}</th>
          <th class="col-actions">{{ $t('dns.rules.table.actions') }}</th>
        </template>

        <!-- Keyed by rule identity, not by index — index keys would make every
             reorder a text patch, which cannot animate. `rowAttrs` carries the
             drag handlers; it is empty off reorder mode. -->
        <tr
          v-for="(rule, index) in dnsRules"
          :key="reorder.keyAt(index)"
          v-bind="reorder.rowAttrs(index)"
        >
          <td class="text-gray-900 dark:text-gray-100">
            <!-- The handle takes the index column's place rather than adding a
                 column: the position number is what the drag is changing, and
                 a table that grows a column on mode change shifts every cell. -->
            <button
              v-if="reorder.enabled.value"
              v-bind="reorder.handleAttrs(index)"
              class="list-drag-handle"
              :aria-label="$t('dns.rules.reorder.handle', { position: index + 1 })"
            >
              <Bars3Icon class="w-4 h-4" />
            </button>
            <template v-else>{{ index + 1 }}</template>
          </td>
          <td>
            <Badge :variant="(rule as any).action === 'reject' ? 'warning' : 'primary'">
              {{ (rule as any).action || 'route' }}
            </Badge>
          </td>
          <td>
            <!-- A predefined rule has no server; showing "-" hid the one
                 thing that matters about it, the answer it returns. -->
            <div v-if="(rule as any).action === 'predefined'" class="font-mono text-gray-600 dark:text-gray-400">
              {{ predefinedSummary(rule) }}
            </div>
            <div v-else class="text-gray-900 dark:text-gray-100">{{ (rule as any).server || '-' }}</div>
          </td>
          <td>
            <div class="text-gray-900 dark:text-gray-100 truncate max-w-md" :title="getRuleConditionsSummary(rule)">
              {{ getRuleConditionsSummary(rule) }}
            </div>
          </td>
          <td class="col-actions font-medium">
            <!-- Hidden in reorder mode on purpose: both are index-addressed,
                 and the indices are exactly what is in motion. -->
            <div v-if="!reorder.enabled.value" class="flex items-center justify-end gap-1">
              <Button @click="openEditRuleModal(index, rule)" variant="ghost" size="sm" action>
                <PencilIcon class="h-4 w-4" />
              </Button>
              <Button @click="openDeleteConfirm(index)" variant="ghost" size="sm" action class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300">
                <TrashIcon class="h-4 w-4" />
              </Button>
            </div>
          </td>
        </tr>
      </Table>
    </div>

    <!-- Add/Edit DNS Rule Modal -->
    <Modal
      :model-value="showRuleModal"
      @update:model-value="(v) => { if (!v) closeRuleModal() }"
      :title="isEditMode ? $t('dns.rules.modal.edit') : $t('dns.rules.modal.add')"
      size="lg"
      show-close
    >
      <div class="space-y-4">
        <!--
          The rule restated as sing-box will run it. The form shows which fields
          exist; this says what they DO. Same component the route rule dialog
          uses — only the outcome phrase and the condition list differ.
        -->
        <RuleFlowPreview
          :rule="currentRule"
          :condition-keys="conditionKeys"
          label-prefix="dns.rules.form.fields"
          :outcome="flowOutcome"
          :continues-matching="flowContinues"
          :catch-all="!flowContinues"
        />

        <!--
          WHEN / THEN, in that order.

          The dialog used to lead with the action. sing-box tests the conditions
          first and only then applies the action, so reading top-to-bottom now
          follows the query through the rule.
        -->
        <section class="space-y-3">
          <div class="flex items-baseline gap-2">
            <span
              class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-pill bg-primary-100 dark:bg-primary-900/40 text-[11px] font-semibold text-primary-700 dark:text-primary-300"
              >1</span
            >
            <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
              {{ $t('dns.rules.form.whenHeading') }}
            </h4>
            <span class="text-xs text-gray-500 dark:text-gray-400">
              {{ $t('dns.rules.form.whenHint') }}
            </span>
          </div>

          <!-- Remounted per modal open (:key) so the collapse state is decided
               from the rule as loaded, not from the previous edit. -->
          <DNSRuleConditions
            :key="conditionsKey"
            v-model:rule-set="currentRule.rule_set"
            v-model:domain="currentRule.domain"
            v-model:domain-suffix="currentRule.domain_suffix"
            v-model:domain-keyword="currentRule.domain_keyword"
            v-model:geosite="currentRule.geosite"
            :rule-set-options="ruleSetOptions"
          />
        </section>

        <section class="space-y-3 border-t border-gray-200 dark:border-gray-700 pt-4">
          <div class="flex items-baseline gap-2">
            <span
              class="inline-flex items-center justify-center h-5 min-w-5 px-1.5 rounded-pill bg-primary-100 dark:bg-primary-900/40 text-[11px] font-semibold text-primary-700 dark:text-primary-300"
              >2</span
            >
            <h4 class="text-sm font-semibold text-gray-900 dark:text-gray-100">
              {{ $t('dns.rules.form.thenHeading') }}
            </h4>
          </div>

          <!--
            The action select stays outside the schema editor, exactly as the
            type select does in DNSServers.vue: it is the discriminator that
            CHOOSES the field set, so it cannot be one of the fields.
          -->
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.rules.form.action') }}</label>
            <Select
              class="w-full"
              optionLabel="label"
              optionValue="value"
              :modelValue="currentAction"
              :options="actionTypes"
              @update:modelValue="changeAction"
            />
          </div>

          <!--
            Everything the action owns, from the generated inventory. Replaces
            three hand-written v-if blocks that covered 3 of 15 fields and left
            `route-options` with an empty form body.

            :key remounts on action change so the previous action's added-field
            state does not leak — same reason conditionsKey exists above.
          -->
          <SchemaFieldsEditor
            :key="currentAction"
            v-model="currentRule"
            :fields="actionFields"
          />
        </section>
      </div>

      <template #footer>
        <Button @click="closeRuleModal" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleSaveRule" variant="primary" :disabled="loading">
          {{ isEditMode ? $t('common.update') : $t('common.add') }}
        </Button>
      </template>
    </Modal>

    <!-- Delete Confirmation Modal -->
    <Modal
      :model-value="showDeleteConfirm"
      @update:model-value="(v) => { if (!v) closeDeleteConfirm() }"
      :title="$t('dns.rules.del.title')"
      size="sm"
      show-close
    >
      <p class="text-gray-700 dark:text-gray-300">
        {{ $t('dns.rules.del.confirm', { index: deletingIndex + 1 }) }}
      </p>

      <template #footer>
        <Button @click="closeDeleteConfirm" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleDeleteRule" variant="danger" :disabled="loading">
          {{ $t('common.delete') }}
        </Button>
      </template>
    </Modal>
  </div>
</template>
