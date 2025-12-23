<script setup lang="ts">
import { ref } from 'vue'
import {
  TransitionRoot,
  TransitionChild,
  Dialog,
  DialogPanel,
  DialogTitle,
} from '@headlessui/vue'
import type { DNSRule, DNSServer } from '../types/api'
import Button from './Button.vue'
import Input from './Input.vue'
import Badge from './Badge.vue'
import { PlusIcon, PencilIcon, TrashIcon, XMarkIcon } from '@heroicons/vue/24/outline'

const props = defineProps<{
  rules: DNSRule[]
  servers: DNSServer[]
  loading: boolean
}>()

const emit = defineEmits<{
  addRule: [rule: DNSRule]
  updateRule: [index: number, rule: DNSRule]
  deleteRule: [index: number]
}>()

// Modal state
const showRuleModal = ref(false)
const isEditMode = ref(false)
const editingIndex = ref(-1)
const currentRule = ref<any>({
  action: 'route',
  server: '',
  domain: [],
  domain_suffix: [],
})

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingIndex = ref(-1)

const actionTypes = [
  { value: 'route', label: 'Route to Server' },
  { value: 'return', label: 'Return' },
  { value: 'reject', label: 'Reject' },
]

const openAddRuleModal = () => {
  isEditMode.value = false
  currentRule.value = {
    action: 'route',
    server: '',
    domain: [],
    domain_suffix: [],
  }
  showRuleModal.value = true
}

const openEditRuleModal = (index: number, rule: DNSRule) => {
  isEditMode.value = true
  editingIndex.value = index
  currentRule.value = { ...rule }
  showRuleModal.value = true
}

const closeRuleModal = () => {
  showRuleModal.value = false
  currentRule.value = {
    action: 'route',
    server: '',
    domain: [],
    domain_suffix: [],
  }
}

const handleSaveRule = () => {
  if (isEditMode.value) {
    emit('updateRule', editingIndex.value, currentRule.value)
  } else {
    emit('addRule', currentRule.value)
  }
  closeRuleModal()
}

const openDeleteConfirm = (index: number) => {
  deletingIndex.value = index
  showDeleteConfirm.value = true
}

const closeDeleteConfirm = () => {
  showDeleteConfirm.value = false
  deletingIndex.value = -1
}

const handleDeleteRule = () => {
  if (deletingIndex.value === -1) return
  emit('deleteRule', deletingIndex.value)
  closeDeleteConfirm()
}

const getRuleConditionsSummary = (rule: any) => {
  const conditions = []
  if (rule.domain?.length) conditions.push(`Domain: ${rule.domain.join(', ')}`)
  if (rule.domain_suffix?.length) conditions.push(`Suffix: ${rule.domain_suffix.join(', ')}`)
  if (rule.domain_keyword?.length) conditions.push(`Keyword: ${rule.domain_keyword.join(', ')}`)
  if (rule.geosite?.length) conditions.push(`GeoSite: ${rule.geosite.join(', ')}`)
  return conditions.length > 0 ? conditions.join(' | ') : 'No conditions'
}
</script>

<template>
  <div>
    <div class="flex justify-end mb-4">
      <Button @click="openAddRuleModal" variant="primary">
        <PlusIcon class="h-5 w-5 mr-2" />
        Add DNS Rule
      </Button>
    </div>

    <!-- DNS Rules Table -->
    <div class="bg-white dark:bg-slate-800 rounded-lg shadow overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">DNS Rules</h3>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-1">
          Rules are processed in order. First match wins.
        </p>
      </div>

      <div v-if="loading && rules.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <div v-else-if="rules.length === 0" class="text-center py-12">
        <p class="text-gray-500 dark:text-gray-500 mb-4">No DNS rules configured</p>
        <Button @click="openAddRuleModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          Add Your First DNS Rule
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-slate-900">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">#</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">Action</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">Server</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">Conditions</th>
              <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-for="(rule, index) in rules" :key="index" class="hover:bg-gray-50 dark:hover:bg-slate-700">
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
                  <Button @click="openDeleteConfirm(index)" variant="ghost" size="sm" class="text-red-600 hover:text-red-700">
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
                    <select
                      v-model="currentRule.action"
                      class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-slate-700 text-gray-900 dark:text-gray-100"
                    >
                      <option v-for="action in actionTypes" :key="action.value" :value="action.value">
                        {{ action.label }}
                      </option>
                    </select>
                  </div>

                  <!-- Server (for route action) -->
                  <div v-if="currentRule.action === 'route'">
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">DNS Server *</label>
                    <select
                      v-model="currentRule.server"
                      class="block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 bg-white dark:bg-slate-700 text-gray-900 dark:text-gray-100"
                    >
                      <option value="">Select a server</option>
                      <option v-for="server in servers" :key="server.tag" :value="server.tag">
                        {{ server.tag }}
                      </option>
                    </select>
                  </div>

                  <!-- Rule Conditions -->
                  <div class="border-t border-gray-200 dark:border-gray-700 pt-4">
                    <h4 class="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">Conditions (at least one required)</h4>

                    <div class="space-y-3">
                      <!-- Domain -->
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain</label>
                        <Input
                          v-model="currentRule.domain"
                          placeholder="example.com, google.com (comma separated)"
                        />
                        <p class="mt-1 text-xs text-gray-500">Exact domain match</p>
                      </div>

                      <!-- Domain Suffix -->
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain Suffix</label>
                        <Input
                          v-model="currentRule.domain_suffix"
                          placeholder=".example.com, .google.com (comma separated)"
                        />
                        <p class="mt-1 text-xs text-gray-500">Matches domain and all subdomains</p>
                      </div>

                      <!-- Domain Keyword -->
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain Keyword</label>
                        <Input
                          v-model="currentRule.domain_keyword"
                          placeholder="google, facebook (comma separated)"
                        />
                        <p class="mt-1 text-xs text-gray-500">Domain contains keyword</p>
                      </div>

                      <!-- GeoSite -->
                      <div>
                        <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">GeoSite</label>
                        <Input
                          v-model="currentRule.geosite"
                          placeholder="google, netflix, cn (comma separated)"
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
