<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { Inbound } from '../../types/api'
import Button from '../../components/Button.vue'
import Input from '../../components/Input.vue'
import Badge from '../../components/Badge.vue'
import Modal from '../../components/Modal.vue'
import { PlusIcon, PencilIcon, TrashIcon, DocumentDuplicateIcon, CheckIcon } from '@heroicons/vue/24/outline'
import { inboundService } from '../../services'
import { useToast } from 'primevue/usetoast'
import { applyInboundTypeDefaults, generateVmessUUID, validateInboundRequiredFields } from '../../utils/inboundRequiredFields'
import { Select } from '../../volt'

const inbounds = ref<Inbound[]>([])
const loading = ref(false)
const copiedTag = ref<string | null>(null)
const toast = useToast()
const { t } = useI18n()

// Modal state
const showModal = ref(false)
const isEditMode = ref(false)
const editingTag = ref<string>('')
const currentInbound = ref<Partial<Inbound>>({
  type: 'mixed',
  tag: '',
  listen: '127.0.0.1',
  listen_port: 1080,
})

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingInbound = ref<Inbound | null>(null)

const inboundTypes = computed(() => [
  { value: 'mixed', label: t('inbounds.types.mixed') },
  { value: 'http', label: t('inbounds.types.http') },
  { value: 'socks', label: t('inbounds.types.socks') },
  { value: 'tun', label: t('inbounds.types.tun') },
  { value: 'redirect', label: t('inbounds.types.redirect') },
  { value: 'tproxy', label: t('inbounds.types.tproxy') },
  { value: 'direct', label: t('inbounds.types.direct') },
  { value: 'shadowsocks', label: t('inbounds.types.shadowsocks') },
  { value: 'vmess', label: t('inbounds.types.vmess') },
  { value: 'trojan', label: t('inbounds.types.trojan') },
  { value: 'vless', label: t('inbounds.types.vless') },
  { value: 'hysteria', label: t('inbounds.types.hysteria') },
  { value: 'hysteria2', label: t('inbounds.types.hysteria2') },
  { value: 'tuic', label: t('inbounds.types.tuic') },
  { value: 'naive', label: t('inbounds.types.naive') },
  { value: 'shadowtls', label: t('inbounds.types.shadowtls') },
])

const shadowsocksMethods = [
  '2022-blake3-aes-128-gcm',
  '2022-blake3-aes-256-gcm',
  '2022-blake3-chacha20-poly1305',
  'none',
  'aes-128-gcm',
  'aes-192-gcm',
  'aes-256-gcm',
  'chacha20-ietf-poly1305',
  'xchacha20-ietf-poly1305',
]

const getInboundTypeLabel = (type: string) => {
  return inboundTypes.value.find(it => it.value === type)?.label || type
}

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
  currentInbound.value = {
    type: 'mixed',
    tag: '',
    listen: '127.0.0.1',
    listen_port: 1080,
  }
  showModal.value = true
}

const openEditModal = (inbound: Inbound) => {
  isEditMode.value = true
  editingTag.value = inbound.tag
  currentInbound.value = { ...inbound }
  applyInboundTypeDefaults(currentInbound.value as any)
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  currentInbound.value = {
    type: 'mixed',
    tag: '',
    listen: '127.0.0.1',
    listen_port: 1080,
  }
}

const handleSave = async () => {
  const validationError = validateInboundRequiredFields(currentInbound.value as any)
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
      await inboundService.updateInbound(editingTag.value, currentInbound.value as Inbound)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('inbounds.toast.updatedOk'),
        life: 3000
      })
    } else {
      await inboundService.addInbound(currentInbound.value as Inbound)
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

const generatePassword = () => {
  const method = (currentInbound.value as any).method || '2022-blake3-aes-128-gcm'
  if (method === 'none') {
    ;(currentInbound.value as any).password = ''
    return
  }
  
  if (method.startsWith('2022-blake3-')) {
    const keyLen = method.includes('128') ? 16 : 32
    const arr = new Uint8Array(keyLen)
    window.crypto.getRandomValues(arr)
    let binary = ''
    const len = arr.byteLength
    for (let i = 0; i < len; i++) {
      binary += String.fromCharCode(arr[i] as number)
    }
    ;(currentInbound.value as any).password = window.btoa(binary)
  } else {
    // Generate a secure random string (32 characters)
    const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
    let result = ''
    const arr = new Uint8Array(32)
    window.crypto.getRandomValues(arr)
    for (let i = 0; i < arr.length; i++) {
      result += chars.charAt((arr[i] as number) % chars.length)
    }
    ;(currentInbound.value as any).password = result
  }
}

const generateVMessUUID = () => {
  const inbound = currentInbound.value as any
  if (!Array.isArray(inbound.users) || inbound.users.length === 0) {
    inbound.users = [{ name: 'sekai', uuid: '', alterId: 0 }]
  }
  inbound.users[0].uuid = generateVmessUUID()
}

watch(() => currentInbound.value.type, (newType) => {
  applyInboundTypeDefaults(currentInbound.value as any)
  if (newType === 'shadowsocks' && !(currentInbound.value as any).password) {
    generatePassword()
  }
})

// Watch for modal close to reset form state
watch(showModal, (newValue) => {
  if (!newValue) {
    // Reset form when modal is closed
    setTimeout(() => {
      currentInbound.value = {
        type: 'mixed',
        tag: '',
        listen: '127.0.0.1',
        listen_port: 1080,
      }
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
  <div class="p-8">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">{{ $t('inbounds.title') }}</h2>
      <Button @click="openAddModal" variant="primary">
        <PlusIcon class="h-5 w-5 mr-2" />
        {{ $t('inbounds.add') }}
      </Button>
    </div>

    <div class="bg-white dark:bg-slate-800 rounded-surface shadow dark:shadow-float dark:shadow-slate-700/50 overflow-hidden">
      <div v-if="loading && inbounds.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <div v-else-if="inbounds.length === 0" class="text-center py-12">
        <p class="text-gray-500 dark:text-gray-500 mb-4">{{ $t('inbounds.empty') }}</p>
        <Button @click="openAddModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          {{ $t('inbounds.addFirst') }}
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('inbounds.table.tag') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('inbounds.table.type') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('inbounds.table.listenAddress') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('inbounds.table.port') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('inbounds.table.sniff') }}</th>
              <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('inbounds.table.actions') }}</th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-for="inbound in inbounds" :key="inbound.tag" class="hover:bg-gray-50 dark:hover:bg-gray-700">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ inbound.tag }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge :variant="getInboundBadgeVariant(inbound.type)">
                  {{ getInboundTypeLabel(inbound.type) }}
                </Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-gray-100">{{ (inbound as any).listen || '-' }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-gray-100">{{ (inbound as any).listen_port || '-' }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge v-if="(inbound as any).sniff" variant="success">{{ $t('inbounds.sniff.enabled') }}</Badge>
                <Badge v-else variant="secondary">{{ $t('inbounds.sniff.disabled') }}</Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="inbound-table-actions flex items-center justify-end gap-2">
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
          </tbody>
        </table>
      </div>
    </div>

    <!-- Add/Edit Modal -->
    <Modal
      :model-value="showModal"
      @update:model-value="(v) => { if (!v) closeModal() }"
      :title="isEditMode ? $t('inbounds.modal.edit') : $t('inbounds.modal.add')"
      size="md"
      show-close
    >
      <div class="space-y-4">
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
            v-model="(currentInbound as any).type"
            :options="inboundTypes"
            optionLabel="label"
            optionValue="value"
            :placeholder="$t('inbounds.form.typePlaceholder')"
            :disabled="isEditMode"
          />
        </div>

        <div v-if="currentInbound.type !== 'tun'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.listenAddress') }}</label>
          <Input
            v-model="(currentInbound as any).listen"
            :placeholder="$t('inbounds.form.listenAddressPlaceholder')"
          />
          <p class="mt-1 text-xs text-gray-500">{{ $t('inbounds.form.listenAddressHelp') }}</p>
        </div>

        <div v-if="currentInbound.type !== 'tun'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.listenPort') }}</label>
          <Input
            v-model.number="(currentInbound as any).listen_port"
            type="number"
            :placeholder="$t('inbounds.form.listenPortPlaceholder')"
          />
        </div>

        <!-- Shadowsocks Options -->
        <div v-if="currentInbound.type === 'shadowsocks'" class="space-y-4 border-t border-gray-100 dark:border-gray-700 pt-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.ssMethod') }}</label>
            <Select
              class="w-full"
              v-model="(currentInbound as any).method"
              :options="shadowsocksMethods"
              @change="generatePassword"
            />
          </div>

          <div v-if="(currentInbound as any).method !== 'none'">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.ssPassword') }}</label>
            <div class="flex gap-2">
              <div class="flex-1">
                <Input
                  v-model="(currentInbound as any).password"
                  placeholder="Password or Key"
                />
              </div>
              <Button type="button" @click="generatePassword" variant="secondary" class="shrink-0 flex items-center justify-center">
                {{ $t('inbounds.form.generate') }}
              </Button>
            </div>
            <p class="mt-1 text-xs text-gray-500">
              {{ (currentInbound as any).method?.startsWith('2022-blake3-')
                ? $t('inbounds.form.ssPasswordHelp2022', { len: (currentInbound as any).method.includes('128') ? 16 : 32 })
                : $t('inbounds.form.ssPasswordHelpOther')
              }}
            </p>
          </div>
        </div>

        <!-- VMess Options -->
        <div v-if="currentInbound.type === 'vmess'" class="space-y-4 border-t border-gray-100 dark:border-gray-700 pt-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.vmessUserName') }}</label>
            <Input
              v-model="(currentInbound as any).users[0].name"
              :placeholder="$t('inbounds.form.vmessUserNamePlaceholder')"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.vmessUUID') }}</label>
            <div class="flex gap-2">
              <div class="flex-1">
                <Input
                  v-model="(currentInbound as any).users[0].uuid"
                  :placeholder="$t('inbounds.form.vmessUUIDPlaceholder')"
                />
              </div>
              <Button type="button" @click="generateVMessUUID" variant="secondary" class="shrink-0 flex items-center justify-center">
                {{ $t('inbounds.form.generate') }}
              </Button>
            </div>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.vmessAlterId') }}</label>
            <Input
              v-model.number="(currentInbound as any).users[0].alterId"
              type="number"
              :placeholder="$t('inbounds.form.vmessAlterIdPlaceholder')"
            />
            <p class="mt-1 text-xs text-gray-500">{{ $t('inbounds.form.vmessAlterIdHelp') }}</p>
          </div>
        </div>

        <div class="flex items-center gap-2">
          <input
            type="checkbox"
            id="sniff"
            v-model="(currentInbound as any).sniff"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <label for="sniff" class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ $t('inbounds.form.enableSniff') }}</label>
        </div>

        <div v-if="(currentInbound as any).sniff" class="flex items-center gap-2">
          <input
            type="checkbox"
            id="sniff_override"
            v-model="(currentInbound as any).sniff_override_destination"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <label for="sniff_override" class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ $t('inbounds.form.overrideDestination') }}</label>
        </div>

        <div v-if="currentInbound.type === 'tun'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.interfaceName') }}</label>
          <Input
            v-model="(currentInbound as any).interface_name"
            :placeholder="$t('inbounds.form.interfaceNamePlaceholder')"
          />
        </div>

        <div v-if="currentInbound.type === 'tun'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('inbounds.form.mtu') }}</label>
          <Input
            v-model.number="(currentInbound as any).mtu"
            type="number"
            :placeholder="$t('inbounds.form.mtuPlaceholder')"
          />
        </div>

        <div v-if="currentInbound.type === 'tun'" class="flex items-center gap-2">
          <input
            type="checkbox"
            id="auto_route"
            v-model="(currentInbound as any).auto_route"
            class="rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <label for="auto_route" class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ $t('inbounds.form.autoRoute') }}</label>
        </div>
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
