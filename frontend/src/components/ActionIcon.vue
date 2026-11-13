<script setup lang="ts">
defineProps<{ action: 'edit' | 'delete' }>()
</script>

<template>
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.65"
    stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4 shrink-0" aria-hidden="true">
    <template v-if="action === 'delete'">
      <path d="M4 7h16M9 7V4h6v3M6.5 7l.8 13h9.4l.8-13M10 10.5v6M14 10.5v6" />
    </template>
    <template v-else>
      <path d="m14.5 5.5 4 4M4 20l4.5-1 11-11a2.8 2.8 0 0 0-4-4l-11 11L4 20Z" />
    </template>
  </svg>
</template>
