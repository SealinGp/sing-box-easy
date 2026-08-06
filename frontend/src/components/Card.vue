<script setup lang="ts">
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
        'p-4': padding === 'sm',
        'p-6': padding === 'md',
        'p-8': padding === 'lg',
      },
    ]"
  >
    <h3 v-if="title" class="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">
      {{ title }}
    </h3>
    <slot />
  </div>
</template>
