<script setup lang="ts">
/**
 * Editor for a `hosts` DNS server's `predefined` map.
 *
 * On the wire this is a `badjson.TypedMap[string, Listable[netip.Addr]]` —
 * domain to one-or-more addresses — which the generator classifies as a plain
 * object, so without this it would land in the raw JSON editor. That is the
 * wrong control for the field most likely to be edited by hand: the live config
 * on this machine has eight entries, all single addresses.
 *
 * SCALAR-OR-ARRAY
 * ───────────────
 * `Listable[T]` accepts a bare value or an array, and sing-box round-trips
 * whatever shape the file used. Reading normalizes to an array so the editor
 * has one shape to work with; writing collapses a single address back to a
 * scalar so a config written by hand does not get noisily rewritten into
 * one-element arrays on the first save.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { PlusIcon, TrashIcon } from '@heroicons/vue/24/outline'
import Input from './Input.vue'
import Button from './Button.vue'

type HostsMap = Record<string, string | string[]>

const props = defineProps<{ modelValue?: HostsMap }>()
const emit = defineEmits<{ 'update:modelValue': [value: HostsMap | undefined] }>()

const { t } = useI18n()

interface Row {
  domain: string
  addresses: string
}

/** Rows are derived, never stored — the map is the single source of truth. */
const rows = computed<Row[]>(() =>
  Object.entries(props.modelValue ?? {}).map(([domain, value]) => ({
    domain,
    addresses: (Array.isArray(value) ? value : [value]).join(', '),
  })),
)

function toValue(addresses: string): string | string[] {
  const parts = addresses
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
  // Collapse back to a scalar when there is exactly one, matching how the
  // config was most likely written.
  return parts.length === 1 ? (parts[0] as string) : parts
}

/**
 * Rebuild the whole map from the edited rows.
 *
 * Insertion order is preserved by rebuilding in row order rather than patching
 * the existing object, so renaming a domain does not move its entry to the end
 * of the list under the cursor.
 */
function commit(next: Row[]) {
  const map: HostsMap = {}
  for (const row of next) {
    const domain = row.domain.trim()
    if (!domain) continue
    map[domain] = toValue(row.addresses)
  }
  emit('update:modelValue', Object.keys(map).length > 0 ? map : undefined)
}

function updateRow(index: number, patch: Partial<Row>) {
  commit(rows.value.map((row, i) => (i === index ? { ...row, ...patch } : row)))
}

function removeRow(index: number) {
  commit(rows.value.filter((_, i) => i !== index))
}

function addRow() {
  // A blank domain is dropped by commit(), so seed a placeholder key that the
  // operator immediately overwrites. Without it, adding a row would emit an
  // unchanged map and the new row would vanish on the next render.
  const map: HostsMap = { ...(props.modelValue ?? {}) }
  let key = 'new-host'
  let n = 2
  while (key in map) key = `new-host-${n++}`
  map[key] = ''
  emit('update:modelValue', map)
}
</script>

<template>
  <div class="space-y-1.5">
    <div v-for="(row, index) in rows" :key="index" class="flex items-start gap-1.5">
      <Input
        class="flex-1"
        :modelValue="row.domain"
        :placeholder="t('dns.servers.form.hostsDomainPlaceholder')"
        @update:modelValue="(v) => updateRow(index, { domain: String(v) })"
      />
      <Input
        class="flex-1"
        :modelValue="row.addresses"
        :placeholder="t('dns.servers.form.hostsAddressPlaceholder')"
        @update:modelValue="(v) => updateRow(index, { addresses: String(v) })"
      />
      <Button
        type="button"
        variant="ghost"
        size="sm"
        action
        class="text-red-600 dark:text-red-400 shrink-0"
        :title="t('common.delete')"
        @click="removeRow(index)"
      >
        <TrashIcon class="h-4 w-4" />
      </Button>
    </div>

    <button
      type="button"
      class="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-control border border-dashed border-gray-300 dark:border-gray-600 text-xs text-gray-500 dark:text-gray-400 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
      @click="addRow"
    >
      <PlusIcon class="h-3.5 w-3.5" />
      {{ t('dns.servers.form.addHost') }}
    </button>

    <p class="text-xs text-gray-500 dark:text-gray-400">
      {{ t('dns.servers.form.hostsHelp') }}
    </p>
  </div>
</template>
