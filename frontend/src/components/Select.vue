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
    <label v-if="label" class="block text-sm font-medium text-gray-700 mb-1">
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
        <svg v-bind="attributes" class="h-5 w-5 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l4-4 4 4m0 6l-4 4-4-4" />
        </svg>
      </template>
    </VueSelect>

    <p v-if="error" class="mt-1 text-sm text-red-600">{{ error }}</p>
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

:deep(.vue-select-wrapper.has-error) {
  --vs-border-color: #fca5a5;
}

:deep(.vue-select-wrapper.has-error .vs__dropdown-toggle) {
  border-color: #fca5a5;
}

:deep(.vue-select-wrapper .vs__dropdown-toggle) {
  border: 1px solid var(--vs-border-color);
  padding: 0.375rem 0.75rem;
  transition: all 0.15s ease-in-out;
}

:deep(.vue-select-wrapper .vs__dropdown-toggle:hover:not(.vs--disabled)) {
  border-color: #9ca3af;
}

:deep(.vue-select-wrapper .vs__dropdown-toggle:focus-within) {
  border-color: #3b82f6;
  outline: none;
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

:deep(.vue-select-wrapper.has-error .vs__dropdown-toggle:focus-within) {
  border-color: #ef4444;
  box-shadow: 0 0 0 3px rgba(239, 68, 68, 0.1);
}

:deep(.vue-select-wrapper .vs__selected) {
  margin: 0;
  padding: 0;
  color: #111827;
}

:deep(.vue-select-wrapper .vs__search::placeholder) {
  color: #9ca3af;
}

:deep(.vue-select-wrapper .vs__dropdown-menu) {
  border: 1px solid #e5e7eb;
  box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1), 0 4px 6px -2px rgba(0, 0, 0, 0.05);
}

:deep(.vue-select-wrapper .vs__dropdown-option) {
  padding: 0.5rem 0.75rem;
}

:deep(.vue-select-wrapper .vs__dropdown-option--disabled) {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
