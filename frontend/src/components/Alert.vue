<script setup lang="ts">
import { computed } from 'vue'
import {
  CheckCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
  XCircleIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'

interface Props {
  type?: 'success' | 'warning' | 'error' | 'info'
  title?: string
  closable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  type: 'info',
  closable: false,
})

const emit = defineEmits<{
  close: []
}>()

const icon = computed(() => {
  switch (props.type) {
    case 'success':
      return CheckCircleIcon
    case 'warning':
      return ExclamationTriangleIcon
    case 'error':
      return XCircleIcon
    default:
      return InformationCircleIcon
  }
})

const classes = computed(() => {
  const base = 'flex items-start p-4 rounded-lg border shadow-sm backdrop-blur-xl'
  const types = {
    success: 'bg-emerald-500/15 border-emerald-500/25 text-emerald-800 dark:text-emerald-100',
    warning: 'bg-amber-500/15 border-amber-500/25 text-amber-800 dark:text-amber-100',
    error: 'bg-red-500/15 border-red-500/25 text-red-800 dark:text-red-100',
    info: 'bg-sky-500/15 border-sky-500/25 text-sky-800 dark:text-sky-100',
  }
  return `${base} ${types[props.type]}`
})

const iconColorClass = computed(() => {
  const colors = {
    success: 'text-emerald-500',
    warning: 'text-amber-500',
    error: 'text-red-500',
    info: 'text-sky-500',
  }
  return colors[props.type]
})
</script>

<template>
  <div :class="classes">
    <component :is="icon" :class="['h-5 w-5 flex-shrink-0', iconColorClass]" />
    <div class="ml-3 flex-1">
      <h3 v-if="title" class="text-sm font-medium mb-1">{{ title }}</h3>
      <div class="text-sm">
        <slot />
      </div>
    </div>
    <button
      v-if="closable"
      type="button"
      class="ml-3 flex-shrink-0 opacity-70 hover:opacity-100 transition-opacity"
      @click="emit('close')"
    >
      <XMarkIcon class="h-5 w-5" />
    </button>
  </div>
</template>
