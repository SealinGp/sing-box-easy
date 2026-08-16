<script setup lang="ts">
import { computed, useId } from 'vue'

interface Props {
  modelValue?: string
  placeholder?: string
  disabled?: boolean
  error?: string
  hint?: string
  label?: string
  required?: boolean
  fullWidth?: boolean
  rows?: number
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  required: false,
  fullWidth: true,
  rows: 4,
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const textareaId = useId()
const describedById = computed(() =>
  props.error || props.hint ? `${textareaId}-desc` : undefined,
)

// Surface styling (fill, border, focus ring, radius) is inherited from
// `src/style/controls.css` — see the note in Input.vue.
const textareaClasses = computed(() =>
  // Compact density pass: was `px-3.5 py-2.5`.
  ['block px-2.5 py-1.5 text-sm resize-y', props.fullWidth ? 'w-full' : ''].join(' '),
)
</script>

<template>
  <div :class="fullWidth ? 'w-full' : ''">
    <label
      v-if="label"
      :for="textareaId"
      class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300"
    >
      {{ label }}
      <span v-if="required" class="ml-1 text-red-500" aria-hidden="true">*</span>
    </label>
    <textarea
      :id="textareaId"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :required="required"
      :rows="rows"
      :aria-invalid="error ? 'true' : undefined"
      :aria-describedby="describedById"
      :class="textareaClasses"
      @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
    />
    <p v-if="error" :id="describedById" class="mt-1.5 text-sm text-red-600 dark:text-red-400">
      {{ error }}
    </p>
    <p
      v-else-if="hint"
      :id="describedById"
      class="mt-1.5 text-xs text-gray-500 dark:text-gray-400"
    >
      {{ hint }}
    </p>
  </div>
</template>
