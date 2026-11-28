<script setup lang="ts">
import IconButton from './IconButton.vue'

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
import { useI18n } from 'vue-i18n'
import { PlusIcon } from '@heroicons/vue/24/outline'
import Input from './Input.vue'
import { useHostsDraft, type HostsMap } from '../composables/useHostsDraft'

const props = defineProps<{ modelValue?: HostsMap }>()
const emit = defineEmits<{
  'update:modelValue': [value: HostsMap | undefined]
  validity: [valid: boolean]
}>()
const { t } = useI18n()
const { rows, errors, edit, add, remove } = useHostsDraft(
  () => props.modelValue,
  value => emit('update:modelValue', value),
  valid => emit('validity', valid),
)
</script>

<template>
  <div class="space-y-1.5">
    <div v-for="(row, index) in rows" :key="row.id" class="flex items-start gap-1.5">
      <Input
        class="flex-1"
        :modelValue="row.domain"
        :placeholder="t('dns.servers.form.hostsDomainPlaceholder')"
        :error="errors[index] ? t(`dns.servers.form.${errors[index]}`) : undefined"
        :aria-label="t('dns.servers.form.hostsDomainLabel')"
        @update:modelValue="(v) => edit(row.id, { domain: String(v) })"
      />
      <Input
        class="flex-1"
        :modelValue="row.addresses"
        :placeholder="t('dns.servers.form.hostsAddressPlaceholder')"
        :aria-label="t('dns.servers.form.hostsAddressLabel')"
        @update:modelValue="(v) => edit(row.id, { addresses: String(v) })"
      />
      <IconButton
        action="delete"
        type="button"
        @click="remove(row.id)"
        :label="t('common.delete')"
      />
    </div>

    <button
      type="button"
      class="w-full flex items-center justify-center gap-1.5 px-3 py-1.5 rounded-control border border-dashed border-gray-300 dark:border-gray-600 text-xs text-gray-500 dark:text-gray-400 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-300 transition-colors"
      @click="add"
    >
      <PlusIcon class="h-3.5 w-3.5" />
      {{ t('dns.servers.form.addHost') }}
    </button>

    <p class="text-xs text-gray-500 dark:text-gray-400">
      {{ t('dns.servers.form.hostsHelp') }}
    </p>
  </div>
</template>
