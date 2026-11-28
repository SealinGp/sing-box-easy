<script setup lang="ts">
/** Retain draft controls; reverse the same transition on rapid toggles. */
defineProps<{ open: boolean }>()
</script>

<template>
  <div class="ui-disclosure" :class="{ 'is-open': open }" :inert="!open" :aria-hidden="!open">
    <div><slot /></div>
  </div>
</template>
