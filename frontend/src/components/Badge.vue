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
  const base = 'inline-flex items-center font-medium rounded-full'

  const variants = {
    primary: 'bg-blue-100 text-blue-800',
    success: 'bg-green-100 text-green-800',
    warning: 'bg-amber-100 text-amber-800',
    danger: 'bg-red-100 text-red-800',
    info: 'bg-purple-100 text-purple-800',
    secondary: 'bg-gray-100 text-gray-800',
    gray: 'bg-gray-100 text-gray-800',
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
    success: 'bg-green-500',
    warning: 'bg-amber-500',
    danger: 'bg-red-500',
    info: 'bg-purple-500',
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
