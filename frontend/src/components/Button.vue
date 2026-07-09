<script setup lang="ts">
import { computed } from 'vue'

interface Props {
  variant?: 'primary' | 'secondary' | 'danger' | 'success' | 'ghost'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
  disabled?: boolean
  fullWidth?: boolean
  action?: boolean
  type?: 'button' | 'submit' | 'reset'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  loading: false,
  disabled: false,
  fullWidth: false,
  action: false,
  type: 'button',
})

const classes = computed(() => {
  const shadow = props.action || props.variant === 'ghost' ? 'shadow-none' : 'shadow-sm'
  const base = `inline-flex items-center justify-center font-semibold rounded-full transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2 dark:focus:ring-offset-gray-900 ${shadow}`

  const variants = {
    primary: 'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500 disabled:bg-blue-300 shadow-blue-500/25',
    secondary: 'bg-white/55 dark:bg-white/10 text-gray-900 dark:text-gray-100 hover:bg-white/75 dark:hover:bg-white/15 border border-white/40 dark:border-white/10 focus:ring-gray-500 disabled:bg-white/25 dark:disabled:bg-white/5',
    danger: 'bg-red-500/90 text-white hover:bg-red-600 focus:ring-red-500 disabled:bg-red-300',
    success: 'bg-emerald-500/90 text-white hover:bg-emerald-600 focus:ring-emerald-500 disabled:bg-emerald-300',
    ghost: 'bg-white/10 text-gray-700 dark:text-gray-300 hover:bg-white/45 dark:hover:bg-white/10 border border-transparent hover:border-white/30 dark:hover:border-white/10 focus:ring-gray-500',
  }

  const sizes = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-base',
    lg: 'px-6 py-3 text-lg',
  }

  const width = props.fullWidth ? 'w-full' : ''

  return [base, variants[props.variant], sizes[props.size], width].join(' ')
})
</script>

<template>
  <button
    :type="type"
    :class="classes"
    :disabled="disabled || loading"
  >
    <svg
      v-if="loading"
      class="animate-spin -ml-1 mr-2 h-4 w-4"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
    >
      <circle
        class="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        stroke-width="4"
      ></circle>
      <path
        class="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
      ></path>
    </svg>
    <slot />
  </button>
</template>
