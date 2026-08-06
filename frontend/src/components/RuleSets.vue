<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import Card from './Card.vue'
import Input from './Input.vue'
import SmartRoutingRuleWizard from './SmartRoutingRuleWizard.vue'
import { Dialog, Select } from '../volt'
import Button from './Button.vue'
import PopConfirm from './PopConfirm.vue'
import type { RuleSet } from '../types/api'
import { routeService } from '../services'
import { useToast } from 'primevue'
import { useRouteStore } from '../stores/route'
import { storeToRefs } from 'pinia'
import { useConfirm } from '../composables/useConfirm'

const toast = useToast()
const { t } = useI18n()
const { confirm } = useConfirm()
const routeStore = useRouteStore()
const { ruleSets, loading } = storeToRefs(routeStore)

// State for dialog
const showAddRuleSetDialog = ref(false)
const editingRuleSet = ref<{ tag: string; ruleSet: RuleSet } | null>(null)

// Smart Routing Rule wizard, opened (seeded with the new rule_set tag) when the
// user opts in after adding a rule set — routing + clean DNS in one flow.
const showWizard = ref(false)
const wizardSeedTag = ref<string>('')

// Form data
const ruleSetForm = ref<RuleSet>({ tag: '', type: 'remote', format: 'source' })

const typeOptions = computed(() => [
  { label: t('route.ruleSets.types.remote'), value: 'remote' },
  { label: t('route.ruleSets.types.local'), value: 'local' },
])

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
  if (!form.url || !form.url.trim()) return t('route.ruleSets.validation.urlRequired')
  try {
    const parsed = new URL(form.url)
    if (!['http:', 'https:'].includes(parsed.protocol)) {
      return t('route.ruleSets.validation.urlScheme')
    }
  } catch {
    return t('route.ruleSets.validation.urlInvalid')
  }
  return null
}

async function handleAddRuleSet() {
  const urlError = validateRuleSetUrl(ruleSetForm.value)
  if (urlError) {
    toast.add({ severity: 'error', summary: t('route.ruleSets.toast.validationError'), detail: urlError, life: 3000 })
    return
  }
  // Capture the tag before dialogVisible reset clears the form.
  const addedTag = ruleSetForm.value.tag
  try {
    await routeStore.addRuleSet(ruleSetForm.value)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('route.ruleSets.toast.added'),
      life: 3000
    })
    dialogVisible.value = false

    // Offer to create a routing rule for the new rule set, which flows into the
    // Smart Routing Rule wizard (and its DNS-pollution guard).
    if (addedTag) {
      const wantRoute = await confirm({
        title: t('route.ruleSets.configurePrompt.title'),
        message: t('route.ruleSets.configurePrompt.message', { tag: addedTag }),
        confirmLabel: t('route.ruleSets.configurePrompt.confirm'),
        cancelLabel: t('route.ruleSets.configurePrompt.cancel'),
      })
      if (wantRoute) {
        wizardSeedTag.value = addedTag
        showWizard.value = true
      }
    }
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.ruleSets.toast.addFailed'),
      life: 3000
    })
  }
}

async function handleUpdateRuleSet() {
  if (editingRuleSet.value) {
    const urlError = validateRuleSetUrl(editingRuleSet.value.ruleSet)
    if (urlError) {
      toast.add({ severity: 'error', summary: t('route.ruleSets.toast.validationError'), detail: urlError, life: 3000 })
      return
    }
    try {
      await routeStore.updateRuleSet(editingRuleSet.value.tag, editingRuleSet.value.ruleSet)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('route.ruleSets.toast.updated'),
        life: 3000
      })
      dialogVisible.value = false
    } catch (err: any) {
      toast.add({
        severity: 'error',
        summary: t('common.error'),
        detail: err.message || t('route.ruleSets.toast.updateFailed'),
        life: 3000
      })
    }
  }
}

/**
 * Per-tag reference lookup backing the delete popovers.
 *
 * Deleting a rule set can cascade into route/DNS rules that reference it, and
 * the user needs to see that before confirming. The lookup used to run *before*
 * opening a modal; with an anchored popover the panel opens immediately, so it
 * now runs on open and the confirm button stays disabled until it settles.
 *
 * Keyed by tag because every row renders its own <PopConfirm>.
 */
interface ReferenceInfo {
  loading: boolean
  /** True when rules reference this set, so the delete must clean them up. */
  cascade: boolean
  /** Human summary of what the cascade will change. Empty when not cascading. */
  details: string
}

const referenceInfo = ref<Record<string, ReferenceInfo>>({})

const IDLE_REFERENCE: ReferenceInfo = { loading: false, cascade: false, details: '' }

// Replaces the map rather than mutating it, per the project's immutability rule.
function setReferenceInfo(tag: string, info: ReferenceInfo) {
  referenceInfo.value = { ...referenceInfo.value, [tag]: info }
}

async function loadRuleSetReferences(tag: string) {
  setReferenceInfo(tag, { ...IDLE_REFERENCE, loading: true })
  try {
    const { data } = await routeService.getRuleSetReferences(tag)
    const refs = data.references || []
    if (!refs.length) {
      setReferenceInfo(tag, IDLE_REFERENCE)
      return
    }
    setReferenceInfo(tag, {
      loading: false,
      cascade: true,
      // `cascade.details` deliberately omits the tag — PopConfirm already shows
      // it as its own chip, so repeating it here would just be noise.
      details: t('route.ruleSets.cascade.details', {
        routeCount: data.route_count,
        dnsCount: data.dns_count,
        stripCount: refs.filter((r) => r.action === 'strip').length,
        deleteCount: refs.filter((r) => r.action === 'delete').length,
      }),
    })
  } catch {
    // Reference lookup failed — fall back to a plain, non-cascading delete
    // rather than blocking the user on a diagnostic call.
    setReferenceInfo(tag, IDLE_REFERENCE)
  }
}

// Confirmation already happened in the row's <PopConfirm>.
async function handleDeleteRuleSet(tag: string) {
  const cascade = referenceInfo.value[tag]?.cascade ?? false

  try {
    await routeStore.deleteRuleSet(tag, { cascade })
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('route.ruleSets.toast.deleted'),
      life: 3000
    })
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('route.ruleSets.toast.deleteFailed'),
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
          {{ $t('route.ruleSets.title') }}
        </h3>
        <button
          @click="showAddRuleSetDialog = true"
          class="px-4 py-2 bg-primary-600 text-white rounded-control hover:bg-primary-700 transition-colors"
        >
          {{ $t('route.ruleSets.add') }}
        </button>
      </div>

      <div v-if="loading" class="text-center py-8">
        <div class="inline-block animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <div v-else-if="ruleSets.length === 0" class="text-center py-8 text-gray-500 dark:text-gray-400">
        {{ $t('route.ruleSets.empty') }}
      </div>

      <div v-else class="space-y-4">
        <div
          v-for="ruleSet in ruleSets"
          :key="ruleSet.tag"
          class="border border-gray-200 dark:border-gray-700 rounded-surface p-4"
        >
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <div class="grid grid-cols-2 gap-4 text-sm">
                <div>
                  <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleSets.fields.tag') }}</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.tag }}</span>
                </div>
                <div>
                  <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleSets.fields.type') }}</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.type }}</span>
                </div>
                <div>
                  <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleSets.fields.format') }}</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.format }}</span>
                </div>
                <div v-if="ruleSet.url">
                  <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleSets.fields.url') }}</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.url }}</span>
                </div>
                <div v-if="ruleSet.update_interval">
                  <span class="font-medium text-gray-700 dark:text-gray-300">{{ $t('route.ruleSets.fields.updateInterval') }}</span>
                  <span class="ml-2 text-gray-900 dark:text-gray-100">{{ ruleSet.update_interval }}</span>
                </div>
              </div>
            </div>
            <div class="flex space-x-2 ml-4">
              <button
                @click="startEditRuleSet(ruleSet)"
                class="text-primary-600 hover:text-primary-800 dark:text-primary-400 dark:hover:text-primary-300"
              >
                {{ $t('common.edit') }}
              </button>
              <PopConfirm
                :message="$t('route.ruleSets.confirm.delete')"
                :target="ruleSet.tag"
                :details="referenceInfo[ruleSet.tag]?.details"
                :loading="referenceInfo[ruleSet.tag]?.loading ?? false"
                :loading-label="$t('route.ruleSets.cascade.checking')"
                :confirm-label="
                  referenceInfo[ruleSet.tag]?.cascade
                    ? $t('route.ruleSets.cascade.confirm')
                    : $t('common.delete')
                "
                tone="danger"
                align="right"
                trigger-class="text-red-600 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300 focus:outline-none focus-visible:ring-2 focus-visible:ring-danger rounded-control"
                @open="loadRuleSetReferences(ruleSet.tag)"
                @confirm="handleDeleteRuleSet(ruleSet.tag)"
              >
                {{ $t('common.delete') }}
              </PopConfirm>
            </div>
          </div>
        </div>
      </div>
    </Card>

    <!-- Smart Routing Rule wizard, seeded with the just-added rule set -->
    <SmartRoutingRuleWizard
      v-model:visible="showWizard"
      seed-match-type="rule_set"
      :seed-values="[wizardSeedTag]"
    />

    <!-- Add/Edit Rule Set Dialog -->
    <Dialog
      v-model:visible="dialogVisible"
      modal
      :header="editingRuleSet ? $t('route.ruleSets.modal.edit') : $t('route.ruleSets.modal.add')"
      class="w-full max-w-lg"
    >
      <div class="space-y-4">
        <Input
          v-model="currentRuleSetTag"
          :label="$t('route.ruleSets.form.tag')"
          :placeholder="$t('route.ruleSets.form.tagPlaceholder')"
          :disabled="!!editingRuleSet"
        />

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {{ $t('route.ruleSets.form.type') }}
          </label>
          <Select
            v-model="currentRuleSetType"
            :options="typeOptions"
            optionLabel="label"
            optionValue="value"
            :placeholder="$t('route.ruleSets.form.typePlaceholder')"
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
          :label="$t('route.ruleSets.form.url')"
          :placeholder="$t('route.ruleSets.form.urlPlaceholder')"
        />

        <Input
          v-model="currentRuleSetUpdateInterval"
          :label="$t('route.ruleSets.form.updateInterval')"
          :placeholder="$t('route.ruleSets.form.updateIntervalPlaceholder')"
        />
      </div>

      <template #footer>
        <Button
          :label="$t('common.cancel')"
          severity="secondary"
          @click="dialogVisible = false"
        />
        <Button
          :label="editingRuleSet ? $t('common.update') : $t('common.add')"
          @click="editingRuleSet ? handleUpdateRuleSet() : handleAddRuleSet()"
        />
      </template>
    </Dialog>
  </div>
</template>
