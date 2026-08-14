<script setup lang="ts">
/**
 * Labelled multi-value field (domains, keywords, geosite codes, ...).
 *
 * Wraps the volt `Chips` control with the two things every call site was
 * repeating — or, worse, omitting: a label, and a hint that says how to commit
 * an entry. A bare chips box gives no clue that Enter is what turns typed text
 * into a chip, which reads as "the input is broken".
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chips } from '../volt'
import { XMarkIcon } from '@heroicons/vue/24/outline'

const props = defineProps<{
  // Optional because several call sites model an absent sing-box field as
  // `undefined` (see RouteRuleMatchers) — an absent list reads as empty.
  modelValue?: string[]
  label: string
  placeholder?: string
  /** Field-specific explanation, shown before the shared "how to add" hint. */
  hint?: string
  disabled?: boolean
  /**
   * Show a "remove" action next to the label. Call sites offer this only for a
   * field the operator opted into and has not filled in — removing a field that
   * holds values would throw those values away silently.
   */
  removable?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string[]]
  remove: []
}>()

const { t } = useI18n()

const value = computed({
  get: () => props.modelValue ?? [],
  set: (next: string[]) => emit('update:modelValue', next),
})

// The shared hint always appears; a field-specific one is prefixed to it so the
// two never fight over the same line.
const hintText = computed(() =>
  props.hint ? `${props.hint} ${t('common.chipsHint')}` : t('common.chipsHint'),
)
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-1">
      <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
        {{ label }}
      </label>
      <button
        v-if="removable"
        type="button"
        class="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
        :title="$t('common.remove')"
        @click="emit('remove')"
      >
        <XMarkIcon class="h-4 w-4" />
      </button>
    </div>
    <Chips
      v-model="value"
      :placeholder="placeholder"
      :disabled="disabled"
      class="w-full"
    />
    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ hintText }}</p>
  </div>
</template>
