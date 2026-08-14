<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DNSServer } from '../types/api'
import Button from './Button.vue'
import Input from './Input.vue'
import Badge from './Badge.vue'
import Modal from './Modal.vue'
import { Select } from '../volt'
import { PlusIcon, PencilIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'primevue'
import { useDNSStore } from '../stores/dns'
import { storeToRefs } from 'pinia'

const toast = useToast()
const { t } = useI18n()
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

const serverTypes = computed(() => [
  { value: 'udp', label: t('dns.servers.types.udp') },
  { value: 'tcp', label: t('dns.servers.types.tcp') },
  { value: 'tls', label: t('dns.servers.types.tls') },
  { value: 'https', label: t('dns.servers.types.https') },
  { value: 'http3', label: t('dns.servers.types.http3') },
  { value: 'quic', label: t('dns.servers.types.quic') },
  { value: 'local', label: t('dns.servers.types.local') },
  { value: 'dhcp', label: t('dns.servers.types.dhcp') },
  { value: 'fakeip', label: t('dns.servers.types.fakeip') },
  { value: 'hosts', label: t('dns.servers.types.hosts') },
])

const getServerTypeLabel = (type: string) => {
  return serverTypes.value.find(s => s.value === type)?.label || type
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
    if (n > 0) parts.push(t('dns.servers.hostsSummary.predefined', { n }))
  }
  const path = server?.path
  if (Array.isArray(path) && path.length > 0) {
    parts.push(t(path.length === 1 ? 'dns.servers.hostsSummary.file' : 'dns.servers.hostsSummary.files', { n: path.length }))
  } else if (typeof path === 'string' && path) {
    parts.push(t('dns.servers.hostsSummary.file', { n: 1 }))
  }
  return parts.length > 0 ? parts.join(', ') : t('dns.servers.hostsSummary.empty')
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
    const server = currentServer.value
    if (!server) return
    const map: Record<string, string | string[]> = {}
    for (const raw of text.split('\n')) {
      const line = raw.trim()
      if (!line || line.startsWith('#')) continue
      // First token is the hostname; remainder (split on whitespace OR comma)
      // is the IP list.
      const m = line.match(/^(\S+)\s+(.+)$/)
      if (!m) continue
      const host = m[1]
      const rawIps = m[2]
      if (!host || !rawIps) continue
      const ips = rawIps.split(/[\s,]+/).filter(Boolean)
      if (ips.length === 0) continue
      map[host] = ips.length === 1 ? ips[0]! : ips
    }
    // Drop the field entirely when no entries remain, matching the
    // `omitempty` behaviour on the backend's HostsDNSServerOptions.
    if (Object.keys(map).length === 0) delete server.predefined
    else server.predefined = map
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
        summary: t('common.success'),
        detail: t('dns.servers.toast.updatedOk'),
        life: 3000
      })
    } else {
      await dnsStore.addDNSServer(currentServer.value)
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('dns.servers.toast.addedOk'),
        life: 3000
      })
    }
    closeServerModal()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('dns.servers.toast.saveFailed'),
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
      summary: t('common.success'),
      detail: t('dns.servers.toast.deletedOk'),
      life: 3000
    })
    closeDeleteConfirm()
  } catch (err: any) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: err.message || t('dns.servers.toast.deleteFailed'),
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
        {{ $t('dns.servers.add') }}
      </Button>
    </div>

    <!-- DNS Servers Table -->
    <div class="bg-white dark:bg-slate-800 rounded-surface shadow overflow-hidden">
      <div class="px-6 py-4 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ $t('dns.servers.heading') }}</h3>
      </div>

      <div v-if="loading && dnsServers.length === 0" class="flex items-center justify-center py-12">
        <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
      </div>

      <div v-else-if="dnsServers.length === 0" class="text-center py-12">
        <p class="text-gray-500 dark:text-gray-500 mb-4">{{ $t('dns.servers.empty') }}</p>
        <Button @click="openAddServerModal" variant="primary" size="sm">
          <PlusIcon class="h-4 w-4 mr-2" />
          {{ $t('dns.servers.addFirst') }}
        </Button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
          <thead class="bg-gray-50 dark:bg-gray-700">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.servers.table.tag') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.servers.table.type') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.servers.table.server') }}</th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.servers.table.port') }}</th>
              <th class="px-4 py-2 text-right text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">{{ $t('dns.servers.table.actions') }}</th>
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
                <div class="dns-server-table-actions flex items-center justify-end gap-2">
                  <Button @click="openEditServerModal(server)" variant="ghost" size="sm" action>
                    <PencilIcon class="h-4 w-4" />
                  </Button>
                  <Button @click="openDeleteConfirm(server)" variant="ghost" size="sm" action class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300">
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
    <Modal
      :model-value="showServerModal"
      @update:model-value="(v: boolean) => { if (!v) closeServerModal() }"
      :title="isEditMode ? $t('dns.servers.modal.edit') : $t('dns.servers.modal.add')"
      size="md"
      show-close
    >
      <div class="space-y-4">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.tag') }}</label>
            <Input
              v-model="currentServer.tag"
              :placeholder="$t('dns.servers.form.tagPlaceholder')"
              :disabled="isEditMode"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.type') }}</label>
            <Select
              class="w-full"
              v-model="currentServer.type"
              :options="serverTypes"
              optionLabel="label"
              optionValue="value"
              :placeholder="$t('dns.servers.form.typePlaceholder')"
              :disabled="isEditMode"
            />
          </div>
        </div>

        <div v-if="needsServerAddress" class="grid grid-cols-3 gap-4">
          <div class="col-span-2">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.serverAddress') }}</label>
            <Input
              v-model="currentServer.server"
              :placeholder="$t('dns.servers.form.serverAddressPlaceholder')"
            />
          </div>

          <div class="col-span-1">
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.port') }}</label>
            <Input
              v-model.number="currentServer.server_port"
              type="number"
              :placeholder="$t('dns.servers.form.portPlaceholder')"
            />
          </div>
        </div>

        <div v-if="needsPath">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.path') }}</label>
          <Input
            v-model="currentServer.path"
            :placeholder="$t('dns.servers.form.pathPlaceholder')"
          />
          <p class="mt-1 text-xs text-gray-500">{{ $t('dns.servers.form.pathHelp') }}</p>
        </div>

        <div v-if="currentServer.type === 'dhcp'">
          <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.interface') }}</label>
          <Input
            v-model="currentServer.interface"
            :placeholder="$t('dns.servers.form.interfacePlaceholder')"
          />
        </div>

        <div v-if="currentServer.type === 'fakeip'" class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.inet4Range') }}</label>
            <Input
              v-model="currentServer.inet4_range"
              :placeholder="$t('dns.servers.form.inet4RangePlaceholder')"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.inet6Range') }}</label>
            <Input
              v-model="currentServer.inet6_range"
              :placeholder="$t('dns.servers.form.inet6RangePlaceholder')"
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
              {{ $t('dns.servers.form.predefinedHosts') }}
            </label>
            <textarea
              v-model="predefinedHostsText"
              rows="6"
              class="w-full px-3 py-2 text-sm font-mono border border-gray-300 dark:border-gray-600 rounded-control bg-white dark:bg-slate-900 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              placeholder="home.example.com 192.168.1.10
nas.example.com 192.168.1.20,192.168.1.21"
            />
            <p class="mt-1 text-xs text-gray-500">
              <i18n-t keypath="dns.servers.form.predefinedHostsHelp" scope="global">
                <template #format><code>hostname IP[,IP2,...]</code></template>
                <template #hash><code>#</code></template>
              </i18n-t>
            </p>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {{ $t('dns.servers.form.hostsFilePaths') }} <span class="font-normal text-gray-500">{{ $t('dns.servers.form.hostsFilePathsOptional') }}</span>
            </label>
            <textarea
              v-model="hostsFilePathsText"
              rows="2"
              class="w-full px-3 py-2 text-sm font-mono border border-gray-300 dark:border-gray-600 rounded-control bg-white dark:bg-slate-900 text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              :placeholder="$t('dns.servers.form.hostsFilePathsPlaceholder')"
            />
            <p class="mt-1 text-xs text-gray-500">{{ $t('dns.servers.form.hostsFilePathsHelp') }}</p>
          </div>
        </div>
      </div>

      <template #footer>
        <Button @click="closeServerModal" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleSaveServer" variant="primary" :disabled="loading">
          {{ isEditMode ? $t('common.update') : $t('common.add') }}
        </Button>
      </template>
    </Modal>

    <!-- Delete Confirmation Modal -->
    <Modal
      :model-value="showDeleteConfirm"
      @update:model-value="(v: boolean) => { if (!v) closeDeleteConfirm() }"
      :title="$t('dns.servers.del.title')"
      size="sm"
      show-close
    >
      <p class="text-gray-700 dark:text-gray-300">
        <i18n-t keypath="dns.servers.del.confirm" scope="global">
          <template #tag><strong>{{ deletingServer?.tag }}</strong></template>
        </i18n-t>
      </p>

      <template #footer>
        <Button @click="closeDeleteConfirm" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleDeleteServer" variant="danger" :disabled="loading">
          {{ $t('common.delete') }}
        </Button>
      </template>
    </Modal>
  </div>
</template>
