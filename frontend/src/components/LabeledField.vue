<script setup lang="ts">
/**
 * Label + optional hint + optional "remove this field" action, wrapped around
 * whatever control the caller slots in.
 *
 * Exists because the condition forms mix control types — chips for free text,
 * MultiSelect for fixed vocabularies — and every one of them needs the same
 * header row. Without this the remove affordance would be reimplemented per
 * control, and the two would drift.
 */
defineProps<{
  /** Omitted when the surrounding block already names the field. */
  label?: string
  /** Explanation rendered under the control. */
  hint?: string
  /**
   * Show a "remove" action next to the label. Call sites offer this only for a
   * field the operator opted into and has not filled in — removing a field that
   * holds values would throw those values away silently.
   */
  removable?: boolean
}>()

const emit = defineEmits<{ remove: [] }>()
</script>

<template>
  <div>
    <div v-if="label || removable" class="flex items-center justify-between mb-1">
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
        <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" d="M6 18 18 6M6 6l12 12" />
        </svg>
      </button>
    </div>

    <slot />

    <p v-if="hint" class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ hint }}</p>
  </div>
</template>
