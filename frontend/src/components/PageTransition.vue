<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ viewKey: string }>()
const stack = ref<HTMLElement | null>(null)

function leave(element: Element) {
  element.setAttribute('inert', '')
  if (stack.value) {
    stack.value.style.minHeight = `${stack.value.getBoundingClientRect().height}px`
  }
}
function cancelLeave(element: Element) {
  element.removeAttribute('inert')
}
function entered() {
  stack.value?.style.removeProperty('min-height')
}
</script>

<template>
  <div ref="stack" class="ui-page-stack">
    <!-- Finish the outgoing view before introducing its replacement. Keep the
         occupied height until entry finishes so the content area stays put. -->
    <Transition name="ui-page" mode="out-in" @before-leave="leave" @leave-cancelled="cancelLeave" @after-enter="entered">
      <div :key="viewKey" class="ui-page-view"><slot /></div>
    </Transition>
  </div>
</template>
