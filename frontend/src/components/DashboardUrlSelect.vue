<script setup lang="ts">
/**
 * Dashboard download-URL picker.
 *
 * Presents the known dashboards (with their icons, so the user can recognise
 * the UI they are choosing) plus a "custom URL" escape hatch. The bound value
 * is always the download URL itself — the preset id is internal UI state — so
 * callers keep storing exactly what sing-box expects in
 * `external_ui_download_url`.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { PencilSquareIcon } from '@heroicons/vue/24/outline'
import {
  CUSTOM_DASHBOARD_ID,
  DASHBOARD_OPTIONS,
  dashboardIdForUrl,
} from '../constants/dashboards'
import { Select } from '../volt'
import Input from './Input.vue'

interface Props {
  modelValue?: string
  disabled?: boolean
  customPlaceholder?: string
  placeholder?: string
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  disabled: false,
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const { t } = useI18n()

/**
 * One row of the dropdown. The presets and the "custom URL" escape hatch share
 * this shape so both render through the same `#option` slot; `icon: null` is
 * what marks the escape hatch (it draws a pencil glyph instead of an image).
 */
interface DashboardRow {
  id: string
  name: string
  descKey: string
  icon: string | null
}

/** Which row is highlighted. Local UI state, derived from the bound URL. */
const selectedId = ref<string>(dashboardIdForUrl(props.modelValue) ?? '')

/** Draft custom URL, kept so switching presets and back does not lose it. */
const customUrl = ref(
  dashboardIdForUrl(props.modelValue) === CUSTOM_DASHBOARD_ID ? props.modelValue : '',
)

// Re-derive from the parent whenever it swaps the value in (e.g. after the
// config loads). An empty URL must NOT clear an explicit "custom" choice —
// the user has picked custom and simply hasn't typed the URL yet.
watch(
  () => props.modelValue,
  (url) => {
    const id = dashboardIdForUrl(url)
    if (id === null) {
      if (selectedId.value !== CUSTOM_DASHBOARD_ID) selectedId.value = ''
      return
    }
    selectedId.value = id
    if (id === CUSTOM_DASHBOARD_ID) customUrl.value = url ?? ''
  },
)

watch(customUrl, (value) => {
  if (selectedId.value === CUSTOM_DASHBOARD_ID) {
    emit('update:modelValue', value.trim())
  }
})

const isCustom = computed(() => selectedId.value === CUSTOM_DASHBOARD_ID)

const selectedPreset = computed(
  () => DASHBOARD_OPTIONS.find((d) => d.id === selectedId.value) ?? null,
)

/** Presets + the synthetic "custom URL" row, in dropdown order. */
const selectOptions = computed<DashboardRow[]>(() => [
  ...DASHBOARD_OPTIONS.map(({ id, name, descKey, icon }) => ({ id, name, descKey, icon })),
  {
    id: CUSTOM_DASHBOARD_ID,
    name: t('experimental.clash.dashboard.custom'),
    descKey: 'experimental.clash.dashboard.customHelp',
    icon: null,
  },
])

/**
 * The `#value` slot hands back the raw bound value (an id, because
 * `option-value="id"`), so the row has to be looked back up to render it.
 */
const optionById = (id: unknown): DashboardRow | null =>
  selectOptions.value.find((row) => row.id === id) ?? null

/** Hide a broken dashboard icon rather than showing the browser's fallback. */
const hideBrokenIcon = (event: Event) => {
  ;(event.target as HTMLImageElement).style.visibility = 'hidden'
}

const selectOption = (id: string) => {
  selectedId.value = id

  if (id === CUSTOM_DASHBOARD_ID) {
    // Switching to custom surfaces the draft URL rather than wiping the field.
    emit('update:modelValue', customUrl.value.trim())
    return
  }

  const preset = DASHBOARD_OPTIONS.find((d) => d.id === id)
  if (preset) emit('update:modelValue', preset.url)
}
</script>

<template>
  <div class="space-y-2">
    <Select
      :model-value="selectedId"
      :options="selectOptions"
      option-label="name"
      option-value="id"
      :disabled="disabled"
      :placeholder="placeholder ?? $t('experimental.clash.dashboard.placeholder')"
      class="w-full"
      @update:model-value="selectOption"
    >
      <!-- Closed trigger: icon + name of whatever id is bound right now. -->
      <template #value="{ value }">
        <span v-if="optionById(value)" class="flex items-center gap-2">
          <img
            v-if="optionById(value)!.icon"
            :src="optionById(value)!.icon!"
            :alt="optionById(value)!.name"
            class="h-5 w-5 flex-shrink-0 rounded"
            @error="hideBrokenIcon"
          />
          <PencilSquareIcon
            v-else
            class="h-5 w-5 flex-shrink-0 text-gray-500 dark:text-gray-400"
          />
          <span class="truncate text-gray-900 dark:text-gray-100">
            {{
              optionById(value)!.icon
                ? optionById(value)!.name
                : (customPlaceholder ?? $t('experimental.clash.dashboard.custom'))
            }}
          </span>
        </span>
        <span v-else class="block truncate text-gray-400 dark:text-gray-500">
          {{ placeholder ?? $t('experimental.clash.dashboard.placeholder') }}
        </span>
      </template>

      <!-- Every row — presets and the custom URL escape hatch alike. -->
      <template #option="{ option }">
        <img
          v-if="option.icon"
          :src="option.icon"
          :alt="option.name"
          class="h-6 w-6 flex-shrink-0 rounded"
          @error="hideBrokenIcon"
        />
        <PencilSquareIcon
          v-else
          class="h-6 w-6 flex-shrink-0 p-0.5 text-gray-500 dark:text-gray-400"
        />
        <span class="min-w-0 flex-1">
          <span class="block truncate">{{ option.name }}</span>
          <span class="block truncate text-xs font-normal text-gray-500 dark:text-gray-400">
            {{ $t(option.descKey) }}
          </span>
        </span>
      </template>
    </Select>

    <!-- Free-form URL entry, only when "custom" is chosen -->
    <Input
      v-if="isCustom"
      v-model="customUrl"
      :disabled="disabled"
      :placeholder="$t('experimental.clash.dashboard.customPlaceholder')"
    />

    <!-- Resolved URL, so the user can see exactly what will be downloaded -->
    <p
      v-else-if="selectedPreset"
      class="break-all px-3 font-mono text-xs text-gray-500 dark:text-gray-400"
    >
      {{ selectedPreset.url }}
    </p>
  </div>
</template>

