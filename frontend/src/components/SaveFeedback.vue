<script setup lang="ts">
defineProps<{ error?: string; message?: string }>()
</script>

<template>
  <div class="relative min-w-0 flex-1 text-sm" aria-live="polite" aria-atomic="true">
    <Transition name="ui-field">
      <p v-if="error" key="error" class="text-danger break-words" role="alert">{{ error }}</p>
      <p v-else-if="message" key="saved" class="text-success break-words">{{ message }}</p>
    </Transition>
  </div>
</template>
