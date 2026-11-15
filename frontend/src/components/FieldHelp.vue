<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
import Tooltip from './Tooltip.vue'
defineProps<{ label: string; text: string }>()
const { t } = useI18n()
</script>
<template>
  <Tooltip class="inline-flex shrink-0">
    <template #default="{ describedBy, show }">
      <button type="button" class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-text-secondary hover:text-primary focus-visible:outline-2 focus-visible:outline-primary"
        :aria-label="t('common.fieldHelp', { name: label })" :aria-describedby="describedBy" @click.stop="show" @mousedown.stop>
        <InformationCircleIcon class="h-4 w-4" aria-hidden="true" />
      </button>
    </template>
    <template #content>{{ text }}</template>
  </Tooltip>
</template>
