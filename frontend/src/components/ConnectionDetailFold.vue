<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ collapsed: [] }>()
const expanded = ref(false)
const reduced = window.matchMedia('(prefers-reduced-motion: reduce)')
let frame = 0
let mounted = false

function settle() {
  if (!props.open) emit('collapsed')
}
function update() {
  cancelAnimationFrame(frame)
  if (reduced.matches) { expanded.value = props.open; settle(); return }
  if (!mounted) return
  if (!props.open && !expanded.value) { settle(); return }
  expanded.value = props.open
}
watch(() => props.open, update)
onMounted(() => {
  mounted = true
  if (reduced.matches) { update(); return }
  // Paint the collapsed grid before expanding. Subsequent reversals retarget
  // the existing CSS transition, without a timer or remounting the details.
  frame = requestAnimationFrame(() => {
    frame = requestAnimationFrame(() => { expanded.value = props.open; if (!props.open) settle() })
  })
})
reduced.addEventListener('change', update)
onBeforeUnmount(() => {
  cancelAnimationFrame(frame)
  reduced.removeEventListener('change', update)
})
function ended(event: TransitionEvent) {
  if (event.target === event.currentTarget && event.propertyName === 'grid-template-rows') settle()
}
</script>

<template>
  <div class="connection-detail-fold" :class="{ 'is-open': expanded }" :inert="!open" :aria-hidden="!open" @transitionend="ended">
    <div><slot /></div>
  </div>
</template>

<style scoped>
.connection-detail-fold {
  display: grid;
  grid-template-rows: 0fr;
  opacity: 0;
  transition: grid-template-rows var(--motion-panel) var(--motion-continuous), opacity var(--motion-feedback) var(--motion-continuous);
}
.connection-detail-fold.is-open { grid-template-rows: 1fr; opacity: 1; }
.connection-detail-fold > div { min-height: 0; overflow: clip; }
</style>
