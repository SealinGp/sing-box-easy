<script setup lang="ts">
import { ref } from 'vue'

defineProps<{ viewKey: string; fill?: boolean }>()
const stack = ref<HTMLElement | null>(null)

function beforeLeave(element: Element) {
  element.setAttribute('inert', '')
  if (stack.value) {
    stack.value.style.minHeight = `${stack.value.getBoundingClientRect().height}px`
  }
}
function leave(_element: Element, done: () => void) {
  // Remove the previous view in this render, rather than delaying the next
  // page's mount and data requests until an exit animation has finished.
  done()
}
function entered() {
  stack.value?.style.removeProperty('min-height')
}
</script>

<template>
  <div ref="stack" class="ui-page-stack" :class="{ 'ui-page-fill': fill }">
    <!-- Only the incoming view animates. Synchronous removal prevents a paint
         with overlapping pages; the height guard keeps loading content stable. -->
    <Transition name="ui-route" @before-leave="beforeLeave" @leave="leave"
      @after-enter="entered" @enter-cancelled="entered">
      <div :key="viewKey" class="ui-page-view"><slot /></div>
    </Transition>
  </div>
</template>
