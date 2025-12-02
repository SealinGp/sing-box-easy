<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import {
  TransitionRoot,
  TransitionChild,
  Dialog,
  DialogPanel,
  DialogTitle,
} from '@headlessui/vue'
import { apiService } from '../../services/api'
import type { Inbound } from '../../types/api'
import Alert from '../../components/Alert.vue'
import Button from '../../components/Button.vue'
import Input from '../../components/Input.vue'
import Select from '../../components/Select.vue'
import Badge from '../../components/Badge.vue'
import { PlusIcon, PencilIcon, TrashIcon, XMarkIcon } from '@heroicons/vue/24/outline'

const inbounds = ref<Inbound[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const successMessage = ref<string | null>(null)

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

const inboundTypes = [
  { value: 'mixed', label: 'Mixed (HTTP/SOCKS)' },
  { value: 'http', label: 'HTTP' },
  { value: 'socks', label: 'SOCKS' },
  { value: 'tun', label: 'TUN' },
  { value: 'redirect', label: 'Redirect' },
  { value: 'tproxy', label: 'TProxy' },
  { value: 'direct', label: 'Direct' },
  { value: 'shadowsocks', label: 'Shadowsocks' },
  { value: 'vmess', label: 'VMess' },
  { value: 'trojan', label: 'Trojan' },
  { value: 'vless', label: 'VLESS' },
  { value: 'hysteria', label: 'Hysteria' },
  { value: 'hysteria2', label: 'Hysteria2' },
  { value: 'tuic', label: 'TUIC' },
  { value: 'naive', label: 'Naive' },
  { value: 'shadowtls', label: 'ShadowTLS' },
]

const getInboundTypeLabel = (type: string) => {
  return inboundTypes.find(t => t.value === type)?.label || type
}

const getInboundBadgeVariant = (type: string): 'primary' | 'success' | 'warning' | 'info' | 'secondary' => {
  if (type === 'mixed' || type === 'http' || type === 'socks') return 'primary'
  if (type === 'tun') return 'success'
  return 'info'
}

const fetchInbounds = async () => {
  loading.value = true
  error.value = null
  try {
    const response = await apiService.getInbounds()
    inbounds.value = response.inbounds || []
  } catch (err: any) {
    console.error('Failed to fetch inbounds:', err)
    error.value = err.response?.data?.error || 'Failed to fetch inbounds'
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
  error.value = null
  successMessage.value = null

  // Validation
  if (!currentInbound.value.tag?.trim()) {
    error.value = 'Tag is required'
    return
  }

  if (!currentInbound.value.type) {
    error.value = 'Type is required'
    return
  }

  // For most inbound types (except TUN), require listen_port
  if (currentInbound.value.type !== 'tun' && !(currentInbound.value as any).listen_port) {
    error.value = 'Listen port is required'
    return
  }

  loading.value = true
  try {
    if (isEditMode.value) {
      await apiService.updateInbound(editingTag.value, currentInbound.value as Inbound)
      successMessage.value = 'Inbound updated successfully'
    } else {
      await apiService.addInbound(currentInbound.value as Inbound)
      successMessage.value = 'Inbound added successfully'
    }
    closeModal()
    await fetchInbounds()
  } catch (err: any) {
    console.error('Failed to save inbound:', err)
    error.value = err.response?.data?.error || 'Failed to save inbound'
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

  error.value = null
  successMessage.value = null
  loading.value = true

  try {
    await apiService.deleteInbound(deletingInbound.value.tag)
    successMessage.value = 'Inbound deleted successfully'
    closeDeleteConfirm()
    await fetchInbounds()
  } catch (err: any) {
    console.error('Failed to delete inbound:', err)
    error.value = err.response?.data?.error || 'Failed to delete inbound'
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
      <h2 class="text-3xl font-bold text-gray-900">Inbounds Management</h2>
      <Button @click="openAddModal" variant="primary">
        <PlusIcon class="h-5 w-5 mr-2" />
        Add Inbound
      </Button>
    </div>

    <Alert v-if="error" type="error" closable @close="error = null" class="mb-6">
      {{ error }}
    </Alert>
    <Alert v-if="successMessage" type="success" closable @close="successMessage = null" class="mb-6">
      {{ successMessage }}
    </Alert>

    <div class="bg-white rounded-lg shadow overflow-hidden">
      <div v-if="loading && inbounds.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <div v-else-if="inbounds.length === 0" class="text-center py-12">
        <p class="text-gray-500 mb-4">No inbounds configured</p>
        <Button @click="openAddModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          Add Your First Inbound
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Tag</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Listen Address</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Port</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Sniff</th>
              <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-for="inbound in inbounds" :key="inbound.tag" class="hover:bg-gray-50">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-gray-900">{{ inbound.tag }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge :variant="getInboundBadgeVariant(inbound.type)">
                  {{ getInboundTypeLabel(inbound.type) }}
                </Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900">{{ (inbound as any).listen || '-' }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900">{{ (inbound as any).listen_port || '-' }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge v-if="(inbound as any).sniff" variant="success">Enabled</Badge>
                <Badge v-else variant="secondary">Disabled</Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="flex items-center justify-end gap-2">
                  <Button @click="openEditModal(inbound)" variant="ghost" size="sm">
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button @click="openDeleteConfirm(inbound)" variant="ghost" size="sm" class="text-red-600 hover:text-red-700">
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
          <div class="fixed inset-0 bg-black/25" />
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
              <DialogPanel class="w-full max-w-lg transform overflow-hidden rounded-lg bg-white p-6 text-left align-middle shadow-xl transition-all">
                <div class="flex items-center justify-between mb-4">
                  <DialogTitle as="h3" class="text-lg font-semibold text-gray-900">
                    {{ isEditMode ? 'Edit Inbound' : 'Add Inbound' }}
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
                    <label class="block text-sm font-medium text-gray-700 mb-1">Tag *</label>
                    <Input
                      v-model="(currentInbound as any).tag"
                      placeholder="e.g., mixed-in"
                      :disabled="isEditMode"
                    />
                    <p class="mt-1 text-xs text-gray-500">Unique identifier for this inbound</p>
                  </div>

                  <div>
                    <label class="block text-sm font-medium text-gray-700 mb-1">Type *</label>
                    <Select v-model="(currentInbound as any).type" :options="inboundTypes" :disabled="isEditMode" />
                  </div>

                  <div v-if="currentInbound.type !== 'tun'">
                    <label class="block text-sm font-medium text-gray-700 mb-1">Listen Address</label>
                    <Input
                      v-model="(currentInbound as any).listen"
                      placeholder="127.0.0.1 or 0.0.0.0"
                    />
                    <p class="mt-1 text-xs text-gray-500">IP address to listen on</p>
                  </div>

                  <div v-if="currentInbound.type !== 'tun'">
                    <label class="block text-sm font-medium text-gray-700 mb-1">Listen Port *</label>
                    <Input
                      v-model.number="(currentInbound as any).listen_port"
                      type="number"
                      placeholder="1080"
                    />
                  </div>

                  <div class="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="sniff"
                      v-model="(currentInbound as any).sniff"
                      class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                    <label for="sniff" class="text-sm font-medium text-gray-700">Enable Traffic Sniffing</label>
                  </div>

                  <div v-if="(currentInbound as any).sniff" class="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="sniff_override"
                      v-model="(currentInbound as any).sniff_override_destination"
                      class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                    <label for="sniff_override" class="text-sm font-medium text-gray-700">Override Destination</label>
                  </div>

                  <div v-if="currentInbound.type === 'tun'">
                    <label class="block text-sm font-medium text-gray-700 mb-1">Interface Name</label>
                    <Input
                      v-model="(currentInbound as any).interface_name"
                      placeholder="tun0"
                    />
                  </div>

                  <div v-if="currentInbound.type === 'tun'">
                    <label class="block text-sm font-medium text-gray-700 mb-1">MTU</label>
                    <Input
                      v-model.number="(currentInbound as any).mtu"
                      type="number"
                      placeholder="1500"
                    />
                  </div>

                  <div v-if="currentInbound.type === 'tun'" class="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="auto_route"
                      v-model="(currentInbound as any).auto_route"
                      class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                    <label for="auto_route" class="text-sm font-medium text-gray-700">Auto Route</label>
                  </div>
                </div>

                <div class="mt-6 flex justify-end gap-3">
                  <Button @click="closeModal" variant="secondary">Cancel</Button>
                  <Button @click="handleSave" variant="primary" :disabled="loading">
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
          <div class="fixed inset-0 bg-black/25" />
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
              <DialogPanel class="w-full max-w-md transform overflow-hidden rounded-lg bg-white p-6 text-left align-middle shadow-xl transition-all">
                <div class="flex items-center justify-between mb-4">
                  <DialogTitle as="h3" class="text-lg font-semibold text-gray-900">
                    Delete Inbound
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeDeleteConfirm"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

                <p class="text-gray-700">
                  Are you sure you want to delete the inbound
                  <strong>{{ deletingInbound?.tag }}</strong>?
                  This action cannot be undone.
                </p>

                <div class="mt-6 flex justify-end gap-3">
                  <Button @click="closeDeleteConfirm" variant="secondary">Cancel</Button>
                  <Button @click="handleDelete" variant="danger" :disabled="loading">
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
