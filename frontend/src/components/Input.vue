<script setup lang="ts">
import { computed, useId } from 'vue'

interface Props {
  modelValue?: string | number
  type?: 'text' | 'password' | 'email' | 'number' | 'url'
  placeholder?: string
  disabled?: boolean
  error?: string
  /**
   * Helper text shown under the input when there is no error,
   * e.g. format examples for duration fields.
   */
  hint?: string
  label?: string
  required?: boolean
  fullWidth?: boolean
  /**
   * Numeric bounds, declared rather than left to fall through: this component's
   * root is the wrapper <div>, so an undeclared `min` would land on the wrapper
   * and silently do nothing.
   */
  min?: number | string
  max?: number | string
}

const props = withDefaults(defineProps<Props>(), {
  type: 'text',
  disabled: false,
  required: false,
  fullWidth: true,
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

// Ties <label>, the error/hint text, and the control together so the field is
// announced correctly and clicking the label focuses the input.
const inputId = useId()
const describedById = computed(() =>
  props.error || props.hint ? `${inputId}-desc` : undefined,
)

/*
 * Border, fill, focus ring, and radius all come from the shared control layer
 * in `src/style/controls.css`, keyed off `--control-*`. Deliberately NOT
 * restated here: this component used to carry `border-white/40` + `rounded-full`
 * inline, which rendered an invisible edge on a white card and disagreed with
 * every other control's radius.
 */
const inputClasses = computed(() =>
  // Compact density pass: was `px-3.5 py-2`. Must stay in step with `.select`
  // in style/controls.css and volt's `.volt-select`, or the three controls
  // stop being pixel-identical.
  ['block px-2.5 py-1.5 text-sm', props.fullWidth ? 'w-full' : ''].join(' '),
)
</script>

<template>
  <div :class="fullWidth ? 'w-full' : ''">
    <label
      v-if="label"
      :for="inputId"
      class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300"
    >
      {{ label }}
      <span v-if="required" class="ml-1 text-red-500" aria-hidden="true">*</span>
    </label>
    <input
      :id="inputId"
      :type="type"
      :min="min"
      :max="max"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :required="required"
      :aria-invalid="error ? 'true' : undefined"
      :aria-describedby="describedById"
      :class="inputClasses"
      @input="
        emit(
          'update:modelValue',
          type === 'number'
            ? Number(($event.target as HTMLInputElement).value)
            : ($event.target as HTMLInputElement).value,
        )
      "
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
