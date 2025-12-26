<script setup lang="ts">
import { computed } from 'vue'
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
}

const props = withDefaults(defineProps<Props>(), {
  fullWidth: true,
  disabled: false,
  required: false,
  placeholder: 'Select an option',
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

// Find selected option object from value
const selectedOption = computed(() => {
  return props.options.find(o => o.value === props.modelValue) || null
})

// Handle selection change
const handleChange = (option: Option | null) => {
  if (option) {
    emit('update:modelValue', option.value)
  }
}
</script>

<template>
  <div :class="fullWidth ? 'w-full' : ''">
    <label v-if="label" class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
      {{ label }}
      <span v-if="required" class="text-red-500 ml-1">*</span>
    </label>

    <VueSelect
      :model-value="selectedOption"
      :options="options"
      :placeholder="placeholder"
      :disabled="disabled"
      label="label"
      :clearable="false"
      :searchable="false"
      @update:model-value="handleChange"
      :class="[
        'vue-select-wrapper',
        'z-50',
        error ? 'has-error' : '',
        disabled ? 'is-disabled' : '',
      ]"
    >
      <template #open-indicator="{ attributes }">
        <svg v-bind="attributes" class="h-5 w-5 text-gray-400 dark:text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4" />
        </svg>
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
</style>
