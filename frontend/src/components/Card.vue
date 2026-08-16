<script setup lang="ts">
/*
 * Padding scale shifted one step down for the compact density pass:
 *   sm 16→12px · md 24→16px · lg 32→24px
 *
 * `md` is the default and covers 41 of the 58 call sites, so this single map
 * is most of the card-level saving. Values are kept as Tailwind utilities
 * rather than `var(--space-card)` because the class list is built
 * conditionally here, and a token would only be readable in one of the four
 * branches.
 */
interface Props {
  title?: string
  hoverable?: boolean
  padding?: 'none' | 'sm' | 'md' | 'lg'
}

withDefaults(defineProps<Props>(), {
  hoverable: false,
  padding: 'md',
})
</script>

<template>
  <div
    :class="[
      'liquid-glass rounded-surface transition-shadow duration-200',
      hoverable ? 'hover:shadow-surface cursor-pointer' : 'shadow-surface',
      {
        'p-0': padding === 'none',
        'p-3': padding === 'sm',
        'p-4': padding === 'md',
        'p-6': padding === 'lg',
      },
    ]"
  >
    <h3 v-if="title" class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-3">
      {{ title }}
    </h3>
    <slot />
  </div>
</template>
