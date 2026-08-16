<script setup lang="ts">
/**
 * Raw-JSON editor for a sing-box option that has no dedicated control yet —
 * `tls`, `transport`, `multiplex`, `fallback`, and the object-valued fields a
 * future sing-box adds before anyone curates them.
 *
 * This exists so the schema-driven form can be honest about its own gaps. The
 * alternative was to omit uncurated object fields entirely, which is how the
 * old hand-written modal ended up unable to edit TLS on any inbound: the field
 * was simply not in the template, and nothing said so.
 *
 * Invalid JSON is kept in the textarea and reported, never pushed to the model.
 * Emitting a half-typed object on every keystroke would let a stray character
 * wipe a populated `tls` block.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  modelValue?: unknown
  placeholder?: string
  disabled?: boolean
  rows?: number
}>()

const emit = defineEmits<{
  'update:modelValue': [value: unknown]
}>()

const { t } = useI18n()

const text = ref('')
const error = ref('')

/** Serialize incoming model changes, unless the user is mid-edit with invalid text. */
watch(
  () => props.modelValue,
  (value) => {
    if (error.value) return
    text.value = value === undefined || value === null ? '' : JSON.stringify(value, null, 2)
  },
  { immediate: true },
)

function onInput(event: Event) {
  const raw = (event.target as HTMLTextAreaElement).value
  text.value = raw

  const trimmed = raw.trim()
  if (trimmed === '') {
    error.value = ''
    emit('update:modelValue', undefined)
    return
  }

  try {
    const parsed = JSON.parse(trimmed)
    error.value = ''
    emit('update:modelValue', parsed)
  } catch (err) {
    // Held locally: the model keeps its last valid value so a typo cannot
    // destroy an existing block.
    error.value = err instanceof Error ? err.message : String(err)
  }
}

const rowCount = computed(() => props.rows ?? 4)
</script>

<template>
  <div>
    <textarea
      :value="text"
      :rows="rowCount"
      :disabled="disabled"
      :placeholder="placeholder ?? '{ }'"
      spellcheck="false"
      class="w-full rounded-control border px-2 py-1.5 font-mono text-xs leading-relaxed"
      :class="
        error
          ? 'border-red-400 dark:border-red-500'
          : 'border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-700 text-gray-900 dark:text-gray-100'
      "
      @input="onInput"
    />
    <p v-if="error" class="mt-1 text-xs text-red-600 dark:text-red-400">
      {{ t('inbounds.form.invalidJson', { message: error }) }}
    </p>
  </div>
</template>
