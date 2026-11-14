<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ActionIcon from './ActionIcon.vue'
withDefaults(defineProps<{
  action: 'edit' | 'delete'
  label?: string
  loading?: boolean
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
}>(), { type: 'button' })
const { t } = useI18n()
</script>

<template>
  <button :type="type" class="icon-action semantic-action ui-control" :class="{ 'icon-action-danger': action === 'delete' }"
    :disabled="disabled || loading" :aria-busy="loading || undefined"
    :aria-label="label || t(`common.${action}`)" :title="label || t(`common.${action}`)">
    <span v-if="loading" class="h-4 w-4 animate-spin rounded-full border-[1.5px] border-current border-t-transparent" aria-hidden="true" />
    <ActionIcon v-else :action="action" />
    <span class="semantic-action-label" aria-hidden="true">{{ label || t(`common.${action}`) }}</span>
  </button>
</template>

<style>
.semantic-action-label { display: none; }

/* Adjacent text actions (such as Cancel) need equally explicit companions.
   Row actions remain icon-only. Sibling selectors work in teleported panels too. */
.semantic-action:has(+ [data-text-button]),
[data-text-button] + .semantic-action {
  width: auto;
  min-width: 28px;
  min-height: 32px;
  height: auto;
  padding: 4px 10px;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
}
.semantic-action:has(+ [data-text-button]) .semantic-action-label,
[data-text-button] + .semantic-action .semantic-action-label {
  display: inline;
}
</style>
