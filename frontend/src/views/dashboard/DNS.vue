<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import {
  TransitionRoot,
  TransitionChild,
  Dialog,
  DialogPanel,
  DialogTitle,
} from '@headlessui/vue'
import type { DNSServer, DNS } from '../../types/api'
import Button from '../../components/Button.vue'
import Input from '../../components/Input.vue'
import Badge from '../../components/Badge.vue'
import Card from '../../components/Card.vue'
import Select from '../../components/Select.vue'
import { PlusIcon, PencilIcon, TrashIcon, XMarkIcon, Cog6ToothIcon } from '@heroicons/vue/24/outline'
import { dnsService } from '../../services'
import { useToast } from 'primevue'
const dnsServers = ref<DNSServer[]>([])
const dnsConfig = ref<DNS | null>(null)
const loading = ref(false)
const toast = useToast()

// DNS Server Modal state
const showServerModal = ref(false)
const isEditMode = ref(false)
const editingServerTag = ref<string>('')
const currentServer = ref<any>({
  type: 'udp',
  tag: '',
  server: '',
  server_port: 53,
})

// DNS Settings Modal
const showSettingsModal = ref(false)
const currentSettings = ref<any>({
  strategy: 'prefer_ipv4',
  disable_cache: false,
  disable_expire: false,
  final: '',
})

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingServer = ref<DNSServer | null>(null)

const serverTypes = [
  { value: 'udp', label: 'UDP' },
  { value: 'tcp', label: 'TCP' },
  { value: 'tls', label: 'DNS over TLS' },
  { value: 'https', label: 'DNS over HTTPS' },
  { value: 'http3', label: 'DNS over HTTP/3' },
  { value: 'quic', label: 'DNS over QUIC' },
  { value: 'local', label: 'Local System DNS' },
  { value: 'dhcp', label: 'DHCP' },
  { value: 'fakeip', label: 'FakeIP' },
  { value: 'hosts', label: 'Hosts File' },
]

const strategyOptions = [
  { value: 'prefer_ipv4', label: 'Prefer IPv4' },
  { value: 'prefer_ipv6', label: 'Prefer IPv6' },
  { value: 'ipv4_only', label: 'IPv4 Only' },
  { value: 'ipv6_only', label: 'IPv6 Only' },
]

const getServerTypeLabel = (type: string) => {
  return serverTypes.find(t => t.value === type)?.label || type
}

const getServerBadgeVariant = (type: string): 'primary' | 'success' | 'warning' | 'info' | 'secondary' => {
  if (type === 'local') return 'success'
  if (type === 'fakeip') return 'warning'
  if (type === 'udp' || type === 'tcp') return 'primary'
  return 'info'
}

const needsServerAddress = computed(() => {
  const types = ['udp', 'tcp', 'tls', 'https', 'http3', 'quic']
  return types.indexOf(currentServer.value.type) !== -1
})

const needsPath = computed(() => {
  const types = ['https', 'http3']
  return types.indexOf(currentServer.value.type) !== -1
})

const fetchDNSServers = async () => {
  loading.value = true
  try {
    const { data } = await dnsService.getDNSServers()
    dnsServers.value = data.servers
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to fetch DNS servers',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const fetchDNSConfig = async () => {
  try {
    const { data } = await dnsService.getDNS()
    dnsConfig.value = data
    if (dnsConfig.value) {
      currentSettings.value = {
        strategy: dnsConfig.value.strategy || 'prefer_ipv4',
        disable_cache: dnsConfig.value.disable_cache || false,
        disable_expire: dnsConfig.value.disable_expire || false,
        final: dnsConfig.value.final || '',
      }
    }
  } catch (err: any) {
    console.error('Failed to fetch DNS config:', err)
  }
}

const openAddServerModal = () => {
  isEditMode.value = false
  currentServer.value = {
    type: 'udp',
    tag: '',
    server: '',
    server_port: 53,
  }
  showServerModal.value = true
}

const openEditServerModal = (server: DNSServer) => {
  isEditMode.value = true
  editingServerTag.value = server.tag
  currentServer.value = { ...server }
  showServerModal.value = true
}

const closeServerModal = () => {
  showServerModal.value = false
  currentServer.value = {
    type: 'udp',
    tag: '',
    server: '',
    server_port: 53,
  }
}

const handleSaveServer = async () => {
  // Validation
  if (!currentServer.value.tag?.trim()) {
    toast.add({
      severity: 'error',
      summary: 'Validation Error',
      detail: 'Tag is required',
      life: 3000
    })
    return
  }

  if (!currentServer.value.type) {
    toast.add({
      severity: 'error',
      summary: 'Validation Error',
      detail: 'Type is required',
      life: 3000
    })
    return
  }

  if (needsServerAddress.value && !currentServer.value.server?.trim()) {
    toast.add({
      severity: 'error',
      summary: 'Validation Error',
      detail: 'Server address is required',
      life: 3000
    })
    return
  }

  loading.value = true
  try {
    if (isEditMode.value) {
      await dnsService.updateDNSServer(editingServerTag.value, currentServer.value)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'DNS server updated successfully',
        life: 300000
      })
    } else {
      await dnsService.addDNSServer(currentServer.value)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'DNS server added successfully',
        life: 3000
      })
    }
    closeServerModal()
    await fetchDNSServers()
  } catch (err: any) {
    console.error('Failed to save DNS server:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to save DNS server',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const openDeleteConfirm = (server: DNSServer) => {
  deletingServer.value = server
  showDeleteConfirm.value = true
}

const closeDeleteConfirm = () => {
  showDeleteConfirm.value = false
  deletingServer.value = null
}

const handleDeleteServer = async () => {
  if (!deletingServer.value) return

  loading.value = true

  try {
    await dnsService.deleteDNSServer(deletingServer.value.tag)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS server deleted successfully',
      life: 3000
    })
    closeDeleteConfirm()
    await fetchDNSServers()
  } catch (err: any) {
    console.error('Failed to delete DNS server:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to delete DNS server',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

const openSettingsModal = () => {
  showSettingsModal.value = true
}

const closeSettingsModal = () => {
  showSettingsModal.value = false
}

const handleSaveSettings = async () => {
  loading.value = true

  try {
    const updatedDNS = {
      ...dnsConfig.value,
      ...currentSettings.value,
    }
    await dnsService.updateDNS(updatedDNS)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS settings updated successfully',
      life: 3000
    })
    closeSettingsModal()
    await fetchDNSConfig()
  } catch (err: any) {
    console.error('Failed to update DNS settings:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.response?.data?.error || 'Failed to update DNS settings',
      life: 3000
    })
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await fetchDNSServers()
  await fetchDNSConfig()
})
</script>

<template>
  <div class="p-8">
 

    <div class="flex justify-between items-center mb-6">
      <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100">DNS Configuration</h2>
      <div class="flex gap-3">
        <Button @click="openSettingsModal" variant="secondary">
          <Cog6ToothIcon class="h-5 w-5 mr-2" />
          Settings
        </Button>
        <Button @click="openAddServerModal" variant="primary">
          <PlusIcon class="h-5 w-5 mr-2" />
          Add DNS Server
        </Button>
      </div>
    </div>

    <!-- DNS Global Settings Card -->
    <Card class="mb-6">
      <div class="flex justify-between items-start">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2">Global DNS Settings</h3>
          <div class="space-y-2 text-sm text-gray-600 dark:text-gray-400">
            <p><span class="font-medium">Strategy:</span> {{ dnsConfig?.strategy || 'prefer_ipv4' }}</p>
            <p><span class="font-medium">Cache:</span> {{ dnsConfig?.disable_cache ? 'Disabled' : 'Enabled' }}</p>
            <p><span class="font-medium">Final Server:</span> {{ dnsConfig?.final || 'Not set' }}</p>
          </div>
        </div>
        <Button @click="openSettingsModal" variant="ghost" size="sm">
          <PencilIcon class="h-4 w-4" />
        </Button>
      </div>
    </Card>

    <!-- DNS Servers Table -->
    <div class="bg-white dark:bg-slate-800 rounded-lg shadow overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">DNS Servers</h3>
      </div>

      <div v-if="loading && dnsServers.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>

      <div v-else-if="dnsServers.length === 0" class="text-center py-12">
        <p class="text-gray-500 dark:text-gray-500 mb-4">No DNS servers configured</p>
        <Button @click="openAddServerModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          Add Your First DNS Server
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-slate-900">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">Tag</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">Type</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">Server</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">Port</th>
              <th class="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-for="server in dnsServers" :key="server.tag" class="hover:bg-gray-50 dark:hover:bg-slate-700">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ server.tag }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge :variant="getServerBadgeVariant((server as any).type || 'udp')">
                  {{ getServerTypeLabel((server as any).type || 'udp') }}
                </Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-gray-100">{{ (server as any).server || '-' }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-gray-100">{{ (server as any).server_port || '-' }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="flex items-center justify-end gap-2">
                  <Button @click="openEditServerModal(server)" variant="ghost" size="sm">
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button @click="openDeleteConfirm(server)" variant="ghost" size="sm" class="text-red-600 hover:text-red-700">
                    <TrashIcon class="h-4 w-4" />
                  </Button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Add/Edit DNS Server Modal -->
    <TransitionRoot appear :show="showServerModal" as="template">
      <Dialog as="div" @close="closeServerModal" class="relative z-50">
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
                    {{ isEditMode ? 'Edit DNS Server' : 'Add DNS Server' }}
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeServerModal"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

                <div class="space-y-4">
                  <div class="grid grid-cols-2 gap-4">
                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Tag *</label>
                      <Input
                        v-model="currentServer.tag"
                        placeholder="e.g., cloudflare"
                        :disabled="isEditMode"
                      />
                    </div>

                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Type *</label>

                      <select class="select" v-model="currentServer.type" :disabled="isEditMode">
                        <option disabled selected>Pick dns type</option>
                        <option v-for="serverType in serverTypes" :value="serverType.value" >{{ serverType.label }}</option>
                      </select>
                    </div>
                  </div>

                  <div v-if="needsServerAddress" class="grid grid-cols-3 gap-4">
                    <div class="col-span-2">
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Server Address *</label>
                      <Input
                        v-model="currentServer.server"
                        placeholder="1.1.1.1 or dns.cloudflare.com"
                      />
                    </div>

                    <div class="col-span-1">
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Port</label>
                      <Input
                        v-model.number="currentServer.server_port"
                        type="number"
                        placeholder="53"
                      />
                    </div>
                  </div>

                  <div v-if="needsPath">
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Path</label>
                    <Input
                      v-model="currentServer.path"
                      placeholder="/dns-query"
                    />
                    <p class="mt-1 text-xs text-gray-500">URL path for DoH queries</p>
                  </div>

                  <div v-if="currentServer.type === 'dhcp'">
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Interface</label>
                    <Input
                      v-model="currentServer.interface"
                      placeholder="e.g., eth0"
                    />
                  </div>

                  <div v-if="currentServer.type === 'fakeip'" class="space-y-4">
                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">IPv4 Range</label>
                      <Input
                        v-model="currentServer.inet4_range"
                        placeholder="198.18.0.0/15"
                      />
                    </div>
                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">IPv6 Range</label>
                      <Input
                        v-model="currentServer.inet6_range"
                        placeholder="fc00::/18"
                      />
                    </div>
                  </div>
                </div>

                <div class="mt-6 flex justify-end gap-3">
                  <Button @click="closeServerModal" variant="secondary">Cancel</Button>
                  <Button @click="handleSaveServer" variant="primary" :disabled="loading">
                    {{ isEditMode ? 'Update' : 'Add' }}
                  </Button>
                </div>
              </DialogPanel>
            </TransitionChild>
          </div>
        </div>
      </Dialog>
    </TransitionRoot>

    <!-- DNS Settings Modal -->
    <TransitionRoot appear :show="showSettingsModal" as="template">
      <Dialog as="div" @close="closeSettingsModal" class="relative z-50">
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
                    DNS Global Settings
                  </DialogTitle>
                  <button
                    type="button"
                    class="text-gray-400 hover:text-gray-500 transition-colors"
                    @click="closeSettingsModal"
                  >
                    <XMarkIcon class="h-6 w-6" />
                  </button>
                </div>

                <div class="space-y-4">
                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Domain Strategy</label>
                    <Select v-model="currentSettings.strategy" :options="strategyOptions" />
                    <p class="mt-1 text-xs text-gray-500">IP version preference for DNS queries</p>
                  </div>

                  <div>
                    <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Final DNS Server</label>
                    <Input
                      v-model="currentSettings.final"
                      placeholder="Tag of the default DNS server"
                    />
                    <p class="mt-1 text-xs text-gray-500">Fallback server when no rules match</p>
                  </div>

                  <div class="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="disable_cache"
                      v-model="currentSettings.disable_cache"
                      class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                    <label for="disable_cache" class="text-sm font-medium text-gray-700 dark:text-gray-300">Disable DNS Cache</label>
                  </div>

                  <div class="flex items-center gap-2">
                    <input
                      type="checkbox"
                      id="disable_expire"
                      v-model="currentSettings.disable_expire"
                      class="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                    />
                    <label for="disable_expire" class="text-sm font-medium text-gray-700 dark:text-gray-300">Disable Cache Expiration</label>
                  </div>
                </div>

                <div class="mt-6 flex justify-end gap-3">
                  <Button @click="closeSettingsModal" variant="secondary">Cancel</Button>
                  <Button @click="handleSaveSettings" variant="primary" :disabled="loading">
                    Save Settings
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
                    Delete DNS Server
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
                  Are you sure you want to delete the DNS server
                  <strong>{{ deletingServer?.tag }}</strong>?
                  This action cannot be undone.
                </p>

                <div class="mt-6 flex justify-end gap-3">
                  <Button @click="closeDeleteConfirm" variant="secondary">Cancel</Button>
                  <Button @click="handleDeleteServer" variant="danger" :disabled="loading">
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
