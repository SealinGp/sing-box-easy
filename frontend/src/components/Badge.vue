<script setup lang="ts">
import { computed } from 'vue'

export interface Props {
  variant?: 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'secondary' | 'gray'
  size?: 'sm' | 'md' | 'lg'
  dot?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'gray',
  size: 'md',
  dot: false,
})

const classes = computed(() => {
  const base = 'inline-flex items-center font-semibold rounded-full border'

  const variants = {
    primary: 'bg-blue-500/15 text-blue-700 dark:text-blue-200 border-blue-500/25',
    success: 'bg-emerald-500/15 text-emerald-700 dark:text-emerald-200 border-emerald-500/25',
    warning: 'bg-amber-500/15 text-amber-700 dark:text-amber-200 border-amber-500/25',
    danger: 'bg-red-500/15 text-red-700 dark:text-red-200 border-red-500/25',
    info: 'bg-sky-500/15 text-sky-700 dark:text-sky-200 border-sky-500/25',
    secondary: 'bg-white/35 dark:bg-white/10 text-gray-700 dark:text-gray-200 border-white/35 dark:border-white/10',
    gray: 'bg-white/35 dark:bg-white/10 text-gray-700 dark:text-gray-200 border-white/35 dark:border-white/10',
  }

  const sizes = {
    sm: 'px-2 py-0.5 text-xs',
    md: 'px-2.5 py-0.5 text-sm',
    lg: 'px-3 py-1 text-base',
  }

  return `${base} ${variants[props.variant]} ${sizes[props.size]}`
})

const dotColorClass = computed(() => {
  const colors = {
    primary: 'bg-blue-500',
    success: 'bg-emerald-500',
    warning: 'bg-amber-500',
    danger: 'bg-red-500',
    info: 'bg-sky-500',
    secondary: 'bg-gray-500',
    gray: 'bg-gray-500',
  }
  return colors[props.variant]
})
</script>

<template>
  <span :class="classes">
    <span v-if="dot" :class="['w-1.5 h-1.5 rounded-full mr-1.5', dotColorClass]"></span>
    <slot />
  </span>
</template>
