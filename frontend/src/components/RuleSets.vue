<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import Card from './Card.vue'
import Input from './Input.vue'
import { Dialog, Button, Select } from '../volt'
import type { RuleSet } from '../types/api'
import { useToast } from 'primevue'
import { useRouteStore } from '../stores/route'
import { storeToRefs } from 'pinia'

const toast = useToast()
const routeStore = useRouteStore()
const { ruleSets, loading } = storeToRefs(routeStore)

// State for dialog
const showAddRuleSetDialog = ref(false)
const editingRuleSet = ref<{ tag: string; ruleSet: RuleSet } | null>(null)

// Form data
const ruleSetForm = ref<RuleSet>({ tag: '', type: 'remote', format: 'source' })

const typeOptions = [
  { label: 'Remote', value: 'remote' },
  { label: 'Local', value: 'local' },
]

// formatOptions removed - not currently used
// const formatOptions = [
//   { label: 'Source', value: 'source' },
//   { label: 'Binary', value: 'binary' },
// ]

// Computed properties for v-model
const currentRuleSetTag = computed({
  get: () => editingRuleSet.value ? editingRuleSet.value.ruleSet.tag : ruleSetForm.value.tag,
  set: (val) => {
    if (editingRuleSet.value) {
      editingRuleSet.value.ruleSet.tag = val
    } else {
      ruleSetForm.value.tag = val
    }
  }
})

const currentRuleSetType = computed({
  get: () => editingRuleSet.value ? editingRuleSet.value.ruleSet.type : ruleSetForm.value.type,
  set: (val) => {
    if (editingRuleSet.value) {
      editingRuleSet.value.ruleSet.type = val
    } else {
      ruleSetForm.value.type = val
    }
  }
})

// currentRuleSetFormat removed - not currently used
// const currentRuleSetFormat = computed({
//   get: () => editingRuleSet.value ? editingRuleSet.value.ruleSet.format : ruleSetForm.value.format,
//   set: (val) => {
//     if (editingRuleSet.value) {
//       editingRuleSet.value.ruleSet.format = val
//     } else {
//       ruleSetForm.value.format = val
//     }
//   }
// })

const currentRuleSetUrl = computed({
  get: () => editingRuleSet.value ? editingRuleSet.value.ruleSet.url : ruleSetForm.value.url,
  set: (val) => {
    if (editingRuleSet.value) {
      editingRuleSet.value.ruleSet.url = val
    } else {
      ruleSetForm.value.url = val
    }
  }
})

const currentRuleSetUpdateInterval = computed({
  get: () => editingRuleSet.value ? editingRuleSet.value.ruleSet.update_interval : ruleSetForm.value.update_interval,
  set: (val) => {
    if (editingRuleSet.value) {
      editingRuleSet.value.ruleSet.update_interval = val
    } else {
      ruleSetForm.value.update_interval = val
    }
  }
})

// Dialog visibility
const dialogVisible = computed({
  get: () => showAddRuleSetDialog.value || !!editingRuleSet.value,
  set: (val) => {
    if (!val) {
      showAddRuleSetDialog.value = false
      editingRuleSet.value = null
      ruleSetForm.value = { tag: '', type: 'remote', format: 'source' }
    }
  }
})

// Functions
function startEditRuleSet(ruleSet: RuleSet) {
  editingRuleSet.value = { tag: ruleSet.tag, ruleSet: { ...ruleSet } }
}

// Remote rule sets are fetched server-side by sing-box. Restrict to http(s)
// so the URL field can't be coerced into a file:// / gopher:// scheme.
function validateRuleSetUrl(form: RuleSet): string | null {
  if (form.type !== 'remote') return null
  if (!form.url || !form.url.trim()) return 'URL is required for remote rule sets'
  try {
    const parsed = new URL(form.url)
    if (!['http:', 'https:'].includes(parsed.protocol)) {
      return 'URL must use http or https'
    }
  } catch {
    return 'Invalid URL format'
  }
  return null
}

async function handleAddRuleSet() {
  const urlError = validateRuleSetUrl(ruleSetForm.value)
  if (urlError) {
    toast.add({ severity: 'error', summary: 'Validation Error', detail: urlError, life: 3000 })
    return
  }
  try {
    await routeStore.addRuleSet(ruleSetForm.value)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Rule set added successfully',
      life: 3000
    })
    dialogVisible.value = false
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to add rule set',
      life: 3000
    })
  }
}

async function handleUpdateRuleSet() {
  if (editingRuleSet.value) {
    const urlError = validateRuleSetUrl(editingRuleSet.value.ruleSet)
    if (urlError) {
      toast.add({ severity: 'error', summary: 'Validation Error', detail: urlError, life: 3000 })
      return
    }
    try {
      await routeStore.updateRuleSet(editingRuleSet.value.tag, editingRuleSet.value.ruleSet)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'Rule set updated successfully',
        life: 3000
      })
      dialogVisible.value = false
    } catch (err: any) {
      toast.add({
        severity: 'error',
        summary: 'Error',
        detail: err.message || 'Failed to update rule set',
        life: 3000
      })
    }
  }
}

async function handleDeleteRuleSet(tag: string) {
  if (!confirm('Are you sure you want to delete this rule set?')) return

  try {
    await routeStore.deleteRuleSet(tag)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Rule set deleted successfully',
      life: 3000
    })
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to delete rule set',
      life: 3000
    })
  }
}

// Load data on mount
onMounted(() => {
  routeStore.fetchRuleSets()
})
</script>

<template>
  <div class="space-y-6">
    <Card>
      <div class="flex justify-between items-center mb-4">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
          Rule Sets
        </h3>
        <button
          @click="showAddRuleSetDialog = true"
          class="px-4 py-2 bg-violet-600 text-white rounded-md hover:bg-violet-700 transition-colors"
        >
          Add Rule Set
        </button>
      </div>

      <div v-if="loading" class="text-center py-8">
        <div class="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
      </div>

      <div v-else-if="ruleSets.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
        No rule sets configured
      </div>

      <div v-else class="space-y-4">
        <div
          v-for="ruleSet in ruleSets"
          :key="ruleSet.tag"
          class="border border-gray-200 dark:border-gray-700 rounded-lg p-4"
        >
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <div class="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span class="font-medium text-gray-700 dark:text-gray-300">Tag:</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.tag }}</span>
                </div>
                <div>
                  <span class="font-medium text-gray-700 dark:text-gray-300">Type:</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.type }}</span>
                </div>
                <div>
                  <span class="font-medium text-gray-700 dark:text-gray-300">Format:</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.format }}</span>
                </div>
                <div v-if="ruleSet.url">
                  <span class="font-medium text-gray-700 dark:text-gray-300">URL:</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.url }}</span>
                </div>
                <div v-if="ruleSet.update_interval">
                  <span class="font-medium text-gray-700 dark:text-gray-300">Update Interval:</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.update_interval }}</span>
                </div>
              </div>
            </div>
            <div class="flex space-x-2 ml-4">
              <button
                @click="startEditRuleSet(ruleSet)"
                class="text-violet-600 hover:text-violet-800 dark:text-violet-400 dark:hover:text-violet-300"
              >
                Edit
              </button>
              <button
                @click="handleDeleteRuleSet(ruleSet.tag)"
                class="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
              >
                Delete
              </button>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- Add/Edit Rule Set Dialog -->
    <Dialog
      v-model:visible="dialogVisible"
      modal
      :header="editingRuleSet ? 'Edit Rule Set' : 'Add Rule Set'"
      class="w-full max-w-lg"
    >
      <div class="space-y-4">
        <Input
          v-model="currentRuleSetTag"
          label="Tag *"
          placeholder="Rule set tag"
          :disabled="!!editingRuleSet"
        />

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Type *
          </label>
          <Select
            v-model="currentRuleSetType"
            :options="typeOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Select type"
            class="w-full"
          />
        </div>

        <!-- <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Format *
          </label>
          <Select
            v-model="currentRuleSetFormat"
            :options="formatOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Select format"
            class="w-full"
          />
        </div> -->

        <Input
          v-if="currentRuleSetType === 'remote'"
          v-model="currentRuleSetUrl"
          label="URL *"
          placeholder="Rule set URL"
        />

        <Input
          v-model="currentRuleSetUpdateInterval"
          label="Update Interval"
          placeholder="e.g., 1d, 12h, 30m"
        />
      </div>

      <template #footer>
        <Button
          label="Cancel"
          severity="secondary"
          @click="dialogVisible = false"
        />
        <Button
          :label="editingRuleSet ? 'Update' : 'Add'"
          @click="editingRuleSet ? handleUpdateRuleSet() : handleAddRuleSet()"
        />
      </template>
    </Dialog>
  </div>
</template>
