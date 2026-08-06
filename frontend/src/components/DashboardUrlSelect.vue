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
import {
  Listbox,
  ListboxButton,
  ListboxOption,
  ListboxOptions,
} from '@headlessui/vue'
import { CheckIcon, ChevronUpDownIcon, PencilSquareIcon } from '@heroicons/vue/24/outline'
import {
  CUSTOM_DASHBOARD_ID,
  DASHBOARD_OPTIONS,
  dashboardIdForUrl,
} from '../constants/dashboards'
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
    <Listbox
      :model-value="selectedId"
      :disabled="disabled"
      @update:model-value="selectOption"
    >
      <div class="relative">
        <ListboxButton
          class="dashboard-select-toggle relative w-full cursor-pointer rounded-full py-2 pl-3 pr-10 text-left text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-60"
        >
          <span v-if="selectedPreset" class="flex items-center gap-2">
            <img
              :src="selectedPreset.icon"
              :alt="selectedPreset.name"
              class="h-5 w-5 flex-shrink-0 rounded"
              @error="(e) => ((e.target as HTMLImageElement).style.visibility = 'hidden')"
            />
            <span class="truncate text-gray-900 dark:text-gray-100">{{ selectedPreset.name }}</span>
          </span>
          <span v-else-if="isCustom" class="flex items-center gap-2">
            <PencilSquareIcon class="h-5 w-5 flex-shrink-0 text-gray-500 dark:text-gray-400" />
            <span class="truncate text-gray-900 dark:text-gray-100">
              {{ customPlaceholder ?? $t('experimental.clash.dashboard.custom') }}
            </span>
          </span>
          <span v-else class="block truncate text-gray-400 dark:text-gray-500">
            {{ placeholder ?? $t('experimental.clash.dashboard.placeholder') }}
          </span>

          <span class="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3">
            <ChevronUpDownIcon class="h-5 w-5 text-gray-400 dark:text-gray-500" />
          </span>
        </ListboxButton>

        <Transition
          leave-active-class="transition duration-100 ease-in"
          leave-from-class="opacity-100"
          leave-to-class="opacity-0"
        >
          <ListboxOptions
            class="dashboard-select-panel absolute z-20 mt-1.5 max-h-72 w-full overflow-auto rounded-2xl p-1.5 text-sm focus:outline-none"
          >
            <ListboxOption
              v-for="dashboard in DASHBOARD_OPTIONS"
              v-slot="{ active, selected }"
              :key="dashboard.id"
              :value="dashboard.id"
              as="template"
            >
              <li
                :class="[
                  'relative flex cursor-pointer select-none items-center gap-2.5 rounded-xl py-2 pl-3 pr-9',
                  active ? 'bg-violet-500/12 text-violet-900 dark:text-violet-100' : 'text-gray-700 dark:text-gray-200',
                ]"
              >
                <img
                  :src="dashboard.icon"
                  :alt="dashboard.name"
                  class="h-6 w-6 flex-shrink-0 rounded"
                  @error="(e) => ((e.target as HTMLImageElement).style.visibility = 'hidden')"
                />
                <span class="min-w-0 flex-1">
                  <span :class="['block truncate', selected ? 'font-semibold' : 'font-normal']">
                    {{ dashboard.name }}
                  </span>
                  <span class="block truncate text-xs text-gray-500 dark:text-gray-400">
                    {{ $t(dashboard.descKey) }}
                  </span>
                </span>
                <CheckIcon
                  v-if="selected"
                  class="absolute right-3 h-4 w-4 text-violet-600 dark:text-violet-400"
                />
              </li>
            </ListboxOption>

            <!-- Custom URL escape hatch -->
            <ListboxOption
              v-slot="{ active, selected }"
              :value="CUSTOM_DASHBOARD_ID"
              as="template"
            >
              <li
                :class="[
                  'relative flex cursor-pointer select-none items-center gap-2.5 rounded-xl py-2 pl-3 pr-9',
                  active ? 'bg-violet-500/12 text-violet-900 dark:text-violet-100' : 'text-gray-700 dark:text-gray-200',
                ]"
              >
                <PencilSquareIcon class="h-6 w-6 flex-shrink-0 p-0.5 text-gray-500 dark:text-gray-400" />
                <span class="min-w-0 flex-1">
                  <span :class="['block truncate', selected ? 'font-semibold' : 'font-normal']">
                    {{ $t('experimental.clash.dashboard.custom') }}
                  </span>
                  <span class="block truncate text-xs text-gray-500 dark:text-gray-400">
                    {{ $t('experimental.clash.dashboard.customHelp') }}
                  </span>
                </span>
                <CheckIcon
                  v-if="selected"
                  class="absolute right-3 h-4 w-4 text-violet-600 dark:text-violet-400"
                />
              </li>
            </ListboxOption>
          </ListboxOptions>
        </Transition>
      </div>
    </Listbox>

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

<style scoped>
/* Mirrors the glass treatment used by Select.vue's toggle so the two controls
   read as the same family. */
.dashboard-select-toggle {
  border: 1px solid rgba(148, 163, 184, 0.28);
  background: var(--glass-bg-strong);
  box-shadow: var(--glass-highlight);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
}

.dashboard-select-toggle:hover:not(:disabled) {
  border-color: #9ca3af;
}

.dashboard-select-toggle:focus-visible {
  border-color: #3b82f6;
  outline: none;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.dashboard-select-panel {
  border: 1px solid var(--glass-border-muted);
  background: var(--glass-bg-strong);
  box-shadow: var(--shadow-md), var(--glass-highlight);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
}

@media (prefers-color-scheme: dark) {
  .dashboard-select-toggle {
    border-color: var(--glass-border-muted);
  }

  .dashboard-select-toggle:hover:not(:disabled) {
    border-color: #6b7280;
  }

  .dashboard-select-toggle:focus-visible {
    border-color: #60a5fa;
    box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.2);
  }
}
</style>
