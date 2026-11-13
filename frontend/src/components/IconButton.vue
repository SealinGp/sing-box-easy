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
  <button :type="type" class="icon-action ui-control" :class="{ 'icon-action-danger': action === 'delete' }"
    :disabled="disabled || loading" :aria-busy="loading || undefined"
    :aria-label="label || t(`common.${action}`)" :title="label || t(`common.${action}`)">
    <span v-if="loading" class="h-4 w-4 animate-spin rounded-full border-[1.5px] border-current border-t-transparent" aria-hidden="true" />
    <ActionIcon v-else :action="action" />
  </button>
</template>
