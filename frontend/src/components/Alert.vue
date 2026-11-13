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
  const base = 'flex items-start p-4 rounded-surface border shadow-surface backdrop-blur-xl'
  const types = {
    success: 'bg-success/10 border-success/25 text-success',
    warning: 'bg-warning/10 border-warning/25 text-warning',
    error: 'bg-danger/10 border-danger/25 text-danger',
    info: 'bg-info/10 border-info/25 text-info',
  }
  return `${base} ${types[props.type]}`
})

const iconColorClass = computed(() => {
  const colors = {
    success: 'text-success',
    warning: 'text-warning',
    error: 'text-danger',
    info: 'text-info',
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
      :aria-label="$t('common.close')"
      class="ml-3 flex-shrink-0 opacity-70 hover:opacity-100 transition-opacity"
      @click="emit('close')"
    >
      <XMarkIcon class="h-5 w-5" />
    </button>
  </div>
</template>
