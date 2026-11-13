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
  const base = 'inline-flex items-center font-medium rounded-badge border'

  const variants = {
    primary: 'bg-primary/10 text-primary border-primary/25',
    success: 'bg-success/10 text-success border-success/25',
    warning: 'bg-warning/10 text-warning border-warning/25',
    danger: 'bg-danger/10 text-danger border-danger/25',
    info: 'bg-info/10 text-info border-info/25',
    secondary: 'bg-bg-tertiary text-text-secondary border-border',
    gray: 'bg-bg-tertiary text-text-secondary border-border',
  }

  /*
   * Compact density pass. `md` was `px-2.5 py-0.5 text-sm` — a 13px label in a
   * 20px pill, which made the badge the tallest thing in a table cell that had
   * no action button, and read as heavy next to 13px body copy beside it.
   *
   * `sm` and `md` deliberately share a font size and differ only in padding,
   * the same rule Button follows: below 11px a pill stops being legible at a
   * glance, which is the only reason to use a badge instead of plain text.
   */
  const sizes = {
    sm: 'px-1.5 py-0 text-xs',
    md: 'px-2 py-0.5 text-xs',
    lg: 'px-2.5 py-0.5 text-sm',
  }

  return `${base} ${variants[props.variant]} ${sizes[props.size]}`
})

const dotColorClass = computed(() => {
  const colors = {
    primary: 'bg-primary',
    success: 'bg-success',
    warning: 'bg-warning',
    danger: 'bg-danger',
    info: 'bg-info',
    secondary: 'bg-text-secondary',
    gray: 'bg-text-secondary',
  }
  return colors[props.variant]
})
</script>

<template>
  <span :class="classes">
    <span v-if="dot" :class="['w-1.5 h-1.5 rounded-pill mr-1', dotColorClass]"></span>
    <slot />
  </span>
</template>
