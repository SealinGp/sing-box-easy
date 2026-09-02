<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import type { DNSServer } from '../types/api'
import Button from './Button.vue'
import Input from './Input.vue'
import Badge from './Badge.vue'
import Table from './Table.vue'
import { Dialog, Select } from '../volt'
import SchemaFieldsEditor from './SchemaFieldsEditor.vue'
import { PlusIcon, PencilIcon, TrashIcon } from '@heroicons/vue/24/outline'
import { useToast } from 'primevue'
import {
  DNS_SERVER_TYPE_NAMES,
  applyTypeDefaults,
  pruneForType,
  resolveDNSServerFields,
  DNS_TYPES_NEEDING_SERVER,
  type DNSServerTypeName,
} from '../schemas/dnsServerFields'
import { useDNSStore } from '../stores/dns'
import { storeToRefs } from 'pinia'

const toast = useToast()
const { t, te } = useI18n()
const dnsStore = useDNSStore()
const { dnsServers, loading } = storeToRefs(dnsStore)

// Modal state
const showServerModal = ref(false)
const isEditMode = ref(false)
const editingServerTag = ref<string>('')
const DEFAULT_TYPE: DNSServerTypeName = 'udp'

function blankServer(): Record<string, unknown> {
  return applyTypeDefaults({ tag: '' }, DEFAULT_TYPE)
}

const currentServer = ref<Record<string, unknown>>(blankServer())

/**
 * The form model is an open record because a server's fields depend on its
 * type, and the declared DNSServer union cannot narrow on `type`. This is the
 * one place the shape is read back as a type name.
 */
const currentType = computed(() =>
  typeof currentServer.value.type === 'string'
    ? (currentServer.value.type as DNSServerTypeName)
    : undefined,
)

// Delete confirmation
const showDeleteConfirm = ref(false)
const deletingServer = ref<DNSServer | null>(null)

/**
 * Driven by the generated inventory rather than a hand-written array.
 *
 * That array spelled HTTP/3 "http3"; sing-box's constant is "h3" and its
 * transport registers under that name, so every server this form saved as
 * "http3" produced a config sing-box could not start. It also predated
 * `tailscale`. Deriving the list means neither can drift again.
 *
 * Labels stay opt-in: a type with no `dns.servers.types.*` entry shows its own
 * name rather than a raw i18n path.
 */
const serverTypes = computed(() =>
  DNS_SERVER_TYPE_NAMES.map((value) => ({
    value,
    label: te(`dns.servers.types.${value}`) ? t(`dns.servers.types.${value}`) : value,
  })),
)

/**
 * Switching type prunes the previous type's fields before seeding the new
 * one's defaults. There was no type-change handler at all before, so switching
 * from `https` to `local` in the add dialog carried `server`, `server_port` and
 * `path` along into the payload — and sing-box decodes DNS options strictly.
 */
function changeType(next: unknown) {
  if (typeof next !== 'string') return
  currentServer.value = pruneForType(currentServer.value, next as DNSServerTypeName)
}

const getServerTypeLabel = (type: string) => {
  return serverTypes.value.find(s => s.value === type)?.label || type
}

const getServerBadgeVariant = (type: string): 'primary' | 'success' | 'warning' | 'info' | 'secondary' => {
  if (type === 'local') return 'success'
  if (type === 'fakeip') return 'warning'
  if (type === 'udp' || type === 'tcp') return 'primary'
  return 'info'
}

/*
 * needsServerAddress / supportsDetour / needsPath used to live here as
 * hand-maintained type lists. Two of them were byte-identical. All three are
 * now answered by the generated inventory: a type has `server` if its option
 * struct embeds DNSServerAddressOptions and `detour` if it embeds
 * DialerOptions, which is exactly what the reflection sees. See
 * schemas/dnsServerFields.ts.
 */

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

/**
 * Client-side validation, mirroring the rules the backend now enforces in
 * dns_validation.go. Returns an i18n key, or '' when valid.
 *
 * Kept in step with the server deliberately: the backend is the authority (it
 * is what sing-box actually parses), but catching it here means the operator
 * sees the problem next to the field instead of as a toast after a round trip.
 */
function validateDNSServer(server: Record<string, unknown>): string {
  const tag = server.tag
  if (typeof tag !== 'string' || !tag.trim()) return 'dns.servers.validation.tagRequired'

  const type = server.type
  if (typeof type !== 'string' || !type) return 'dns.servers.validation.typeRequired'

  if (DNS_TYPES_NEEDING_SERVER.includes(type as DNSServerTypeName)) {
    const address = server.server
    if (typeof address !== 'string' || !address.trim()) {
      return 'dns.servers.validation.serverRequired'
    }
  }

  return ''
}

const openAddServerModal = () => {
  isEditMode.value = false
  currentServer.value = blankServer()
  showServerModal.value = true
}

const openEditServerModal = (server: DNSServer) => {
  isEditMode.value = true
  editingServerTag.value = server.tag
  // No defaults seeded on edit: writing one into a config that deliberately
  // omitted the key would change behaviour before the operator touched
  // anything. Any field holding a value still renders.
  currentServer.value = { ...(server as unknown as Record<string, unknown>) }
  showServerModal.value = true
}

const closeServerModal = () => {
  showServerModal.value = false
  currentServer.value = blankServer()
}

const handleSaveServer = async () => {
  const error = validateDNSServer(currentServer.value)
  if (error) {
    toast.add({
      severity: 'error',
      summary: t('common.error'),
      detail: t(error),
      life: 3000,
    })
    return
  }

  try {
    if (isEditMode.value) {
      await dnsStore.updateDNSServer(
        editingServerTag.value,
        currentServer.value as unknown as DNSServer,
      )
      toast.add({
        severity: 'success',
        summary: t('common.success'),
        detail: t('dns.servers.toast.updatedOk'),
        life: 3000
      })
    } else {
      await dnsStore.addDNSServer(currentServer.value as unknown as DNSServer)
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
  // The detour picker's outbound list is fetched by SchemaFieldControl, and
  // only when a detour field is actually rendered — so opening this page no
  // longer costs an outbounds request it may never use.
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
      <div class="px-4 py-3 border-b border-gray-200 dark:border-gray-700">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">{{ $t('dns.servers.heading') }}</h3>
      </div>

      <Table :loading="loading && dnsServers.length === 0" :empty="dnsServers.length === 0">
        <template #empty>
          <p class="text-gray-500 dark:text-gray-500 mb-3">{{ $t('dns.servers.empty') }}</p>
          <Button @click="openAddServerModal" variant="primary" size="sm">
            <PlusIcon class="h-4 w-4 mr-1.5" />
            {{ $t('dns.servers.addFirst') }}
          </Button>
        </template>

        <template #head>
          <th>{{ $t('dns.servers.table.tag') }}</th>
          <th>{{ $t('dns.servers.table.type') }}</th>
          <th>{{ $t('dns.servers.table.server') }}</th>
          <th>{{ $t('dns.servers.table.port') }}</th>
          <th>{{ $t('dns.servers.table.detour') }}</th>
          <th class="col-actions">{{ $t('dns.servers.table.actions') }}</th>
        </template>

        <tr v-for="server in dnsServers" :key="server.tag">
          <td class="font-medium text-gray-900 dark:text-gray-100">{{ server.tag }}</td>
          <td>
            <Badge :variant="getServerBadgeVariant((server as any).type || 'udp')">
              {{ getServerTypeLabel((server as any).type || 'udp') }}
            </Badge>
          </td>
          <td>
            <!-- For type=hosts, server/port aren't meaningful; show a
                 summary of predefined hosts + file paths instead. -->
            <div
              v-if="(server as any).type === 'hosts'"
              class="text-gray-600 dark:text-gray-400 italic"
              :title="(server as any).predefined ? Object.keys((server as any).predefined).join(', ') : ''"
            >
              {{ formatHostsSummary(server) }}
            </div>
            <div v-else class="text-gray-900 dark:text-gray-100">{{ (server as any).server || '-' }}</div>
          </td>
          <td class="text-gray-900 dark:text-gray-100">
            {{ (server as any).type === 'hosts' ? '—' : ((server as any).server_port || '-') }}
          </td>
          <td>
            <!-- Whether this resolver's queries go through a proxy is the
                 difference between working and censored foreign DNS, so it
                 belongs in the table rather than only in the modal. -->
            <div v-if="(server as any).detour" class="text-gray-900 dark:text-gray-100 truncate max-w-[12rem]" :title="(server as any).detour">
              {{ (server as any).detour }}
            </div>
            <div v-else class="text-gray-400 dark:text-gray-500">{{ $t('dns.servers.form.detourDirect') }}</div>
          </td>
          <td class="col-actions font-medium">
            <div class="flex items-center justify-end gap-1">
              <Button @click="openEditServerModal(server)" variant="ghost" size="sm" action>
                <PencilIcon class="h-4 w-4" />
              </Button>
              <Button @click="openDeleteConfirm(server)" variant="ghost" size="sm" action class="text-red-600 dark:text-red-400 hover:text-red-700 dark:hover:text-red-300">
                <TrashIcon class="h-4 w-4" />
              </Button>
            </div>
          </td>
        </tr>
      </Table>
    </div>

    <!-- Add/Edit DNS Server Modal -->
    <Dialog
      :visible="showServerModal"
      @update:visible="(v: boolean) => { if (!v) closeServerModal() }"
      :header="isEditMode ? $t('dns.servers.modal.edit') : $t('dns.servers.modal.add')"
      modal
      class="w-full max-w-lg"
    >
      <div class="space-y-3">
        <!--
          Tag and type live on the DNS server wrapper rather than in any type's
          option struct, so they are rendered here rather than coming from the
          schema. Both lock on edit: the tag is what `dns.rules` reference, and
          changing the type would reinterpret every remaining field.
        -->
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.tag') }}</label>
            <Input
              v-model="(currentServer.tag as string)"
              :placeholder="$t('dns.servers.form.tagPlaceholder')"
              :disabled="isEditMode"
            />
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{{ $t('dns.servers.form.type') }}</label>
            <Select
              class="w-full"
              :modelValue="currentServer.type"
              :options="serverTypes"
              optionLabel="label"
              optionValue="value"
              filter
              :filterPlaceholder="$t('common.search')"
              :emptyFilterMessage="$t('common.noMatch')"
              :placeholder="$t('dns.servers.form.typePlaceholder')"
              :disabled="isEditMode"
              @update:modelValue="changeType"
            />
          </div>
        </div>

        <!--
          :key remounts the editor on every type change, dropping the set of
          fields the operator had added. Without it, switching https -> local
          would leave https's fields on screen bound to keys the new type does
          not have.
        -->
        <SchemaFieldsEditor
          v-if="currentType"
          :key="currentType"
          v-model="currentServer"
          :fields="resolveDNSServerFields(currentType)"
          :empty-hint="$t('dns.servers.form.localHint')"
        />
      </div>

      <template #footer>
        <Button @click="closeServerModal" variant="secondary">{{ $t('common.cancel') }}</Button>
        <Button @click="handleSaveServer" variant="primary" :disabled="loading">
          {{ isEditMode ? $t('common.update') : $t('common.add') }}
        </Button>
      </template>
    </Dialog>

    <!-- Delete Confirmation Modal -->
    <Dialog
      :visible="showDeleteConfirm"
      @update:visible="(v: boolean) => { if (!v) closeDeleteConfirm() }"
      :header="$t('dns.servers.del.title')"
      modal
      class="w-full max-w-md"
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
    </Dialog>
  </div>
</template>
