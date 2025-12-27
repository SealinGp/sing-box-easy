<script setup lang="ts">
import { computed } from 'vue'
import {
  CheckCircleIcon,
  ExclamationTriangleIcon,
  InformationCircleIcon,
  XCircleIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import type { NotificationType } from '../composables/useNotification'

interface Props {
  type: NotificationType
  title?: string
  message: string
  closable?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  closable: true,
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
  const base = 'flex items-start gap-3 p-4 rounded-lg shadow-lg border min-w-[320px] max-w-md'
  const types = {
    success: 'bg-green-50 border-green-200 text-green-800',
    warning: 'bg-amber-50 border-amber-200 text-amber-800',
    error: 'bg-red-50 border-red-200 text-red-800',
    info: 'bg-violet-50 border-violet-200 text-violet-800',
  }
  return `${base} ${types[props.type]}`
})

const iconColorClass = computed(() => {
  const colors = {
    success: 'text-green-500',
    warning: 'text-amber-500',
    error: 'text-red-500',
    info: 'text-violet-500',
  }
  return colors[props.type]
})
</script>

<template>
  <div :class="classes">
    <component :is="icon" :class="['h-5 w-5 flex-shrink-0', iconColorClass]" />
    <div class="flex-1 min-w-0">
      <h3 v-if="title" class="text-sm font-semibold mb-1">{{ title }}</h3>
      <div class="text-sm">{{ message }}</div>
    </div>
    <button
      v-if="closable"
      type="button"
      class="flex-shrink-0 opacity-70 hover:opacity-100 transition-opacity"
      @click="emit('close')"
    >
      <XMarkIcon class="h-5 w-5" />
    </button>
  </div>
</template>
