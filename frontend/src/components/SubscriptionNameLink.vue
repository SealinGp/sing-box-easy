<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/outline'
import Tooltip from './Tooltip.vue'
import TruncatedText from './TruncatedText.vue'

defineProps<{ name: string; href: string | null }>()
const { t } = useI18n()
</script>

<template>
  <Tooltip v-if="href" class="min-w-0 max-w-full">
    <template #content>{{ t('subscriptions.tooltip.openSite', { name }) }}</template>
    <template #default="{ describedBy }">
      <a
        :href="href"
        target="_blank"
        rel="noopener noreferrer"
        :aria-describedby="describedBy"
        class="group inline-flex max-w-full min-w-0 items-center gap-1 rounded-sm text-sm font-medium text-primary-600 transition-colors hover:text-primary-700 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-primary-400 dark:hover:text-primary-300"
      >
        <span class="truncate">{{ name }}</span>
        <ArrowTopRightOnSquareIcon aria-hidden="true" class="h-3 w-3 shrink-0 opacity-60 group-hover:opacity-100" />
      </a>
    </template>
  </Tooltip>
  <TruncatedText v-else :text="name" />
</template>
