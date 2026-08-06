<script setup lang="ts">
import { computed } from 'vue'

/**
 * The single button in the design system.
 *
 * This component absorbed `src/volt/Button.vue`, which wrapped PrimeVue's
 * unstyled Button. The two disagreed on brand colour (primary-600 vs blue-600)
 * and radius (`rounded-md` vs `rounded-full`), so which one a screen showed
 * depended on which import path the author happened to pick.
 *
 * `severity` is retained purely as a compatibility alias for the PrimeVue
 * spelling used by the former volt call sites; `variant` is the canonical prop.
 */
type Variant = 'primary' | 'secondary' | 'danger' | 'success' | 'ghost'
type Severity = 'primary' | 'secondary' | 'success' | 'info' | 'warn' | 'danger' | 'help' | 'contrast'

interface Props {
  variant?: Variant
  /** PrimeVue-style alias for `variant`. Takes precedence when both are set. */
  severity?: Severity
  /** Convenience for the common `<Button>{{ label }}</Button>` case. */
  label?: string
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
  disabled?: boolean
  fullWidth?: boolean
  /** Drops the resting shadow — for buttons sitting inside toolbars/tables. */
  action?: boolean
  /** Circular. Reserved for icon-only controls; never for text buttons. */
  pill?: boolean
  type?: 'button' | 'submit' | 'reset'
}

const props = withDefaults(defineProps<Props>(), {
  variant: 'primary',
  size: 'md',
  loading: false,
  disabled: false,
  fullWidth: false,
  action: false,
  pill: false,
  type: 'button',
})

const SEVERITY_TO_VARIANT: Record<Severity, Variant> = {
  primary: 'primary',
  secondary: 'secondary',
  success: 'success',
  info: 'primary',
  warn: 'primary',
  danger: 'danger',
  help: 'primary',
  contrast: 'secondary',
}

const resolvedVariant = computed<Variant>(() =>
  props.severity ? SEVERITY_TO_VARIANT[props.severity] : props.variant,
)

const VARIANTS: Record<Variant, string> = {
  primary:
    'bg-primary text-white hover:bg-primary-hover focus-visible:ring-primary disabled:bg-primary/40',
  secondary:
    'bg-white/70 dark:bg-white/10 text-gray-900 dark:text-gray-100 border border-border hover:bg-white dark:hover:bg-white/15 focus-visible:ring-gray-400 disabled:opacity-50',
  danger:
    'bg-danger text-white hover:bg-red-600 focus-visible:ring-danger disabled:bg-danger/40',
  success:
    'bg-success text-white hover:bg-emerald-600 focus-visible:ring-success disabled:bg-success/40',
  ghost:
    'bg-transparent text-gray-700 dark:text-gray-300 border border-transparent hover:bg-black/5 dark:hover:bg-white/10 focus-visible:ring-gray-400',
}

const SIZES: Record<NonNullable<Props['size']>, string> = {
  sm: 'px-3 py-1.5 text-sm gap-1.5',
  md: 'px-4 py-2 text-sm gap-2',
  lg: 'px-6 py-2.5 text-base gap-2',
}

const classes = computed(() =>
  [
    'inline-flex items-center justify-center font-semibold transition-colors duration-200',
    'focus:outline-none focus-visible:ring-2 focus-visible:ring-offset-2',
    'dark:focus-visible:ring-offset-gray-900 disabled:cursor-not-allowed',
    props.pill ? 'rounded-pill' : 'rounded-control',
    props.action || resolvedVariant.value === 'ghost' ? '' : 'shadow-surface',
    VARIANTS[resolvedVariant.value],
    SIZES[props.size],
    props.fullWidth ? 'w-full' : '',
  ]
    .filter(Boolean)
    .join(' '),
)
</script>

<template>
  <button :type="type" :class="classes" :disabled="disabled || loading" :aria-busy="loading">
    <svg
      v-if="loading"
      class="-ml-1 h-4 w-4 animate-spin"
      xmlns="http://www.w3.org/2000/svg"
      fill="none"
      viewBox="0 0 24 24"
      aria-hidden="true"
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
    <slot>{{ label }}</slot>
  </button>
</template>
