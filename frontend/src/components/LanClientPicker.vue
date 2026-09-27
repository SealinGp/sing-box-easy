<script setup lang="ts">
/**
 * Picks a route rule's `source_mac_address` or `source_hostname` from the
 * devices the router actually sees (GET /system/lan-clients: DHCP leases,
 * static hosts, neighbour table).
 *
 * Picking rather than typing for the same reason the outbound picker exists: a
 * mistyped MAC is not an error anywhere — sing-box accepts it and the rule
 * silently matches nothing.
 *
 * Typing stays possible, below the list: a device that is off right now has no
 * lease and no neighbour entry, and it is still a reasonable thing to write a
 * rule for. Off OpenWrt (or when the call fails) the typed entry is all there
 * is — the same chips the field used before this picker existed.
 */
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import ChipsField from './ChipsField.vue'
import Input from './Input.vue'
import { MultiSelect } from '../volt'
import { useLanClients } from '../composables/useLanClients'
import {
  buildLanClientOptions,
  normalizeMacInput,
  type LanClientPickMode,
} from '../utils/lanClientOptions'

const props = defineProps<{
  mode: LanClientPickMode
  value: unknown
  placeholder?: string
  disabled?: boolean
}>()

const emit = defineEmits<{ change: [value: string[] | undefined] }>()

const { t } = useI18n()
const { clients, supported, loading, refresh } = useLanClients()

/**
 * The rule's current list. In MAC mode each entry is shown canonical
 * (lower-case, colons) so `00:19:0F:…` in config.json selects the discovered
 * `00:19:0f:…` instead of appearing twice.
 */
const held = computed<string[]>(() => {
  const list = Array.isArray(props.value) ? props.value.map(String) : []
  return props.mode === 'mac' ? list.map((v) => normalizeMacInput(v) ?? v) : list
})

const options = computed(() => buildLanClientOptions(clients.value, props.mode, held.value))

function emitList(next: string[]) {
  emit('change', next.length ? [...new Set(next)] : undefined)
}

/* ── Manual entry ─────────────────────────────────────────────────────── */
const draft = ref('')
const draftError = ref('')

function addDraft() {
  const raw = draft.value.trim()
  if (!raw) return
  const value = props.mode === 'mac' ? normalizeMacInput(raw) : raw
  if (!value) {
    draftError.value = t('route.rules.lanPicker.invalidMac')
    return
  }
  draftError.value = ''
  draft.value = ''
  emitList([...held.value, value])
}
</script>

<template>
  <!-- Off OpenWrt, or the lookup failed: plain chips, exactly as before. -->
  <ChipsField
    v-if="supported === false"
    :modelValue="held"
    :placeholder="placeholder"
    :disabled="disabled"
    @update:modelValue="emitList"
  />

  <div v-else class="space-y-2">
    <div class="flex items-center gap-2">
      <MultiSelect
        class="w-full min-w-0"
        :modelValue="held"
        :options="options"
        optionLabel="label"
        optionValue="value"
        display="chip"
        filter
        :loading="loading"
        :disabled="disabled"
        :filterPlaceholder="t('common.search')"
        :emptyFilterMessage="t('common.noMatch')"
        :emptyMessage="t('route.rules.lanPicker.empty')"
        :placeholder="t(mode === 'mac' ? 'route.rules.lanPicker.pickMac' : 'route.rules.lanPicker.pickHostname')"
        @update:modelValue="(v: unknown) => emitList(Array.isArray(v) ? (v as string[]) : [])"
      >
        <template #option="{ option }">
          <span class="flex items-center gap-2 min-w-0">
            <span
              class="h-2 w-2 shrink-0 rounded-full"
              :class="option.online ? 'bg-emerald-500' : 'bg-gray-300 dark:bg-gray-600'"
              :title="t(option.online ? 'route.rules.lanPicker.online' : 'route.rules.lanPicker.offline')"
            />
            <span class="truncate">{{ option.label }}</span>
            <span v-if="option.missing" class="shrink-0 text-xs text-amber-600 dark:text-amber-400">
              {{ t('route.rules.lanPicker.notSeen') }}
            </span>
          </span>
        </template>
      </MultiSelect>
      <button
        type="button"
        class="shrink-0 p-2 rounded-control text-gray-500 hover:text-gray-700 hover:bg-gray-100 dark:hover:bg-gray-700 dark:hover:text-gray-200 transition-colors disabled:opacity-50"
        :title="t('route.rules.lanPicker.refresh')"
        :aria-label="t('route.rules.lanPicker.refresh')"
        :disabled="loading || disabled"
        @click="refresh(true)"
      >
        <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': loading }" />
      </button>
    </div>

    <div class="flex items-center gap-2">
      <Input
        :modelValue="draft"
        @update:modelValue="(v: string | number) => (draft = String(v))"
        class="flex-1"
        :placeholder="placeholder ?? t('route.rules.lanPicker.manualPlaceholder')"
        :disabled="disabled"
        @keydown.enter.prevent="addDraft"
      />
      <button
        type="button"
        class="shrink-0 px-3 py-1.5 text-sm rounded-control border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 transition-colors disabled:opacity-50"
        :disabled="disabled || !draft.trim()"
        @click="addDraft"
      >
        {{ t('common.add') }}
      </button>
    </div>
    <p v-if="draftError" class="text-xs text-red-600 dark:text-red-400">{{ draftError }}</p>
  </div>
</template>
