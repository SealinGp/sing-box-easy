<script setup lang="ts">
import { computed, onBeforeUnmount, ref, useAttrs, useId } from 'vue'
import PrimePopover, { type PopoverPassThroughOptions, type PopoverProps } from 'primevue/popover'
import { popoverMotion } from '../composables/useInterfaceMotion'
import { ptViewMerge } from './utils'
import { twMerge } from 'tailwind-merge'

interface Props extends /* @vue-ignore */ PopoverProps {
  panelClass?: string
  contentClass?: string
}
const props = defineProps<Props>()
const emit = defineEmits<{ show: []; hide: [] }>()
const attrs = useAttrs()
const autoId = useId()
const id = computed(() => String(attrs.id ?? autoId))
const panel = ref<InstanceType<typeof PrimePopover> | null>(null)
const availableHeight = ref<number>()
let trigger: HTMLElement | null = null
let returnFocus = false
let resizeObserver: ResizeObserver | null = null
let alignmentFrame = 0
let openingFrame = 0
let shown = false

function trackFocus(event: Event) {
  returnFocus = !!document.getElementById(id.value)?.contains(event.target as Node)
}
function removeListeners() {
  document.removeEventListener('focusin', trackFocus, true)
  document.removeEventListener('pointerdown', trackFocus, true)
  resizeObserver?.disconnect()
  cancelAnimationFrame(alignmentFrame)
}
onBeforeUnmount(removeListeners)
onBeforeUnmount(() => cancelAnimationFrame(openingFrame))

const theme = computed<PopoverPassThroughOptions>(() => ({
  root: {
    class: twMerge('liquid-glass-float rounded-surface mt-2 w-80 max-w-[calc(100vw-2rem)] max-h-[70dvh] overflow-y-auto', props.panelClass),
    'aria-modal': 'false',
    style: availableHeight.value ? { maxHeight: `${availableHeight.value}px` } : undefined,
  },
  content: twMerge('p-4', props.contentClass),
  transition: popoverMotion,
}))

function toggle(event: Event) {
  const target = event.currentTarget as HTMLElement | null
  if (shown || openingFrame) hide()
  else if (target) showAt(target)
}
function hide() {
  cancelAnimationFrame(openingFrame)
  openingFrame = 0
  panel.value?.hide()
}
function onKeydown(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  // Select consumes Escape even when closed. Let an open picker dismiss its
  // own list first; otherwise Escape belongs to this contextual panel.
  const hasOpenPicker = [...document.querySelectorAll('[role="listbox"]')]
    .some(element => element.getClientRects().length > 0)
  if (hasOpenPicker) return
  returnFocus = true
  event.preventDefault()
  event.stopPropagation()
  hide()
}
function onShow() {
  const root = document.getElementById(id.value)
  const first = root?.querySelector<HTMLElement>('[autofocus]')
    ?? root?.querySelector<HTMLElement>('input:not(:disabled):not([type="hidden"]), textarea:not(:disabled)')
    ?? root?.querySelector<HTMLElement>('button:not(:disabled), a[href], [tabindex="0"]')
  first?.focus({ preventScroll: true })
  returnFocus = true
  document.addEventListener('focusin', trackFocus, true)
  document.addEventListener('pointerdown', trackFocus, true)
  // Async details and expandable fields can resize a panel after it opens.
  // Reposition at its anchor so growing content stays within the viewport.
  resizeObserver = new ResizeObserver(() => {
    cancelAnimationFrame(alignmentFrame)
    alignmentFrame = requestAnimationFrame(alignWithinViewport)
  })
  if (root) resizeObserver.observe(root)
  emit('show')
}
function onHide() {
  // Do not steal focus from a person who clicked elsewhere to dismiss.
  removeListeners()
  shown = false
  if (returnFocus) trigger?.focus({ preventScroll: true })
  emit('hide')
}
function showAt(target: HTMLElement) {
  trigger = target
  // PrimeVue uses currentTarget to ignore the opening click in its document
  // listener. Preserve that anchor when opening after a reactive update too.
  const event = new Event('open')
  Object.defineProperty(event, 'currentTarget', { value: target })
  cancelAnimationFrame(openingFrame)
  // Keyboard focus and scroll-into-view can queue a table scroll until the
  // next frame. Open after layout settles so that pending scroll cannot
  // dismiss a newly opened panel through PrimeVue's scroll listener.
  openingFrame = requestAnimationFrame(() => {
    openingFrame = requestAnimationFrame(() => {
      openingFrame = 0
      if (!target.isConnected) return
      const bounds = target.getBoundingClientRect()
      const above = bounds.top - 24
      const below = window.innerHeight - bounds.bottom - 24
      availableHeight.value = Math.min(window.innerHeight * .8, Math.max(above, below))
      shown = true
      panel.value?.show(event, target)
    })
  })
}
function alignWithinViewport() {
  panel.value?.alignOverlay()
  const root = document.getElementById(id.value)
  if (!root) return
  const bounds = root.getBoundingClientRect()
  const left = Math.max(16, Math.min(bounds.left, window.innerWidth - bounds.width - 16))
  root.style.left = `${left + window.scrollX}px`
}
defineExpose({ toggle, showAt, hide })
</script>

<template>
  <PrimePopover ref="panel" :id="id" unstyled :pt="theme" :pt-options="{mergeProps: ptViewMerge}"
    @show="onShow" @hide="onHide" @keydown.capture="onKeydown">
    <slot />
  </PrimePopover>
</template>
