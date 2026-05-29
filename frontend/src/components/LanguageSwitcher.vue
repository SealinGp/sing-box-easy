<script setup lang="ts">
import { LanguageIcon } from '@heroicons/vue/24/outline'
import { useLocale } from '../i18n/useLocale'

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
    class="inline-flex items-center gap-1 px-2 py-1 rounded-lg text-xs font-medium text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
    :title="$t('common.language')"
    :aria-label="$t('common.language')"
  >
    <LanguageIcon class="h-4 w-4" />
    <span>{{ shortLabel }}</span>
  </button>

  <!-- Full: a labeled select, for the Settings page and the wizard. -->
  <select
    v-else
    v-model="locale"
    class="w-40 rounded-lg border border-gray-300 dark:border-gray-600 bg-white dark:bg-gray-900 px-3 py-2 text-gray-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-violet-500"
    :aria-label="$t('common.language')"
  >
    <option v-for="l in availableLocales" :key="l.code" :value="l.code">
      {{ l.label }}
    </option>
  </select>
</template>
