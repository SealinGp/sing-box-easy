<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  TransitionRoot,
  TransitionChild,
  Dialog,
  DialogPanel,
  DialogTitle,
} from '@headlessui/vue'
import type { DNSServer } from '../types/api'
import Button from './Button.vue'
import Input from './Input.vue'
import Badge from './Badge.vue'
import { PlusIcon, PencilIcon, TrashIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'primevue'
import { useDNSStore } from '../stores/dns'
import { storeToRefs } from 'pinia'

const toast = useToast()
const dnsStore = useDNSStore()
const { dnsServers, loading } = storeToRefs(dnsStore)

// Modal state
const showServerModal = ref(false)
const isEditMode = ref(false)
const editingServerTag = ref<string>('')
const currentServer = ref<any>({
  type: 'udp',
  tag: '',
  server: '',
  server_port: 53,
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

// ─── hosts-type specific helpers ─────────────────────────────────────────────
// Render a short summary of the row data when type=hosts, so the table cell
// isn't blank ("-"). Server/port aren't meaningful for hosts entries; what
// matters is how many predefined hostnames and/or hosts-file paths it has.
const formatHostsSummary = (server: any): string => {
  const parts: string[] = []
  const predefined = server?.predefined as Record<string, string | string[]> | undefined
  if (predefined) {
    const n = Object.keys(predefined).length
    if (n > 0) parts.push(`${n} predefined`)
  }
  const path = server?.path
  if (Array.isArray(path) && path.length > 0) parts.push(`${path.length} file${path.length === 1 ? '' : 's'}`)
  else if (typeof path === 'string' && path) parts.push('1 file')
  return parts.length > 0 ? parts.join(', ') : '(empty)'
}

// Two-way bridge between the `predefined` object and a textarea where each
// line is `hostname IP[,IP2,...]`. We keep the format simple and forgiving:
//   - blank lines and #-comments are skipped
//   - whitespace between hostname and IPs collapses to a single delimiter
//   - multiple IPs may be separated by commas or spaces
// Round-trips cleanly for the common single-IP case in config.json.
const predefinedHostsText = computed<string>({
  get(): string {
    const map = currentServer.value?.predefined as Record<string, string | string[]> | undefined
    if (!map) return ''
    return Object.entries(map)
      .map(([host, val]) => `${host} ${Array.isArray(val) ? val.join(',') : val}`)
      .join('\n')
  },
  set(text: string) {
    const map: Record<string, string | string[]> = {}
    for (const raw of text.split('\n')) {
      const line = raw.trim()
      if (!line || line.startsWith('#')) continue
      // First token is the hostname; remainder (split on whitespace OR comma)
      // is the IP list.
      const m = line.match(/^(\S+)\s+(.+)$/)
      if (!m) continue
      const host = m[1]
      const ips = m[2].split(/[\s,]+/).filter(Boolean)
      if (ips.length === 0) continue
      map[host] = ips.length === 1 ? ips[0] : ips
    }
    // Drop the field entirely when no entries remain, matching the
    // `omitempty` behaviour on the backend's HostsDNSServerOptions.
    if (Object.keys(map).length === 0) delete currentServer.value.predefined
    else currentServer.value.predefined = map
  },
})

// Similar bridge for the optional `path` field (list of hosts-file paths).
const hostsFilePathsText = computed<string>({
  get(): string {
    const p = currentServer.value?.path
    if (!p) return ''
    return Array.isArray(p) ? p.join('\n') : String(p)
  },
  set(text: string) {
    const paths = text
      .split('\n')
      .map(s => s.trim())
      .filter(Boolean)
    if (paths.length === 0) delete currentServer.value.path
    else currentServer.value.path = paths.length === 1 ? paths[0] : paths
  },
})

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
  try {
    if (isEditMode.value) {
      await dnsStore.updateDNSServer(editingServerTag.value, currentServer.value)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'DNS server updated successfully',
        life: 3000
      })
    } else {
      await dnsStore.addDNSServer(currentServer.value)
      toast.add({
        severity: 'success',
        summary: 'Success',
        detail: 'DNS server added successfully',
        life: 3000
      })
    }
    closeServerModal()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to save DNS server',
      life: 3000
    })
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
  try {
    await dnsStore.deleteDNSServer(deletingServer.value.tag)
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'DNS server deleted successfully',
      life: 3000
    })
    closeDeleteConfirm()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: err.message || 'Failed to delete DNS server',
      life: 3000
    })
  }
}

// Load data on mount
onMounted(() => {
  dnsStore.fetchDNSServers()
})
</script>

<template>
  <div>
    <div class="flex justify-end mb-2">
      <Button @click="openAddServerModal" variant="primary">
        <PlusIcon class="h-5 w-5 mr-2" />
        Add DNS Server
      </Button>
    </div>

    <!-- DNS Servers Table -->
    <div class="bg-white dark:bg-slate-800 rounded-lg shadow overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">DNS Servers</h3>
      </div>

      <div v-if="loading && dnsServers.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-violet-600"></div>
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
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Tag</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Type</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Server</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Port</th>
              <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody class="bg-white dark:bg-slate-800 divide-y divide-gray-200 dark:divide-gray-700">
            <tr v-for="server in dnsServers" :key="server.tag" class="hover:bg-gray-50 dark:hover:bg-gray-700">
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm font-medium text-gray-900 dark:text-gray-100">{{ server.tag }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <Badge :variant="getServerBadgeVariant((server as any).type || 'udp')">
                  {{ getServerTypeLabel((server as any).type || 'udp') }}
                </Badge>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <!-- For type=hosts, server/port aren't meaningful; show a
                     summary of predefined hosts + file paths instead. -->
                <div
                  v-if="(server as any).type === 'hosts'"
                  class="text-sm text-gray-600 dark:text-gray-400 italic"
                  :title="(server as any).predefined ? Object.keys((server as any).predefined).join(', ') : ''"
                >
                  {{ formatHostsSummary(server) }}
                </div>
                <div v-else class="text-sm text-gray-900 dark:text-gray-100">{{ (server as any).server || '-' }}</div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap">
                <div class="text-sm text-gray-900 dark:text-gray-100">
                  {{ (server as any).type === 'hosts' ? '—' : ((server as any).server_port || '-') }}
                </div>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                <div class="flex items-center justify-end gap-2">
                  <Button @click="openEditServerModal(server)" variant="ghost" size="sm">
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button @click="openDeleteConfirm(server)" variant="ghost" size="sm" class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300">
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
                        <option v-for="serverType in serverTypes" :key="serverType.value" :value="serverType.value">
                          {{ serverType.label }}
                        </option>
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

                  <!--
                    Editor for type=hosts. Two-way bridged through computed
                    refs (predefinedHostsText, hostsFilePathsText). Format is
                    intentionally /etc/hosts-like: one entry per line,
                    `hostname IP[,IP2,...]`. Multiple IPs per host stay an
                    array on the wire, matching badoption.Listable on the
                    backend.
                  -->
                  <div v-if="currentServer.type === 'hosts'" class="space-y-4">
                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Predefined Hosts
                      </label>
                      <textarea
                        v-model="predefinedHostsText"
                        rows="6"
                        class="w-full px-3 py-2 text-sm font-mono border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-slate-900 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:ring-2 focus:ring-violet-500 focus:border-violet-500"
                        placeholder="home.example.com 192.168.1.10
nas.example.com 192.168.1.20,192.168.1.21"
                      />
                      <p class="mt-1 text-xs text-gray-500">
                        One mapping per line: <code>hostname IP[,IP2,...]</code>. Lines starting with <code>#</code> and blank lines are ignored.
                      </p>
                    </div>

                    <div>
                      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                        Hosts File Paths <span class="font-normal text-gray-500">(optional)</span>
                      </label>
                      <textarea
                        v-model="hostsFilePathsText"
                        rows="2"
                        class="w-full px-3 py-2 text-sm font-mono border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-slate-900 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:ring-2 focus:ring-violet-500 focus:border-violet-500"
                        placeholder="/etc/hosts"
                      />
                      <p class="mt-1 text-xs text-gray-500">One path per line. Loaded in addition to the predefined mappings above.</p>
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
