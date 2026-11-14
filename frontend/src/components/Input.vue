<script setup lang="ts">
import { computed, useId } from 'vue'
import FieldHelp from './FieldHelp.vue'
import { useInterfaceMotion } from '../composables/useInterfaceMotion'

const interfaceMotion = useInterfaceMotion()

interface Props {
  modelValue?: string | number
  type?: 'text' | 'password' | 'email' | 'number' | 'url'
  placeholder?: string
  disabled?: boolean
  error?: string
  /**
   * Optional explanation, shown on demand next to the label.
   * Use hintDisplay="inline" for concise examples that should remain visible.
   */
  hint?: string
  hintDisplay?: 'tooltip' | 'inline'
  label?: string
  required?: boolean
  fullWidth?: boolean
  fieldSize?: 'normal' | 'compact' | 'wide'
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
  hintDisplay: 'tooltip',
})

const emit = defineEmits<{
  'update:modelValue': [value: string | number]
}>()

// Ties <label>, the error/hint text, and the control together so the field is
// announced correctly and clicking the label focuses the input.
const inputId = useId()
const fieldSize = computed(() => props.fieldSize ?? (
  props.type === 'number' || /^\d+(ms|s|m|h|d)$/.test(props.placeholder ?? '') ? 'compact'
    : props.type === 'url' || (props.placeholder ?? '').includes('://') ? 'wide' : 'normal'
))
const describedById = computed(() =>
  props.error || (props.hint && props.hintDisplay === 'inline') ? `${inputId}-desc` : undefined,
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
  <div class="form-field" :class="[`field-${fieldSize}`, fullWidth ? 'w-full' : '', interfaceMotion ? 'ui-field relative' : '']">
    <div v-if="label || hint" class="mb-1 flex items-center gap-1">
      <label v-if="label" :for="inputId" class="text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ label }}<span v-if="required" class="ml-1 text-red-500" aria-hidden="true">*</span>
      </label>
      <FieldHelp v-if="hint && hintDisplay === 'tooltip'" :label="label || placeholder || ''" :text="hint" />
    </div>
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
    <Transition name="ui-field" :css="interfaceMotion">
    <p v-if="error" key="error" :id="describedById" role="alert" class="mt-1.5 text-sm text-red-600 dark:text-red-400">
      {{ error }}
    </p>
    <p
      v-else-if="hint && hintDisplay === 'inline'"
      key="hint"
      :id="describedById"
      class="mt-1.5 text-xs text-gray-500 dark:text-gray-400"
    >
      {{ hint }}
    </p>
    </Transition>
  </div>
</template>
