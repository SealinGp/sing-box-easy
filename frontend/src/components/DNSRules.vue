<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DNSRule } from '../types/api'
import Button from './Button.vue'
import Modal from './Modal.vue'
import { Select } from '../volt'
import Badge from './Badge.vue'
import DNSRuleConditions from './DNSRuleConditions.vue'
import { PlusIcon, PencilIcon, TrashIcon } from '@heroicons/vue/24/outline'
import {  dnsService } from '../services'
import { useToast } from 'primevue'
import { useDNSStore } from '../stores/dns'
import { useRouteStore } from '../stores/route'
import { dnsServerOptionLabel } from '../utils/dnsServerLabel'
import { storeToRefs } from 'pinia'

const toast = useToast()
const { t } = useI18n()
const dnsStore = useDNSStore()
const routeStore = useRouteStore()
const { dnsServers } = storeToRefs(dnsStore)
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
// Form model uses arrays for list fields to match the Chips v-model shape and
// the sing-box wire format. sing-box also accepts a scalar — openEditRuleModal
// coerces scalar → [value] on load, and handleSaveRule strips empties.
const currentRule = ref<any>({
  action: 'route',
  server: '',
  method: 'default',
  rcode: 'NXDOMAIN',
  rule_set: '',
  domain: [] as string[],
  domain_suffix: [] as string[],
  domain_keyword: [] as string[],
  geosite: [] as string[],
})

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

// Verified against sing-box 1.12.12: uppercase only, and this is the whole set.
const rcodeOptions = [
  { value: 'NXDOMAIN', label: 'NXDOMAIN' },
  { value: 'NOERROR', label: 'NOERROR' },
  { value: 'SERVFAIL', label: 'SERVFAIL' },
  { value: 'REFUSED', label: 'REFUSED' },
  { value: 'FORMERR', label: 'FORMERR' },
  { value: 'NOTIMP', label: 'NOTIMP' },
]

const serverOptions = computed(() => {
  const options = [
    { value: '', label: t('dns.rules.serverSelect') }
  ]
  if (dnsServers.value) {
    dnsServers.value.forEach(server => {
      // Label carries the address + transport, not just the tag — picking the
      // right DNS server for a rule is impossible from a bare tag.
      options.push({ value: server.tag, label: dnsServerOptionLabel(server) })
    })
  }
  return options
})

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

// sing-box accepts exactly two reject methods (constant/rule.go:40-41).
// The list previously also offered success/refused/nxdomain, which sing-box
// rejects outright — "unknown reject method: nxdomain" — so the modal could
// save a config that passed panel validation and then refused to start.
// To answer with a specific rcode, use the `predefined` action instead.
const rejectMethods = computed(() => [
  { value: 'default', label: t('dns.rules.rejectMethods.default') },
  { value: 'drop', label: t('dns.rules.rejectMethods.drop') },
])

function emptyRuleForm() {
  return {
    action: 'route',
    server: '',
    method: 'default',
    rcode: 'NXDOMAIN',
    rule_set: '',
    domain: [] as string[],
    domain_suffix: [] as string[],
    domain_keyword: [] as string[],
    geosite: [] as string[],
  }
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
  currentRule.value = {
    action: (raw.action as string) || 'route',
    server: (raw.server as string) || '',
    method: (raw.method as string) || 'default',
    rcode: (raw.rcode as string) || 'NXDOMAIN',
    // rule_set is a single-select in the UI; the backend allows array — pick the first.
    rule_set: Array.isArray(raw.rule_set)
      ? ((raw.rule_set as string[])[0] || '')
      : ((raw.rule_set as string) || ''),
    domain: toArrayField(raw.domain),
    domain_suffix: toArrayField(raw.domain_suffix),
    domain_keyword: toArrayField(raw.domain_keyword),
    geosite: toArrayField(raw.geosite),
  }
  conditionsKey.value++
  showRuleModal.value = true
}

const closeRuleModal = () => {
  showRuleModal.value = false
  currentRule.value = emptyRuleForm()
}

const handleSaveRule = async () => {
  // currentRule.* list fields are already string[] from Chips.
  // Strip empty arrays so we don't send `{ domain: [] }` to the backend.
  const processedRule: Record<string, unknown> = {
    ...currentRule.value,
    rule_set: currentRule.value.rule_set ? [currentRule.value.rule_set] : undefined,
    domain: currentRule.value.domain.length ? currentRule.value.domain : undefined,
    domain_suffix: currentRule.value.domain_suffix.length ? currentRule.value.domain_suffix : undefined,
    domain_keyword: currentRule.value.domain_keyword.length ? currentRule.value.domain_keyword : undefined,
    geosite: currentRule.value.geosite.length ? currentRule.value.geosite : undefined,
  }

  Object.keys(processedRule).forEach((key) => {
    const v = processedRule[key]
    if (v === undefined || v === '' || (Array.isArray(v) && v.length === 0)) {
      delete processedRule[key]
    }
  })

  // Keep only the fields valid for the selected action. sing-box strict-parses
  // DNS rules and rejects any field that does not belong to the action — e.g.
  // `method` is reject-only ("unknown field method" on a route rule).
  switch (processedRule.action) {
    case 'reject':
      delete processedRule.server
      delete processedRule.rcode
      break
    case 'predefined':
      // Answers the query directly; it neither routes to a server nor rejects.
      delete processedRule.server
      delete processedRule.method
      delete processedRule.no_drop
      break
    default: // route, route-options
      delete processedRule.method
      delete processedRule.no_drop
      delete processedRule.rcode
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

// Load data on mount
onMounted(() => {
  fetchDNSRules()
  routeStore.fetchRuleSets() // Fetch shared rule sets
  dnsStore.fetchDNSServers() // Fetch shared DNS servers
})
</script>

<template>
  <div>
    <div class="flex justify-end mb-2">
      <Button @click="openAddRuleModal" variant="primary">
        <PlusIcon class="h-5 w-5 mr-2" />
        {{ $t('dns.rules.add') }}
      </Button>
    </div>

    <!-- DNS Rules Table -->
    <div class="bg-white dark:bg-slate-800 rounded-surface shadow dark:shadow-float dark:shadow-slate-700/50 overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ $t('dns.rules.heading') }}</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          {{ $t('dns.rules.subheading') }}
        </p>
      </div>

      <div v-if="loading && dnsRules.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <div v-else-if="dnsRules.length === 0" class="text-center py-12">
        <p class="text-gray-500 dark:text-gray-500 mb-4">{{ $t('dns.rules.empty') }}</p>
        <Button @click="openAddRuleModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          {{ $t('dns.rules.addFirst') }}
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.rules.table.index') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.rules.table.action') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.rules.table.server') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.rules.table.conditions') }}</th>
              <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.rules.table.actions') }}</th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-for="(rule, index) in dnsRules" :key="index" class="hover:bg-gray-50 dark:hover:bg-gray-700">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-gray-100">{{ index + 1 }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge :variant="(rule as any).action === 'reject' ? 'warning' : 'primary'">
                  {{ (rule as any).action || 'route' }}
                </Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <!-- A predefined rule has no server; showing "-" hid the one
                     thing that matters about it, the answer it returns. -->
                <div v-if="(rule as any).action === 'predefined'" class="text-sm font-mono text-gray-600 dark:text-gray-400">
                  {{ (rule as any).rcode || 'NXDOMAIN' }}
                </div>
                <div v-else class="text-sm text-gray-900 dark:text-gray-100">{{ (rule as any).server || '-' }}</div>
              </td>
              <td class="px-6 py-4">
                <div class="text-sm text-gray-900 dark:text-gray-100 truncate max-w-md" :title="getRuleConditionsSummary(rule)">
                  {{ getRuleConditionsSummary(rule) }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="dns-rule-table-actions flex items-center justify-end gap-2">
                  <Button @click="openEditRuleModal(index, rule)" variant="ghost" size="sm" action>
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button @click="openDeleteConfirm(index)" variant="ghost" size="sm" action class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300">
                    <TrashIcon class="h-4 w-4" />
                  </Button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
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
        <!-- Action -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.rules.form.action') }}</label>
          <Select class="w-full" optionLabel="label" optionValue="value" v-model="currentRule.action" :options="actionTypes" />
        </div>

        <!-- Server (for route action) -->
        <div v-if="currentRule.action === 'route'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.rules.form.server') }}</label>
          <Select class="w-full" optionLabel="label" optionValue="value" v-model="currentRule.server" :options="serverOptions" />
        </div>

        <!-- Response code (for predefined action) -->
        <div v-if="currentRule.action === 'predefined'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.rules.form.rcode') }}</label>
          <Select class="w-full" optionLabel="label" optionValue="value" v-model="currentRule.rcode" :options="rcodeOptions" />
          <p class="mt-1 text-xs text-gray-500">{{ $t('dns.rules.form.rcodeHelp') }}</p>
        </div>

        <!-- Reject Method (for reject action) -->
        <div v-if="currentRule.action === 'reject'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.rules.form.rejectMethod') }}</label>
          <Select class="w-full" optionLabel="label" optionValue="value" v-model="currentRule.method" :options="rejectMethods" />
          <p class="mt-1 text-xs text-gray-500">{{ $t('dns.rules.form.rejectMethodHelp') }}</p>
        </div>

        <!-- Rule Conditions -->
        <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
          <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">{{ $t('dns.rules.form.conditionsHeading') }}</h4>

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
        </div>
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
