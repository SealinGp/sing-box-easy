<script setup lang="ts">
import { ref } from 'vue'
import Tooltip from './Tooltip.vue'
const content = ref<HTMLElement>()
const text = ref('')
function read() { text.value = content.value?.innerText.trim() ?? '' }
</script>
<template>
  <Tooltip @mouseenter="read" @focusin="read" @click="read">
    <template #default="{ describedBy }">
      <div ref="content" class="collection-cell-content" tabindex="0" :aria-describedby="describedBy"><slot /></div>
    </template>
    <template #content><span class="whitespace-pre-wrap">{{ text }}</span></template>
  </Tooltip>
</template>
<style>
.collection-cell-content { max-height: 28px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.collection-cell-content:focus-visible { outline: 2px solid var(--color-primary); outline-offset: -2px; }
.collection-cell-content > .flex-col { display: flex; flex-direction: row; flex-wrap: nowrap; }
.collection-cell-content > .flex-col > :not(:first-child) { display: none; }
</style>
