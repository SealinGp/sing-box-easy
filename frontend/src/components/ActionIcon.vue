<script setup lang="ts">
/** Compact 20-unit action glyphs, with consistent weight and rounded joins. */
defineProps<{ action: 'copy' | 'edit' | 'delete' | 'check' | 'close' }>()
</script>

<template>
  <svg viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.5"
    stroke-linecap="round" stroke-linejoin="round" class="h-5 w-5 shrink-0" aria-hidden="true">
    <template v-if="action === 'copy'">
      <rect x="7" y="6.5" width="9" height="10" rx="2" />
      <path d="M12.5 6.5V5a2 2 0 0 0-2-2H6a2 2 0 0 0-2 2v6a2 2 0 0 0 2 2h1" />
    </template>
    <template v-else-if="action === 'delete'">
      <path d="M4 6h12M8 6V4.75a1.25 1.25 0 0 1 1.25-1.25h1.5A1.25 1.25 0 0 1 12 4.75V6M5.5 6v8.5a2 2 0 0 0 2 2h5a2 2 0 0 0 2-2V6M8.5 9v4.5M11.5 9v4.5" />
    </template>
    <template v-else-if="action === 'edit'">
      <path d="m11.75 4.75 3.5 3.5M4 16l3.75-.75 8-8a2.47 2.47 0 0 0-3.5-3.5l-8 8L4 16Z" />
    </template>
    <path v-else-if="action === 'check'" d="m4.5 10 3.5 3.5 7.5-7.5" />
    <path v-else d="m5.5 5.5 9 9m0-9-9 9" />
  </svg>
</template>
