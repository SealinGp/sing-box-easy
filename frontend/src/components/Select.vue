<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import VueSelect from 'vue-select'
import 'vue-select/dist/vue-select.css'

interface Option {
  label: string
  value: string | number
  disabled?: boolean
}

interface Props {
  modelValue?: string | number | null
  options: Option[]
  placeholder?: string
  label?: string
  error?: string
  required?: boolean
  disabled?: boolean
  fullWidth?: boolean
  searchable?: boolean
  loading?: boolean
  searchPlaceholder?: string
  noOptionsText?: string
  serverSideSearch?: boolean
  debounce?: number
  clearable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  fullWidth: true,
  disabled: false,
  required: false,
  placeholder: 'Select an option',
  searchable: false,
  loading: false,
  searchPlaceholder: 'Type to search...',
  noOptionsText: 'No matching options',
  serverSideSearch: false,
  debounce: 250,
  clearable: false,
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
  'search': [query: string, loading: (isLoading: boolean) => void]
}>()

// Local state for search
const searchQuery = ref('')
const isSearching = ref(false)
const filteredOptions = ref<Option[]>([])
const searchDebounceTimer = ref<any>()

// Find selected option object from value
const selectedOption = computed(() => {
  return props.options.find(o => o.value === props.modelValue) || null
})

// Handle selection change
const handleChange = (option: Option | null) => {
  if (option) {
    emit('update:modelValue', option.value)
  } else {
    emit('update:modelValue', '')
  }
}

// Options to display (filtered or original)
const displayOptions = computed(() => {
  // If not searchable or no search query, return all options
  if (!props.searchable || !searchQuery.value) {
    return props.options
  }

  // If server-side search, return filtered options from server
  if (props.serverSideSearch) {
    return filteredOptions.value
  }

  // Client-side filtering
  const query = searchQuery.value.toLowerCase()
  return props.options.filter(option =>
    option.label.toLowerCase().includes(query)
  )
})

// Handle search input
const handleSearch = (query: string, loading: (isLoading: boolean) => void) => {
  searchQuery.value = query

  // Clear previous debounce timer
  if (searchDebounceTimer.value) {
    clearTimeout(searchDebounceTimer.value)
  }

  if (!query) {
    filteredOptions.value = []
    return
  }

  if (props.serverSideSearch) {
    // Debounce server-side search
    loading(true)
    isSearching.value = true

    searchDebounceTimer.value = setTimeout(() => {
      // Emit search event for parent to handle
      emit('search', query, (isLoading: boolean) => {
        loading(isLoading)
        isSearching.value = isLoading
      })
    }, props.debounce)
  }
  // Client-side search is handled by computed property
}

// Watch for external options changes (for server-side search results)
watch(() => props.options, (newOptions) => {
  if (props.serverSideSearch && searchQuery.value) {
    filteredOptions.value = newOptions
  }
})

// Clean up on unmount
watch(() => props.searchable, (newVal) => {
  if (!newVal) {
    searchQuery.value = ''
    filteredOptions.value = []
  }
})
</script>

<template>
  <div :class="fullWidth ? 'w-full' : ''">
    <label v-if="label" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
      {{ label }}
      <span v-if="required" class="text-red-500 ml-1">*</span>
    </label>

    <VueSelect
      :model-value="selectedOption"
      :options="displayOptions"
      :placeholder="placeholder"
      :disabled="disabled"
      label="label"
      :clearable="clearable"
      :searchable="searchable"
      :loading="loading || isSearching"
      :filterable="false"
      @search="handleSearch"
      @update:model-value="handleChange"
      :class="[
        'vue-select-wrapper',
        error ? 'has-error' : '',
        disabled ? 'is-disabled' : '',
      ]"
    >
      <!-- Custom search input placeholder -->
      <template v-if="searchable" #search="{ attributes, events }">
        <input
          class="vs__search"
          :placeholder="searchPlaceholder"
          v-bind="attributes"
          v-on="events"
        />
      </template>

      <!-- No options message -->
      <template #no-options="{ search }">
        <div class="text-center py-2 text-gray-500 dark:text-gray-400">
          {{ search && searchable ? noOptionsText : 'No options available' }}
        </div>
      </template>

      <!-- Loading indicator -->
      <template v-if="searchable && serverSideSearch" #spinner="{ loading }">
        <div v-if="loading" class="vs__spinner-container">
          <svg class="vs__spinner animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
          </svg>
        </div>
      </template>

      <!-- Custom dropdown indicator -->
      <template #open-indicator="{ attributes }">
        <svg v-bind="attributes" class="h-5 w-5 text-gray-400 dark:text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4" />
        </svg>
      </template>

      <!-- Custom option template with disabled state -->
      <template #option="{ label: optionLabel, disabled: optionDisabled }">
        <div :class="['vs__option-content', optionDisabled ? 'opacity-50 cursor-not-allowed' : '']">
          {{ optionLabel }}
        </div>
      </template>
    </VueSelect>

    <p v-if="error" class="mt-1 text-sm text-red-600 dark:text-red-400">{{ error }}</p>
  </div>
</template>

<style scoped>
/* Override vue-select styles to match project theme */
:deep(.vue-select-wrapper) {
  --vs-border-color: #d1d5db;
  --vs-border-radius: 0.5rem;
  --vs-state-disabled-bg: #f9fafb;
  --vs-state-disabled-color: #6b7280;
  --vs-dropdown-option--active-bg: #dbeafe;
  --vs-dropdown-option--active-color: #1e40af;
}


/* Dark mode variables */
@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper) {
    --vs-border-color: #4b5563;
    --vs-state-disabled-bg: #1f2937;
    --vs-state-disabled-color: #9ca3af;
    --vs-dropdown-option--active-bg: #1e3a8a;
    --vs-dropdown-option--active-color: #93c5fd;
  }
}

:deep(.vue-select-wrapper.has-error) {
  --vs-border-color: #fca5a5;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper.has-error) {
    --vs-border-color: #f87171;
  }
}

:deep(.vue-select-wrapper.has-error .vs__dropdown-toggle) {
  border-color: #fca5a5;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper.has-error .vs__dropdown-toggle) {
    border-color: #f87171;
  }
}

:deep(.vue-select-wrapper .vs__dropdown-toggle) {
  border: 1px solid var(--vs-border-color);
  padding: 0.375rem 0.75rem;
  transition: all 0.15s ease-in-out;
  background-color: white;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__dropdown-toggle) {
    background-color: #1f2937 !important;
    border-color: #4b5563 !important;
  }
}

:deep(.vue-select-wrapper .vs__dropdown-toggle:hover:not(.vs--disabled)) {
  border-color: #9ca3af;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__dropdown-toggle:hover:not(.vs--disabled)) {
    border-color: #6b7280;
  }
}

:deep(.vue-select-wrapper .vs__dropdown-toggle:focus-within) {
  border-color: #3b82f6;
  outline: none;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__dropdown-toggle:focus-within) {
    border-color: #60a5fa;
    box-shadow: 0 0 0 3px rgba(96, 165, 250, 0.2);
  }
}

:deep(.vue-select-wrapper.has-error .vs__dropdown-toggle:focus-within) {
  border-color: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper.has-error .vs__dropdown-toggle:focus-within) {
    border-color: #f87171;
    box-shadow: 0 0 0 3px rgba(248, 113, 113, 0.2);
  }
}

:deep(.vue-select-wrapper .vs__selected) {
  margin: 0;
  padding: 0;
  color: #111827;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__selected) {
    color: #f9fafb !important;
  }
}

:deep(.vue-select-wrapper .vs__search) {
  color: #111827;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__search) {
    color: #f9fafb !important;
  }
}

:deep(.vue-select-wrapper .vs__search::placeholder) {
  color: #9ca3af;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__search::placeholder) {
    color: #6b7280 !important;
  }
}

:deep(.vue-select-wrapper .vs__dropdown-menu) {
  border: 1px solid #e5e7eb;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
  background-color: white;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__dropdown-menu) {
    background-color: #1f2937 !important;
    border-color: #4b5563 !important;
    box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.3), 0 4px 6px -2px rgba(0, 0, 0, 0.15);
  }
}

:deep(.vue-select-wrapper .vs__dropdown-option) {
  padding: 0.5rem 0.75rem;
  color: #374151;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__dropdown-option) {
    color: #e5e7eb !important;
  }
}

:deep(.vue-select-wrapper .vs__dropdown-option--highlight) {
  background-color: #eff6ff;
  color: #1e40af;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__dropdown-option--highlight) {
    background-color: #1e3a8a !important;
    color: #93c5fd !important;
  }
}

:deep(.vue-select-wrapper .vs__dropdown-option--selected) {
  background-color: #dbeafe;
  color: #1e40af;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__dropdown-option--selected) {
    background-color: #1e3a8a !important;
    color: #93c5fd !important;
  }
}

:deep(.vue-select-wrapper .vs__dropdown-option--disabled) {
  opacity: 0.5;
  cursor: not-allowed;
}

:deep(.vue-select-wrapper .vs__clear) {
  color: #6b7280;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__clear) {
    color: #9ca3af;
  }
}

:deep(.vue-select-wrapper .vs__spinner) {
  border-color: #d1d5db;
  border-top-color: #3b82f6;
}

@media (prefers-color-scheme: dark) {
  :deep(.vue-select-wrapper .vs__spinner) {
    border-color: #4b5563;
    border-top-color: #60a5fa;
  }
}

/* Search input styles when searchable */
:deep(.vue-select-wrapper.vs--searchable .vs__search) {
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
  color: inherit;
}

:deep(.vue-select-wrapper.vs--searchable .vs__search:focus) {
  outline: none;
}

/* Loading spinner container */
:deep(.vue-select-wrapper .vs__spinner-container) {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0.25rem;
}

/* Adjust dropdown when loading */
:deep(.vue-select-wrapper.vs--loading .vs__dropdown-menu) {
  min-height: 60px;
}

/* Custom option content */
:deep(.vue-select-wrapper .vs__option-content) {
  display: flex;
  align-items: center;
  width: 100%;
}
</style>
