<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import {
  TransitionRoot,
  TransitionChild,
  Dialog,
  DialogPanel,
  DialogTitle,
} from '@headlessui/vue'
import type { Outbound } from '../../types/api'
import type { OutboundType } from '../../types/outbound'
import Button from '../../components/Button.vue'
import Input from '../../components/Input.vue'
import Badge from '../../components/Badge.vue'
import Textarea from '../../components/Textarea.vue'
import { PlusIcon, PencilIcon, TrashIcon, XMarkIcon, ArrowDownTrayIcon } from '@heroicons/vue/24/outline'
import { nodesService, outboundService } from '../../services'

const outbounds = ref<Outbound[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const successMessage = ref<string | null>(null)

// Modal state
const showModal = ref(false)
const isEditMode = ref(false)
const editingTag = ref<string>('')
const currentOutbound = ref<any>({
  type: 'direct',
  tag: '',
})

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingOutbound = ref<Outbound | null>(null)
const deletingOutboundIdx = ref<number>(-1)

// Import modal state
const showImportModal = ref(false)
const importInput = ref('')
const parsing = ref(false)
const parsedNodes = ref<Outbound[]>([])
const selectedNodes = ref<Set<string>>(new Set())
const parseError = ref('')
const importing = ref(false)

const outboundTypes = [
  { value: 'direct', label: 'Direct' },
  { value: 'block', label: 'Block' },
  { value: 'socks', label: 'SOCKS' },
  { value: 'http', label: 'HTTP' },
  { value: 'shadowsocks', label: 'Shadowsocks' },
  { value: 'vmess', label: 'VMess' },
  { value: 'trojan', label: 'Trojan' },
  { value: 'vless', label: 'VLESS' },
  { value: 'hysteria', label: 'Hysteria' },
  { value: 'hysteria2', label: 'Hysteria2' },
  { value: 'tuic', label: 'TUIC' },
  { value: 'wireguard', label: 'WireGuard' },
  { value: 'ssh', label: 'SSH' },
  { value: 'tor', label: 'Tor' },
  { value: 'shadowtls', label: 'ShadowTLS' },
  { value: 'selector', label: 'Selector (Group)' },
  { value: 'urltest', label: 'URLTest (Group)' },
]



const getOutboundTypeLabel = (type: string) => {
  return outboundTypes.find(t => t.value === type)?.label || type
}

const getOutboundBadgeVariant = (type: string): 'primary' | 'success' | 'warning' | 'info' | 'secondary' | 'danger' => {
  if (type === 'direct') return 'success'
  if (type === 'block') return 'danger'
  if (type === 'selector' || type === 'urltest') return 'warning'
  return 'primary'
}

const isProxyType = (type: OutboundType) => {
  const nonProxyTypes = ['direct', 'block', 'dns', 'selector', 'urltest']
  return nonProxyTypes.indexOf(type) === -1
}

const isGroupType = (type: OutboundType) => {
  const groupTypes = ['selector', 'urltest']
  return groupTypes.indexOf(type) !== -1
}

const needsServer = computed(() => isProxyType(currentOutbound.value.type))
const needsPassword = computed(() => {
  const types = ['shadowsocks', 'trojan', 'hysteria2', 'tuic']
  return types.indexOf(currentOutbound.value.type) !== -1
})
const needsUUID = computed(() => {
  const types = ['vmess', 'vless', 'tuic']
  return types.indexOf(currentOutbound.value.type) !== -1
})
const needsMethod = computed(() => currentOutbound.value.type === 'shadowsocks')
const needsOutbounds = computed(() => isGroupType(currentOutbound.value.type))

const fetchOutbounds = async () => {
  loading.value = true
  error.value = null
  try {
    const {data} = await outboundService.getOutbounds() 
    outbounds.value = data.outbounds
  } catch (err: any) {
    console.error('Failed to fetch outbounds:', err)
    error.value = err.response?.data?.error || 'Failed to fetch outbounds'
  } finally {
    loading.value = false
  }
}

const openAddModal = () => {
  isEditMode.value = false
  currentOutbound.value = {
    type: 'direct',
    tag: '',
  }
  showModal.value = true
}

const openEditModal = (outbound: Outbound) => {
  isEditMode.value = true
  editingTag.value = outbound.tag
  currentOutbound.value = { ...outbound }
  showModal.value = true
}

const closeModal = () => {
  showModal.value = false
  currentOutbound.value = {
    type: 'direct',
    tag: '',
  }
}

const handleSave = async () => {
  error.value = null
  successMessage.value = null

  // Validation
  if (!currentOutbound.value.tag?.trim()) {
    error.value = 'Tag is required'
    return
  }

  if (!currentOutbound.value.type) {
    error.value = 'Type is required'
    return
  }

  // Validate proxy-specific fields
  if (needsServer.value) {
    if (!currentOutbound.value.server?.trim()) {
      error.value = 'Server address is required'
      return
    }
    if (!currentOutbound.value.server_port) {
      error.value = 'Server port is required'
      return
    }
  }

  if (needsPassword.value && !currentOutbound.value.password?.trim()) {
    error.value = 'Password is required'
    return
  }

  if (needsUUID.value && !currentOutbound.value.uuid?.trim()) {
    error.value = 'UUID is required'
    return
  }

  if (needsMethod.value && !currentOutbound.value.method?.trim()) {
    error.value = 'Encryption method is required'
    return
  }

  if (needsOutbounds.value && (!currentOutbound.value.outbounds || currentOutbound.value.outbounds.length === 0)) {
    error.value = 'At least one outbound is required for groups'
    return
  }

  loading.value = true
  try {
    if (isEditMode.value) {
      await outboundService.updateOutbound(editingTag.value, currentOutbound.value)
      successMessage.value = 'Outbound updated successfully'
    } else {
      await outboundService.addOutbound(currentOutbound.value)
      successMessage.value = 'Outbound added successfully'
    }
    closeModal()
    await fetchOutbounds()
  } catch (err: any) {
    console.error('Failed to save outbound:', err)
    error.value = err.response?.data?.error || 'Failed to save outbound'
  } finally {
    loading.value = false
  }
}

const openDeleteConfirm = (outbound: Outbound, i:number) => {
  deletingOutbound.value = outbound
  deletingOutboundIdx.value = i
  showDeleteConfirm.value = true
}

const closeDeleteConfirm = () => {
  showDeleteConfirm.value = false
  deletingOutbound.value = null
}

const handleDelete = async () => {
  if (!deletingOutbound.value) return

  error.value = null
  successMessage.value = null
  loading.value = true

  try {
    const val = deletingOutboundIdx.value > -1 ? `${deletingOutboundIdx.value}` : deletingOutbound.value.tag
    await outboundService.deleteOutbound(val)
    successMessage.value = 'Outbound deleted successfully'
    closeDeleteConfirm()
    await fetchOutbounds()
  } catch (err: any) {
    console.error('Failed to delete outbound:', err)
    error.value = err.response?.data?.error || 'Failed to delete outbound'
  } finally {
    loading.value = false
  }
}

// Import functions
const openImportModal = () => {
  importInput.value = ''
  parsedNodes.value = []
  selectedNodes.value.clear()
  parseError.value = ''
  showImportModal.value = true
}

const closeImportModal = () => {
  showImportModal.value = false
  importInput.value = ''
  parsedNodes.value = []
  selectedNodes.value.clear()
  parseError.value = ''
}

const parseSubscription = async () => {
  if (!importInput.value.trim()) {
    parseError.value = 'Please enter subscription URL(s) or node link(s)'
    return
  }

  parsing.value = true
  parseError.value = ''
  parsedNodes.value = []
  selectedNodes.value.clear()

  try {
    const lines = importInput.value
      .split('\n')
      .map(line => line.trim())
      .filter(line => line.length > 0)

    const linesToParse = lines.join('\n')
    const {data} = await nodesService.parseNodes(linesToParse)
    parsedNodes.value = data.nodes

    // Select all by default
    data.nodes.forEach((node) => {
      selectedNodes.value.add(node.tag)
    })

    if (data.nodes.length === 0) {
      parseError.value = 'No nodes found'
    }
  } catch (err: any) {
    parseError.value = err.response?.data?.error || err.message || 'Failed to parse subscription/nodes'
  } finally {
    parsing.value = false
  }
}

const toggleNode = (tag: string) => {
  if (selectedNodes.value.has(tag)) {
    selectedNodes.value.delete(tag)
  } else {
    selectedNodes.value.add(tag)
  }
}

const toggleSelectAll = () => {
  if (selectedNodes.value.size === parsedNodes.value.length) {
    selectedNodes.value.clear()
  } else {
    parsedNodes.value.forEach((node) => {
      selectedNodes.value.add(node.tag)
    })
  }
}

const handleImport = async () => {
  importing.value = true
  error.value = ''

  try {
    const outboundsToAdd: Outbound[] = []

    parsedNodes.value.forEach((node) => {
      if (selectedNodes.value.has(node.tag)) {
        outboundsToAdd.push(node)
      }
    })

    if (outboundsToAdd.length > 0) {
      const {data} = await outboundService.addOutboundsBatch(outboundsToAdd)
      successMessage.value = `Successfully imported ${data.added} outbounds (${data.skipped} skipped)`
      closeImportModal()
      await fetchOutbounds()
    }
  } catch (err: any) {
    error.value = err.response?.data?.error || err.message || 'Failed to import outbounds'
  } finally {
    importing.value = false
  }
}

// Watch for modal close to reset form state
watch(showModal, (newValue) => {
  if (!newValue) {
    setTimeout(() => {
      currentOutbound.value = {
        type: 'direct',
        tag: '',
      }
    }, 300)
  }
})

watch(showDeleteConfirm, (newValue) => {
  if (!newValue) {
    setTimeout(() => {
      deletingOutbound.value = null
    }, 300)
  }
})

// Reset fields when type changes
watch(() => currentOutbound.value.type, () => {
  const tag = currentOutbound.value.tag
  const type = currentOutbound.value.type
  currentOutbound.value = { type, tag }
})

onMounted(fetchOutbounds)
</script>

<template>
  <div class="p-8">
    <div class="flex justify-between items-center mb-6">
      <h2 class="text-3xl font-bold text-gray-900">Outbounds Management</h2>
      <div class="flex gap-3">
        <Button @click="openImportModal" variant="secondary">
          <ArrowDownTrayIcon class="h-5 w-5 mr-2" />
          Import
        </Button>
        <Button @click="openAddModal" variant="primary">
          <PlusIcon class="h-5 w-5 mr-2" />
          Add Outbound
        </Button>
      </div>
    </div>


    <Alert v-if="error" type="error" closable @close="error = null" class="mb-6">
      {{ error }}
    </Alert>
    <Alert v-if="successMessage" type="success" closable @close="successMessage = null" class="mb-6">
      {{ successMessage }}
    </Alert>

    <div class="bg-white rounded-lg shadow overflow-hidden">
      <div v-if="loading && outbounds.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <div v-else-if="outbounds.length === 0" class="text-center py-12">
        <p class="text-gray-500 mb-4">No outbounds configured</p>
        <Button @click="openAddModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          Add Your First Outbound
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="table">
          <!-- head -->
          <thead>
            <tr>
              <th></th>
              <th>Tag</th>
              <th>Type</th>
              <th>Server</th>
              <th>Port</th>
              <th class="text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            <!-- 动态行 -->
            <tr v-for="(outbound,i) in outbounds" :key="outbound.tag" class="hover">
              <th>{{ i + 1 }}</th>
              <td>{{ outbound.tag || i }}</td>
              <td>
                <Badge :variant="getOutboundBadgeVariant(outbound.type)">
                  {{ getOutboundTypeLabel(outbound.type) }}
                </Badge>
              </td>
              <td>{{ (outbound as any).server || '-' }}</td>
              <td>{{ (outbound as any).server_port || '-' }}</td>
              <td class="text-right">
                <div class="flex items-center justify-end gap-2">
                  <Button @click="openEditModal(outbound)" variant="ghost" size="sm">
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button @click="openDeleteConfirm(outbound, i)" variant="ghost" size="sm" class="text-red-600 hover:text-red-700">
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
              <DialogPanel class="w-full max-w-2xl transform overflow-hidden rounded-lg bg-white p-6 text-left align-middle shadow-xl transition-all">
                <div class="flex items-center justify-between mb-4">
                  <DialogTitle as="h3" class="text-lg font-semibold text-gray-900">
                    {{ isEditMode ? 'Edit Outbound' : 'Add Outbound' }}
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeModal"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

                <div class="space-y-4 pr-2">
                  <div class="grid grid-cols-2 gap-4">
                    <div>
                      <label class="block text-sm font-medium text-gray-700 mb-1">Tag *</label>
                      <Input
                        v-model="currentOutbound.tag"
                        placeholder="e.g., proxy-us"
                        :disabled="isEditMode"
                      />
                      <p class="mt-1 text-xs text-gray-500">Unique identifier</p>
                    </div>

                    <div>
                      <label class="block text-sm font-medium text-gray-700 mb-1">Type *</label>
                      <select class="select" v-model="currentOutbound.type" :disabled="isEditMode">
                        <option disabled selected>Pick outbound type</option>
                        <option v-for="outboundType in outboundTypes" :value="outboundType.value" >{{ outboundType.label }}</option>
                      </select>
                    </div>
                  </div>

                  <!-- Proxy Server Fields -->
                  <div v-if="needsServer" class="grid grid-cols-2 gap-4">
                    <div class="col-span-1">
                      <label class="block text-sm font-medium text-gray-700 mb-1">Server *</label>
                      <Input
                        v-model="currentOutbound.server"
                        placeholder="example.com or 1.2.3.4"
                      />
                    </div>

                    <div class="col-span-1">
                      <label class="block text-sm font-medium text-gray-700 mb-1">Port *</label>
                      <Input
                        v-model.number="currentOutbound.server_port"
                        type="number"
                        placeholder="443"
                      />
                    </div>
                  </div>

                  <!-- Method (Shadowsocks) -->
                  <div v-if="needsMethod">
                    <label class="block text-sm font-medium text-gray-700 mb-1">Encryption Method *</label>
                    <Input
                      v-model="currentOutbound.method"
                      placeholder="e.g., aes-256-gcm, chacha20-ietf-poly1305"
                    />
                  </div>

                  <!-- Password -->
                  <div v-if="needsPassword">
                    <label class="block text-sm font-medium text-gray-700 mb-1">Password *</label>
                    <Input
                      v-model="currentOutbound.password"
                      type="password"
                      placeholder="Enter password"
                    />
                  </div>

                  <!-- UUID (VMess, VLESS, TUIC) -->
                  <div v-if="needsUUID">
                    <label class="block text-sm font-medium text-gray-700 mb-1">UUID *</label>
                    <Input
                      v-model="currentOutbound.uuid"
                      placeholder="e.g., 12345678-1234-1234-1234-123456789012"
                    />
                  </div>

                  <!-- Group Outbounds (Selector, URLTest) -->
                  <div v-if="needsOutbounds">
                    <label class="block text-sm font-medium text-gray-700 mb-1">Outbounds *</label>
                    <Input
                      v-model="currentOutbound.outbounds"
                      placeholder="Comma-separated tags: proxy-us, proxy-uk"
                    />
                    <p class="mt-1 text-xs text-gray-500">List of outbound tags to include in this group</p>
                  </div>

                  <!-- URLTest specific -->
                  <div v-if="currentOutbound.type === 'urltest'" class="grid grid-cols-2 gap-4">
                    <div>
                      <label class="block text-sm font-medium text-gray-700 mb-1">Test URL</label>
                      <Input
                        v-model="currentOutbound.url"
                        placeholder="https://www.gstatic.com/generate_204"
                      />
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-700 mb-1">Interval</label>
                      <Input
                        v-model="currentOutbound.interval"
                        placeholder="3m"
                      />
                    </div>
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

    <!-- Import Modal -->
    <TransitionRoot appear :show="showImportModal" as="template">
      <Dialog as="div" @close="closeImportModal" class="relative z-50">
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
              <DialogPanel class="w-full max-w-2xl transform overflow-hidden rounded-lg bg-white p-6 text-left align-middle shadow-xl transition-all">
                <div class="flex items-center justify-between mb-4">
                  <DialogTitle as="h3" class="text-lg font-semibold text-gray-900">
                    Import Outbounds
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeImportModal"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

                <div class="space-y-4">
                  <!-- Input Section -->
                  <div v-if="parsedNodes.length === 0">
                    <p class="text-sm text-gray-600 mb-3">
                      Enter subscription URL(s) or direct node links (vmess://, ss://, trojan://, etc.). One per line for multiple entries.
                    </p>
                    <Textarea
                      v-model="importInput"
                      placeholder="Examples:&#10;https://example.com/subscribe?token=xxx&#10;vmess://eyJhZGQiOiIxMC4xMC4xMC4xMCIsImFpZCI6IjAiLCJob3N0IjoiIiwiaWQiOiI...&#10;ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ=@192.168.1.1:8388#MyNode&#10;trojan://password@example.com:443?sni=example.com#TrojanNode"
                      :disabled="parsing"
                      :error="parseError"
                      :rows="6"
                      full-width
                    />
                    <div class="flex justify-end mt-3">
                      <Button
                        variant="primary"
                        :loading="parsing"
                        :disabled="parsing"
                        @click="parseSubscription"
                      >
                        {{ parsing ? 'Parsing...' : 'Parse' }}
                      </Button>
                    </div>
                  </div>

                  <!-- Parsed Nodes List -->
                  <div v-else>
                    <div class="flex items-center justify-between mb-3">
                      <div>
                        <h4 class="text-sm font-semibold text-gray-900">
                          Parsed Nodes
                          <Badge variant="primary" class="ml-2" size="sm">{{ parsedNodes.length }}</Badge>
                        </h4>
                        <p class="text-xs text-gray-600 mt-1">
                          {{ selectedNodes.size }} selected
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        @click="toggleSelectAll"
                      >
                        {{ selectedNodes.size === parsedNodes.length ? 'Deselect All' : 'Select All' }}
                      </Button>
                    </div>

                    <!-- Nodes List -->
                    <div class="max-h-80 overflow-y-auto border border-gray-200 rounded-lg">
                      <div
                        v-for="node in parsedNodes"
                        :key="node.tag"
                        class="flex items-center gap-3 p-3 hover:bg-gray-50 border-b border-gray-100 last:border-0 cursor-pointer"
                        @click="toggleNode(node.tag)"
                      >
                        <input
                          type="checkbox"
                          :checked="selectedNodes.has(node.tag)"
                          class="w-4 h-4 text-blue-600 border-gray-300 rounded focus:ring-blue-500"
                          @click.stop="toggleNode(node.tag)"
                        />
                        <div class="flex-1 min-w-0">
                          <div class="flex items-center gap-2">
                            <p class="text-sm font-medium text-gray-900 truncate">
                              {{ node.tag }}
                            </p>
                            <Badge variant="info" size="sm">{{ node.type }}</Badge>
                          </div>
                          <p class="text-xs text-gray-500 truncate">
                            {{ (node as any).server || '-' }}:{{ (node as any).server_port || '-' }}
                          </p>
                        </div>
                      </div>
                    </div>

                    <div class="flex justify-between items-center mt-4 pt-4 border-t">
                      <Button
                        variant="ghost"
                        @click="() => { parsedNodes = []; selectedNodes.clear(); parseError = '' }"
                      >
                        Back to Input
                      </Button>
                      <div class="flex gap-3">
                        <Button
                          variant="secondary"
                          @click="closeImportModal"
                        >
                          Cancel
                        </Button>
                        <Button
                          variant="primary"
                          :loading="importing"
                          :disabled="importing || selectedNodes.size === 0"
                          @click="handleImport"
                        >
                          {{ importing ? 'Importing...' : `Import ${selectedNodes.size} Nodes` }}
                        </Button>
                      </div>
                    </div>
                  </div>
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
                    Delete Outbound
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
                  Are you sure you want to delete the outbound
                  <strong>{{ deletingOutbound?.tag }}</strong>?
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
