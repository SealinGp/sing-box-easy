<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  TransitionRoot,
  TransitionChild,
  Dialog,
  DialogPanel,
  DialogTitle,
} from '@headlessui/vue'
import type { Inbound } from '../../types/api'
import Button from '../../components/Button.vue'
import Input from '../../components/Input.vue'
import Badge from '../../components/Badge.vue'
import { PlusIcon, PencilIcon, TrashIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import { inboundService } from '../../services'
import { useToast } from 'primevue/usetoast'

const inbounds = ref<Inbound[]>([])
const loading = ref(false)
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
  // Validation
  if (!currentInbound.value.tag?.trim()) {
    toast.add({
      severity: 'error',
      summary: t('inbounds.validation.title'),
      detail: t('inbounds.validation.tagRequired'),
      life: 3000
    })
    return
  }

  if (!currentInbound.value.type) {
    toast.add({
      severity: 'error',
      summary: t('inbounds.validation.title'),
      detail: t('inbounds.validation.typeRequired'),
      life: 3000
    })
    return
  }

  // For most inbound types (except TUN), require listen_port
  if (currentInbound.value.type !== 'tun' && !(currentInbound.value as any).listen_port) {
    toast.add({
      severity: 'error',
      summary: t('inbounds.validation.title'),
      detail: t('inbounds.validation.listenPortRequired'),
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

    <div class="bg-white dark:bg-slate-800 rounded-lg shadow dark:shadow-xl dark:shadow-slate-700/50 overflow-hidden">
      <div v-if="loading && inbounds.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
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
                <div class="flex items-center justify-end gap-2">
                  <Button @click="openEditModal(inbound)" variant="ghost" size="sm">
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button @click="openDeleteConfirm(inbound)" variant="ghost" size="sm" class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300">
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
    <TransitionRoot appear :show="showModal" as="template">
      <Dialog as="div" @close="closeModal" class="relative z-50">
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
              <DialogPanel class="w-full max-w-lg transform overflow-hidden rounded-lg bg-white dark:bg-slate-800 p-6 text-left align-middle shadow-xl transition-all">
                <div class="flex items-center justify-between mb-4">
                  <DialogTitle as="h3" class="text-lg font-semibold text-gray-900 dark:text-gray-100">
                    {{ isEditMode ? $t('inbounds.modal.edit') : $t('inbounds.modal.add') }}
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeModal"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

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
                    <select class="select" v-model="(currentInbound as any).type" :disabled="isEditMode">
                        <option disabled selected>{{ $t('inbounds.form.typePlaceholder') }}</option>
                        <option v-for="inboundType in inboundTypes" :value="inboundType.value" >{{ inboundType.label }}</option>
                      </select>
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

                  <div class="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="sniff"
                      v-model="(currentInbound as any).sniff"
                      class="rounded border-gray-300 text-violet-600 focus:ring-violet-500"
                    />
                    <label for="sniff" class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ $t('inbounds.form.enableSniff') }}</label>
                  </div>

                  <div v-if="(currentInbound as any).sniff" class="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="sniff_override"
                      v-model="(currentInbound as any).sniff_override_destination"
                      class="rounded border-gray-300 text-violet-600 focus:ring-violet-500"
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
                      class="rounded border-gray-300 text-violet-600 focus:ring-violet-500"
                    />
                    <label for="auto_route" class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ $t('inbounds.form.autoRoute') }}</label>
                  </div>
                </div>

                <div class="mt-6 flex justify-end gap-3">
                  <Button @click="closeModal" variant="secondary">{{ $t('common.cancel') }}</Button>
                  <Button @click="handleSave" variant="primary" :disabled="loading">
                    {{ isEditMode ? $t('common.update') : $t('common.add') }}
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
                    {{ $t('inbounds.del.title') }}
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
                  {{ $t('inbounds.del.confirm', { tag: deletingInbound?.tag }) }}
                </p>

                <div class="mt-6 flex justify-end gap-3">
                  <Button @click="closeDeleteConfirm" variant="secondary">{{ $t('common.cancel') }}</Button>
                  <Button @click="handleDelete" variant="danger" :disabled="loading">
                    {{ $t('common.delete') }}
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
