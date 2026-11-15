<script setup lang="ts">
import { ref } from 'vue'
import Tooltip from './Tooltip.vue'
defineProps<{ text: string }>()
const anchor = ref<HTMLElement>()
function overflows() {
  return !!anchor.value && anchor.value.scrollWidth > anchor.value.clientWidth
}
</script>
<template>
  <Tooltip :should-show="overflows">
    <template #default="{ describedBy }">
      <span ref="anchor" tabindex="0" class="truncated-text" :aria-describedby="describedBy">{{ text }}</span>
    </template>
    <template #content>{{ text }}</template>
  </Tooltip>
</template>
<style scoped>
.truncated-text { display: block; max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.truncated-text:focus-visible { outline: 2px solid var(--color-primary-500); outline-offset: -2px; }
</style>
