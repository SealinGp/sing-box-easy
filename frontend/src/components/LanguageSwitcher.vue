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
    class="language-toggle inline-flex items-center gap-1 px-2 py-1 rounded-control text-xs font-medium text-text-secondary hover:bg-bg-tertiary transition-colors"
    :title="$t('common.language')"
    :aria-label="$t('common.language')"
  >
    <LanguageIcon class="h-4 w-4" />
    <span>{{ shortLabel }}</span>
  </button>

  <!-- Full: a labeled select, for the Settings page. -->
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
