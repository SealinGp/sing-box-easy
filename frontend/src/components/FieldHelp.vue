<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { InformationCircleIcon } from '@heroicons/vue/24/outline'
const props = defineProps<{ label: string; text: string }>()
const { t } = useI18n()
const id = useId()
const trigger = ref<HTMLButtonElement>()
const tooltip = ref<HTMLElement>()
const open = ref(false)
const position = ref({ left: '0px', top: '0px' })
let timer: ReturnType<typeof setTimeout> | undefined
function close() { clearTimeout(timer); open.value = false }
function leave() { timer = setTimeout(close, 150) }
function keep() { clearTimeout(timer) }
async function show() {
  keep()
  open.value = true
  await nextTick()
  const anchor = trigger.value?.getBoundingClientRect()
  const box = tooltip.value?.getBoundingClientRect()
  if (!anchor || !box) return
  position.value = {
    left: `${Math.max(12, Math.min(anchor.left, innerWidth - box.width - 12))}px`,
    top: `${anchor.bottom + box.height + 8 <= innerHeight - 12 ? anchor.bottom + 8 : Math.max(12, anchor.top - box.height - 8)}px`,
  }
}
function outside(event: PointerEvent) {
  const target = event.target as Node
  if (!trigger.value?.contains(target) && !tooltip.value?.contains(target)) close()
}
function escape(event: KeyboardEvent) {
  if (open.value && event.key === 'Escape') { event.preventDefault(); event.stopPropagation(); close() }
}
document.addEventListener('keydown', escape, true)
document.addEventListener('pointerdown', outside, true)
window.addEventListener('scroll', close, true)
window.addEventListener('resize', close)
onBeforeUnmount(() => {
  close()
  document.removeEventListener('keydown', escape, true)
  document.removeEventListener('pointerdown', outside, true)
  window.removeEventListener('scroll', close, true)
  window.removeEventListener('resize', close)
})
</script>
<template>
  <button ref="trigger" type="button" class="inline-flex h-6 w-6 shrink-0 items-center justify-center rounded text-text-secondary hover:text-primary focus-visible:outline-2 focus-visible:outline-primary"
    :aria-label="t('common.fieldHelp', { name: props.label })" :aria-describedby="open ? id : undefined" :aria-controls="open ? id : undefined"
    @mouseenter="show" @mouseleave="leave" @focus="show" @blur="leave" @click="show" @keydown.esc.stop.prevent="close">
    <InformationCircleIcon class="h-4 w-4" aria-hidden="true" />
  </button>
  <Teleport to="body">
    <div v-if="open" :id="id" ref="tooltip" role="tooltip" class="fixed z-[10000] w-max max-w-[min(18rem,calc(100vw-1.5rem))] rounded-control border border-border bg-bg-secondary px-3 py-2 text-xs leading-relaxed text-text-primary shadow-surface" :style="position" @mouseenter="keep" @mouseleave="leave">
      {{ text }}
    </div>
  </Teleport>
</template>
