<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Inbound } from '../../types/api'
import Button from '../../components/Button.vue'
import Input from '../../components/Input.vue'
import Badge from '../../components/Badge.vue'
import Modal from '../../components/Modal.vue'
import Table from '../../components/Table.vue'
import SchemaFieldsEditor from '../../components/SchemaFieldsEditor.vue'
import { PlusIcon, PencilIcon, TrashIcon, DocumentDuplicateIcon, CheckIcon } from '@heroicons/vue/24/outline'
import { inboundService } from '../../services'
import { useToast } from 'primevue/usetoast'
import {
  prepareInboundForEdit,
  prepareInboundForType,
  validateInboundRequiredFields,
} from '../../utils/inboundRequiredFields'
import {
  INBOUND_TYPE_NAMES,
  USER_FIELDS,
  resolveInboundFields,
  type InboundTypeName,
} from '../../schemas/inboundFields'
import { Select } from '../../volt'

const inbounds = ref<Inbound[]>([])
const loading = ref(false)
const copiedTag = ref<string | null>(null)
const toast = useToast()
const { t, te } = useI18n()

const DEFAULT_TYPE: InboundTypeName = 'mixed'

function blankInbound(): Record<string, unknown> {
  return prepareInboundForType({ tag: '' }, DEFAULT_TYPE)
}

// Modal state
const showModal = ref(false)
const isEditMode = ref(false)
const editingTag = ref<string>('')
const currentInbound = ref<Record<string, unknown>>(blankInbound())

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingInbound = ref<Inbound | null>(null)

/**
 * Driven by the generated inventory rather than a hand-written list, which is
 * how "anytls" came to be registered on the backend since 1.12 while never
 * appearing in this dropdown. Labels stay opt-in: a type without an
 * `inbounds.types.*` entry shows its own name rather than a raw i18n path.
 */
const inboundTypes = computed(() =>
  INBOUND_TYPE_NAMES.map((value) => ({
    value,
    label: te(`inbounds.types.${value}`) ? t(`inbounds.types.${value}`) : value,
  })),
)

const getInboundTypeLabel = (type: string) => {
  return inboundTypes.value.find(it => it.value === type)?.label || type
}

/**
 * Switching type prunes the previous type's fields before seeding the new
 * one's defaults. Carrying them over used to produce a payload with, say,
 * shadowsocks' `method` on a trojan inbound — which sing-box rejects, because
 * it decodes inbound options strictly.
 */
function changeType(next: unknown) {
  if (typeof next !== 'string') return
  currentInbound.value = prepareInboundForType(currentInbound.value, next as InboundTypeName)
}

/**
 * The form model is an open record because an inbound's fields depend on its
 * type, and the declared `Inbound` union cannot narrow on `type` (the
 * discriminant sits outside the union). This is the one place that shape is
 * read back as a type name, so the assertion lives here rather than at every
 * template use.
 */
const currentType = computed(() =>
  typeof currentInbound.value.type === 'string'
    ? (currentInbound.value.type as InboundTypeName)
    : undefined,
)

const getInboundBadgeVariant = (type: string): 'primary' | 'success' | 'warning' | 'info' | 'secondary' => {
  if (type === 'mixed' || type === 'http' || type === 'socks') return 'primary'
  if (type === 'tun') return 'success'
  return 'info'
}

const fetchInbounds = async () => {
  loading.value = true
  try {
    const { data } = await inboundService.getInbounds()
    inbounds.value = data.inbounds || []
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('inbounds.toast.fetchFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const openAddModal = () => {
  isEditMode.value = false
  currentInbound.value = blankInbound()
  showModal.value = true
}

const openEditModal = (inbound: Inbound) => {
  isEditMode.value = true
  editingTag.value = inbound.tag
  // No defaults seeded on edit: writing one into a config that deliberately
  // omitted the key would change behaviour before the operator touched
  // anything, and show up in the diff as their change.
  currentInbound.value = prepareInboundForEdit(inbound as unknown as Record<string, unknown>)
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  currentInbound.value = blankInbound()
}

const handleSave = async () => {
  const validationError = validateInboundRequiredFields(currentInbound.value)
  if (validationError) {
    toast.add({
      severity: 'error',
      summary: t('inbounds.validation.title'),
      detail: t(validationError.key),
      life: 3000
    })
    return
  }

  loading.value = true
  try {
    if (isEditMode.value) {
      await inboundService.updateInbound(editingTag.value, currentInbound.value as unknown as Inbound)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('inbounds.toast.updatedOk'),
        life: 3000
      })
    } else {
      await inboundService.addInbound(currentInbound.value as unknown as Inbound)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('inbounds.toast.addedOk'),
        life: 3000
      })
    }
    closeModal()
    await fetchInbounds()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('inbounds.toast.saveFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const openDeleteConfirm = (inbound: Inbound) => {
  deletingInbound.value = inbound
  showDeleteConfirm.value = true
}

const closeDeleteConfirm = () => {
  showDeleteConfirm.value = false
  deletingInbound.value = null
}

const handleDelete = async () => {
  if (!deletingInbound.value) return

  loading.value = true

  try {
    await inboundService.deleteInbound(deletingInbound.value.tag)
    toast.add({
      severity: 'success',
      summary: t('common.success'),
      detail: t('inbounds.toast.deletedOk'),
      life: 3000
    })
    closeDeleteConfirm()
    await fetchInbounds()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('inbounds.toast.deleteFailed'),
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

// Credential generation moved to utils/credentials.ts and is reached through
// the users editor, so it now works for every type that takes credentials
// rather than only the shadowsocks and vmess blocks that used to be hardcoded
// here. Type changes are handled by changeType(), which prunes before seeding.

// Watch for modal close to reset form state
watch(showModal, (newValue) => {
  if (!newValue) {
    // Reset form when modal is closed
    setTimeout(() => {
      currentInbound.value = blankInbound()
    }, 300) // Wait for transition to complete
  }
})

watch(showDeleteConfirm, (newValue) => {
  if (!newValue) {
    setTimeout(() => {
      deletingInbound.value = null
    }, 300)
  }
})

const copyClientConfig = async (inbound: Inbound) => {
  const anyInbound = inbound as any
  const server = !anyInbound.listen || anyInbound.listen === '0.0.0.0' || anyInbound.listen === '::' || anyInbound.listen === '127.0.0.1'
    ? window.location.hostname || 'YOUR_SERVER_IP'
    : anyInbound.listen

  const outbound: any = {
    type: anyInbound.type === 'mixed' ? 'socks' : anyInbound.type,
    tag: `${anyInbound.tag}-out`,
    server: server,
    server_port: anyInbound.listen_port,
  }

  // Copy standard user credentials if available
  const firstUser = (inbound as any).users?.[0]
  if (firstUser) {
    if (firstUser.uuid) outbound.uuid = firstUser.uuid
    if (firstUser.password) outbound.password = firstUser.password
    if (firstUser.username) outbound.username = firstUser.username
    else if (firstUser.name) outbound.username = firstUser.name
  }

  // Copy other top level credentials/attributes
  if ((inbound as any).method) outbound.method = (inbound as any).method
  if ((inbound as any).password) outbound.password = (inbound as any).password
  if ((inbound as any).uuid) outbound.uuid = (inbound as any).uuid
  if ((inbound as any).security) outbound.security = (inbound as any).security

  // Copy TLS settings if present, pruning server-side keys
  if ((inbound as any).tls) {
    const tls = { ...((inbound as any).tls) }
    tls.enabled = true
    tls.server_name = server
    delete tls.certificate
    delete tls.certificate_path
    delete tls.key
    delete tls.key_path
    outbound.tls = tls
  }

  // Copy transport settings if present
  if ((inbound as any).transport) {
    outbound.transport = (inbound as any).transport
  }

  const jsonText = JSON.stringify(outbound, null, 2)

  try {
    await navigator.clipboard.writeText(jsonText)
    copiedTag.value = inbound.tag
    setTimeout(() => {
      if (copiedTag.value === inbound.tag) {
        copiedTag.value = null
      }
    }, 2000)
  } catch (err: any) {
    console.error('Failed to copy client config:', err)
  }
}

onMounted(fetchInbounds)
</script>

<template>
  <div class="page-shell">
    <div class="flex justify-between items-center mb-4">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">{{ $t('inbounds.title') }}</h2>
      <Button @click="openAddModal" variant="primary">
        <PlusIcon class="h-5 w-5 mr-2" />
        {{ $t('inbounds.add') }}
      </Button>
    </div>

    <div class="bg-white dark:bg-slate-800 rounded-surface shadow-surface overflow-hidden">
      <Table :loading="loading && inbounds.length === 0" :empty="inbounds.length === 0">
        <template #empty>
          <p class="text-gray-500 dark:text-gray-500 mb-3">{{ $t('inbounds.empty') }}</p>
          <Button @click="openAddModal" variant="primary" size="sm">
            <PlusIcon class="h-4 w-4 mr-1.5" />
            {{ $t('inbounds.addFirst') }}
          </Button>
        </template>

        <template #head>
          <th>{{ $t('inbounds.table.tag') }}</th>
          <th>{{ $t('inbounds.table.type') }}</th>
          <th>{{ $t('inbounds.table.listenAddress') }}</th>
          <th>{{ $t('inbounds.table.port') }}</th>
          <th>{{ $t('inbounds.table.sniff') }}</th>
          <th class="col-actions">{{ $t('inbounds.table.actions') }}</th>
        </template>

        <tr v-for="inbound in inbounds" :key="inbound.tag">
          <td class="font-medium text-gray-900 dark:text-gray-100">{{ inbound.tag }}</td>
          <td>
            <Badge :variant="getInboundBadgeVariant(inbound.type)">
              {{ getInboundTypeLabel(inbound.type) }}
            </Badge>
          </td>
          <td class="text-gray-900 dark:text-gray-100">{{ (inbound as any).listen || '-' }}</td>
          <td class="text-gray-900 dark:text-gray-100">{{ (inbound as any).listen_port || '-' }}</td>
          <td>
            <Badge v-if="(inbound as any).sniff" variant="success">{{ $t('inbounds.sniff.enabled') }}</Badge>
            <Badge v-else variant="secondary">{{ $t('inbounds.sniff.disabled') }}</Badge>
          </td>
          <td class="col-actions font-medium">
            <div class="flex items-center justify-end gap-1">
              <Button @click="copyClientConfig(inbound)" variant="ghost" size="sm" action :title="$t('inbounds.tooltip.copyConfig')">
                <CheckIcon v-if="copiedTag === inbound.tag" class="h-4.5 w-4.5 text-emerald-500 dark:text-emerald-400" />
                <DocumentDuplicateIcon v-else class="h-4.5 w-4.5 text-primary-600 dark:text-primary-400" />
              </Button>
              <Button @click="openEditModal(inbound)" variant="ghost" size="sm" action>
                <PencilIcon class="h-4 w-4" />
              </Button>
              <Button @click="openDeleteConfirm(inbound)" variant="ghost" size="sm" action class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300">
                <TrashIcon class="h-4 w-4" />
              </Button>
            </div>
          </td>
        </tr>
      </Table>
    </div>

    <!-- Add/Edit Modal -->
    <Modal
      :model-value="showModal"
      @update:model-value="(v) => { if (!v) closeModal() }"
      :title="isEditMode ? $t('inbounds.modal.edit') : $t('inbounds.modal.add')"
      size="md"
      show-close
    >
      <div class="space-y-3">
        <!--
          Tag and type are not part of any type's option struct — they live on
          the inbound wrapper — so they are rendered here rather than coming
          from the schema. Both are locked while editing: the tag is the
          identity other config sections reference, and changing the type would
          reinterpret every remaining field.
        -->
        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.tag') }}</label>
          <Input
            v-model="(currentInbound as any).tag"
            :placeholder="$t('inbounds.form.tagPlaceholder')"
            :disabled="isEditMode"
          />
          <p class="mt-1 text-xs text-gray-500">{{ $t('inbounds.form.tagHelp') }}</p>
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.type') }}</label>
          <Select
            class="w-full"
            :modelValue="currentInbound.type"
            :options="inboundTypes"
            optionLabel="label"
            optionValue="value"
            :placeholder="$t('inbounds.form.typePlaceholder')"
            :disabled="isEditMode"
            @update:modelValue="changeType"
          />
        </div>

        <!--
          :key remounts the editor on every type change, which resets the set of
          fields the operator had added. Without it, switching shadowsocks →
          trojan would leave shadowsocks' fields on screen bound to keys the new
          type does not have. Same mechanism as RoutingRules' matchersKey.
        -->
        <SchemaFieldsEditor
          v-if="currentType"
          :key="currentType"
          v-model="currentInbound"
          :fields="resolveInboundFields(currentType)"
          :user-fields="USER_FIELDS[currentType]"
        />
      </div>

      <template #footer>
        <Button @click="closeModal" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleSave" variant="primary" :disabled="loading">
          {{ isEditMode ? $t('common.update') : $t('common.add') }}
        </Button>
      </template>
    </Modal>

    <!-- Delete Confirmation Modal -->
    <Modal
      :model-value="showDeleteConfirm"
      @update:model-value="(v) => { if (!v) closeDeleteConfirm() }"
      :title="$t('inbounds.del.title')"
      size="sm"
      show-close
    >
      <p class="text-gray-700 dark:text-gray-300">
        {{ $t('inbounds.del.confirm', { tag: deletingInbound?.tag }) }}
      </p>

      <template #footer>
        <Button @click="closeDeleteConfirm" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleDelete" variant="danger" :disabled="loading">
          {{ $t('common.delete') }}
        </Button>
      </template>
    </Modal>
  </div>
</template>
