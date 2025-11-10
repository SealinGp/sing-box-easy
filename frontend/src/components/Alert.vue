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
  const base = 'flex items-start p-4 rounded-lg border'
  const types = {
    success: 'bg-green-50 border-green-200 text-green-800',
    warning: 'bg-amber-50 border-amber-200 text-amber-800',
    error: 'bg-red-50 border-red-200 text-red-800',
    info: 'bg-blue-50 border-blue-200 text-blue-800',
  }
  return `${base} ${types[props.type]}`
})

const iconColorClass = computed(() => {
  const colors = {
    success: 'text-green-500',
    warning: 'text-amber-500',
    error: 'text-red-500',
    info: 'text-blue-500',
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
