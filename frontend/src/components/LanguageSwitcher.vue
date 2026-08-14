<script setup lang="ts">
import { LanguageIcon } from '@heroicons/vue/24/outline'
import { useLocale } from '../i18n/useLocale'
import { Select } from '../volt'

withDefaults(defineProps<{ variant?: 'compact' | 'full' }>(), {
  variant: 'compact',
})

const { locale, toggle, shortLabel, availableLocales } = useLocale()
</script>

<template>
  <!-- Compact: a single toggle button cycling between the two locales. -->
  <button
    v-if="variant === 'compact'"
    @click="toggle"
    class="inline-flex items-center gap-1 px-2 py-1 rounded-control text-xs font-medium text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
    :title="$t('common.language')"
    :aria-label="$t('common.language')"
  >
    <LanguageIcon class="h-4 w-4" />
    <span>{{ shortLabel }}</span>
  </button>

  <!-- Full: a labeled select, for the Settings page and the wizard. -->
  <Select
    v-else
    class="w-40"
    v-model="locale"
    :options="availableLocales"
    optionLabel="label"
    optionValue="code"
    :aria-label="$t('common.language')"
  />
</template>
