<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  TransitionRoot,
  TransitionChild,
  Dialog,
  DialogPanel,
  DialogTitle,
} from '@headlessui/vue'
import type { DNSRule } from '../types/api'
import Button from './Button.vue'
import Select from './Select.vue'
import Badge from './Badge.vue'
import { Chips } from '../volt'
import { PlusIcon, PencilIcon, TrashIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import {  dnsService } from '../services'
import { useToast } from 'primevue'
import { useDNSStore } from '../stores/dns'
import { useRouteStore } from '../stores/route'
import { storeToRefs } from 'pinia'

const toast = useToast()
const dnsStore = useDNSStore()
const routeStore = useRouteStore()
const { dnsServers } = storeToRefs(dnsStore)
const { ruleSets } = storeToRefs(routeStore)

// Local state for DNS rules
const loading = ref(false)
const dnsRules = ref<DNSRule[]>([])

// Modal state
const showRuleModal = ref(false)
const isEditMode = ref(false)
const editingIndex = ref(-1)
// Form model uses arrays for list fields to match the Chips v-model shape and
// the sing-box wire format. sing-box also accepts a scalar — openEditRuleModal
// coerces scalar → [value] on load, and handleSaveRule strips empties.
const currentRule = ref<any>({
  action: 'route',
  server: '',
  method: 'default',
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
      summary: 'Error',
      detail: err.message || 'Failed to fetch DNS rules',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}


const actionTypes = [
  { value: 'route', label: 'Route - Forward to specified DNS server' },
  { value: 'route-options', label: 'Route Options - Set route options without changing server' },
  { value: 'reject', label: 'Reject - Reject DNS requests with specific method' },
]

const serverOptions = computed(() => {
  const options = [
    { value: '', label: 'Select a server' }
  ]
  if (dnsServers.value) {
    dnsServers.value.forEach(server => {
      options.push({ value: server.tag, label: server.tag })
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
          label: `${ruleSet.tag} (${type} - ${format})`
        })
      }
    })
  }
  return options
})

const rejectMethods = [
  { value: 'default', label: 'Default - Return empty response' },
  { value: 'success', label: 'Success - Return success response' },
  { value: 'refused', label: 'Refused - Return refused response' },
  { value: 'nxdomain', label: 'NXDOMAIN - Domain does not exist' },
]

function emptyRuleForm() {
  return {
    action: 'route',
    server: '',
    method: 'default',
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
    // rule_set is a single-select in the UI; the backend allows array — pick the first.
    rule_set: Array.isArray(raw.rule_set)
      ? ((raw.rule_set as string[])[0] || '')
      : ((raw.rule_set as string) || ''),
    domain: toArrayField(raw.domain),
    domain_suffix: toArrayField(raw.domain_suffix),
    domain_keyword: toArrayField(raw.domain_keyword),
    geosite: toArrayField(raw.geosite),
  }
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

  loading.value = true
  try {
    if (isEditMode.value) {
      await dnsService.updateDNSRule(editingIndex.value, processedRule)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'DNS rule updated successfully',
        life: 3000
      })
    } else {
      await dnsService.addDNSRule(processedRule)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'DNS rule added successfully',
        life: 3000
      })
    }
    await fetchDNSRules()
    closeRuleModal()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to save DNS rule',
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
      summary: 'Success',
      detail: 'DNS rule deleted successfully',
      life: 3000
    })
    await fetchDNSRules()
    closeDeleteConfirm()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to delete DNS rule',
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
      conditions.push(`Rule Set: ${rule.rule_set.join(', ')}`)
    } else if (typeof rule.rule_set === 'string' && rule.rule_set.trim()) {
      conditions.push(`Rule Set: ${rule.rule_set}`)
    }
  }

  // Handle arrays for other fields
  if (Array.isArray(rule.domain) && rule.domain.length) {
    conditions.push(`Domain: ${rule.domain.join(', ')}`)
  }
  if (Array.isArray(rule.domain_suffix) && rule.domain_suffix.length) {
    conditions.push(`Suffix: ${rule.domain_suffix.join(', ')}`)
  }
  if (Array.isArray(rule.domain_keyword) && rule.domain_keyword.length) {
    conditions.push(`Keyword: ${rule.domain_keyword.join(', ')}`)
  }
  if (Array.isArray(rule.geosite) && rule.geosite.length) {
    conditions.push(`GeoSite: ${rule.geosite.join(', ')}`)
  }

  return conditions.length > 0 ? conditions.join(' | ') : 'No conditions'
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
        Add DNS Rule
      </Button>
    </div>

    <!-- DNS Rules Table -->
    <div class="bg-white dark:bg-slate-800 rounded-lg shadow dark:shadow-xl dark:shadow-slate-700/50 overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">DNS Rules</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          Rules are processed in order. First match wins.
        </p>
      </div>

      <div v-if="loading && dnsRules.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
      </div>

      <div v-else-if="dnsRules.length === 0" class="text-center py-12">
        <p class="text-gray-500 dark:text-gray-500 mb-4">No DNS rules configured</p>
        <Button @click="openAddRuleModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          Add Your First DNS Rule
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">#</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Action</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Server</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Conditions</th>
              <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Actions</th>
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
                <div class="text-sm text-gray-900 dark:text-gray-100">{{ (rule as any).server || '-' }}</div>
              </td>
              <td class="px-6 py-4">
                <div class="text-sm text-gray-900 dark:text-gray-100 truncate max-w-md" :title="getRuleConditionsSummary(rule)">
                  {{ getRuleConditionsSummary(rule) }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="flex items-center justify-end gap-2">
                  <Button @click="openEditRuleModal(index, rule)" variant="ghost" size="sm">
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button @click="openDeleteConfirm(index)" variant="ghost" size="sm" class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300">
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
    <TransitionRoot appear :show="showRuleModal" as="template">
      <Dialog as="div" @close="closeRuleModal" class="relative z-50">
        <TransitionChild
          as="template"
          enter="duration-300 ease-out"
          enter-from="opacity-0"
          enter-to="opacity-100"
          leave="duration-200 ease-in"
          leave-from="opacity-100"
          leave-to="opacity-0"
        >
          <div class="fixed inset-0 bg-black/50 dark:bg-black/70" />
        </TransitionChild>

        <div class="fixed inset-0 overflow-y-auto">
          <div class="flex min-h-full items-center justify-center p-4 text-center">
            <TransitionChild
              as="template"
              enter="duration-300 ease-out"
              enter-from="opacity-0 scale-95"
              enter-to="opacity-100 scale-100"
              leave="duration-200 ease-in"
              leave-from="opacity-100 scale-100"
              leave-to="opacity-0 scale-95"
            >
              <DialogPanel class="w-full max-w-2xl transform overflow-hidden rounded-lg bg-white dark:bg-slate-800 p-6 text-left align-middle shadow-xl transition-all">
                <div class="flex items-center justify-between mb-4">
                  <DialogTitle as="h3" class="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    {{ isEditMode ? 'Edit DNS Rule' : 'Add DNS Rule' }}
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeRuleModal"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

                <div class="space-y-4">
                  <!-- Action -->
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Action *</label>
                    <Select v-model="currentRule.action" :options="actionTypes" />
                  </div>

                  <!-- Server (for route action) -->
                  <div v-if="currentRule.action === 'route'">
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">DNS Server *</label>
                    <Select v-model="currentRule.server" :options="serverOptions" />
                  </div>

                  <!-- Reject Method (for reject action) -->
                  <div v-if="currentRule.action === 'reject'">
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Reject Method</label>
                    <Select v-model="currentRule.method" :options="rejectMethods" />
                    <p class="mt-1 text-xs text-gray-500">Method to reject DNS requests</p>
                  </div>

                  <!-- Rule Conditions -->
                  <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
                    <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Conditions (at least one required)</h4>

                    <div class="space-y-3">
                      <!-- Rule Set -->
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Rule Set</label>
                        <Select
                          v-model="currentRule.rule_set"
                          :options="ruleSetOptions"
                          :searchable="true"
                          :clearable="true"
                          placeholder="Select or search a rule set"
                          search-placeholder="Type to filter rule sets..."
                          no-options-text="No matching rule sets found"
                        />
                        <p class="mt-1 text-xs text-gray-500">Use a predefined rule set for this DNS rule</p>
                      </div>

                      <!-- Domain -->
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain</label>
                        <Chips
                          v-model="currentRule.domain"
                          placeholder="Add domains (press Enter after each)"
                          class="w-full"
                        />
                        <p class="mt-1 text-xs text-gray-500">Exact domain match</p>
                      </div>

                      <!-- Domain Suffix -->
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain Suffix</label>
                        <Chips
                          v-model="currentRule.domain_suffix"
                          placeholder="Add suffixes (.example.com)"
                          class="w-full"
                        />
                        <p class="mt-1 text-xs text-gray-500">Matches domain and all subdomains</p>
                      </div>

                      <!-- Domain Keyword -->
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain Keyword</label>
                        <Chips
                          v-model="currentRule.domain_keyword"
                          placeholder="Add keywords"
                          class="w-full"
                        />
                        <p class="mt-1 text-xs text-gray-500">Domain contains keyword</p>
                      </div>

                      <!-- GeoSite -->
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">GeoSite</label>
                        <Chips
                          v-model="currentRule.geosite"
                          placeholder="Add geosite tags (e.g. google, netflix, cn)"
                          class="w-full"
                        />
                        <p class="mt-1 text-xs text-gray-500">Use geosite database</p>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="mt-6 flex justify-end gap-3">
                  <Button @click="closeRuleModal" variant="secondary">Cancel</Button>
                  <Button @click="handleSaveRule" variant="primary" :disabled="loading">
                    {{ isEditMode ? 'Update' : 'Add' }}
                  </Button>
                </div>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>

    <!-- Delete Confirmation Modal -->
    <TransitionRoot appear :show="showDeleteConfirm" as="template">
      <Dialog as="div" @close="closeDeleteConfirm" class="relative z-50">
        <TransitionChild
          as="template"
          enter="duration-300 ease-out"
          enter-from="opacity-0"
          enter-to="opacity-100"
          leave="duration-200 ease-in"
          leave-from="opacity-100"
          leave-to="opacity-0"
        >
          <div class="fixed inset-0 bg-black/50 dark:bg-black/70" />
        </TransitionChild>

        <div class="fixed inset-0 overflow-y-auto">
          <div class="flex min-h-full items-center justify-center p-4 text-center">
            <TransitionChild
              as="template"
              enter="duration-300 ease-out"
              enter-from="opacity-0 scale-95"
              enter-to="opacity-100 scale-100"
              leave="duration-200 ease-in"
              leave-from="opacity-100 scale-100"
              leave-to="opacity-0 scale-95"
            >
              <DialogPanel class="w-full max-w-md transform overflow-hidden rounded-lg bg-white dark:bg-slate-800 p-6 text-left align-middle shadow-xl transition-all">
                <div class="flex items-center justify-between mb-4">
                  <DialogTitle as="h3" class="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    Delete DNS Rule
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeDeleteConfirm"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

                <p class="text-gray-700 dark:text-gray-300">
                  Are you sure you want to delete rule #{{ deletingIndex + 1 }}?
                  This action cannot be undone.
                </p>

                <div class="mt-6 flex justify-end gap-3">
                  <Button @click="closeDeleteConfirm" variant="secondary">Cancel</Button>
                  <Button @click="handleDeleteRule" variant="danger" :disabled="loading">
                    Delete
                  </Button>
                </div>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>
  </div>
</template>
