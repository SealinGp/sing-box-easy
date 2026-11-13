<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'primary' | 'success' | 'warning' | 'danger' | 'info' | 'secondary' | 'gray'
  busy?: boolean
}>(), { variant: 'secondary' })
</script>

<template>
  <span class="status-indicator" :data-tone="variant" :aria-busy="busy || undefined">
    <span class="status-indicator-mark" :class="{ 'status-indicator-busy': busy }" aria-hidden="true" />
    <span><slot /></span>
  </span>
</template>
