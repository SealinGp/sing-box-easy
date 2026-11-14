<script setup lang="ts">
import { computed, onBeforeUnmount, ref, useAttrs, useId } from 'vue'
import PrimePopover, { type PopoverPassThroughOptions, type PopoverProps } from 'primevue/popover'
import { popoverMotion } from '../composables/useInterfaceMotion'
import { ptViewMerge } from './utils'
import { twMerge } from 'tailwind-merge'
import { overlayOrigin } from '../composables/useOverlayAnchor'
import { popoverPlacement, type PopoverOrigin } from '../utils/popoverPlacement'

interface Props extends /* @vue-ignore */ PopoverProps {
  dismissable?: boolean
  panelClass?: string
  contentClass?: string
}
const props = defineProps<Props>()
const emit = defineEmits<{ show: []; hide: [] }>()
const attrs = useAttrs()
const autoId = useId()
const id = computed(() => String(attrs.id ?? autoId))
const panel = ref<InstanceType<typeof PrimePopover> | null>(null)
let origin: PopoverOrigin | undefined
let trigger: HTMLElement | null = null
let returnFocus = false
let resizeObserver: ResizeObserver | null = null
let alignmentFrame = 0
let openingFrame = 0
let shown = false
let pointerStartedInside = false

// Select menus are teleported to body. Selection happens on mousedown and may
// remove the option before click, so remember where the gesture began.
function ownsTarget(target: EventTarget | null): boolean {
  if (!(target instanceof Node)) return false
  const root = document.getElementById(id.value)
  if (root?.contains(target) || trigger?.contains(target)) return true
  return [...(root?.querySelectorAll('[aria-controls]') ?? [])].some(control =>
    (control.getAttribute('aria-controls') ?? '').split(/\s+/).some(controlledId => {
      const controlled = document.getElementById(controlledId)
      const overlay = controlled?.closest('[data-pc-section="overlay"]') ?? controlled
      return overlay?.contains(target)
    }),
  )
}
function onPointerDown(event: PointerEvent) {
  pointerStartedInside = ownsTarget(event.target)
  returnFocus = pointerStartedInside
}
function onOutsideClick(event: MouseEvent) {
  const inside = ownsTarget(event.target) || (event.detail > 0 && pointerStartedInside)
  pointerStartedInside = false
  if (!inside && props.dismissable !== false) hide()
}

function trackFocus(event: Event) {
  returnFocus = ownsTarget(event.target)
}
function removeListeners() {
  document.removeEventListener('focusin', trackFocus, true)
  document.removeEventListener('pointerdown', onPointerDown, true)
  document.removeEventListener('click', onOutsideClick)
  window.removeEventListener('resize', alignWithinViewport)
  resizeObserver?.disconnect()
  cancelAnimationFrame(alignmentFrame)
}
onBeforeUnmount(removeListeners)
onBeforeUnmount(() => cancelAnimationFrame(openingFrame))

const theme = computed<PopoverPassThroughOptions>(() => ({
  root: {
    class: twMerge('liquid-glass-float rounded-surface w-80 max-w-[calc(100vw-2rem)] max-h-[70dvh] overflow-y-auto', props.panelClass),
    'aria-modal': 'false',

  },
  content: twMerge('p-4', props.contentClass),
  transition: popoverMotion,
}))

function toggle(event: Event) {
  const target = event.currentTarget as HTMLElement | null
  if (shown || openingFrame) hide()
  else if (target) showAt(target, event instanceof MouseEvent && event.detail > 0 ? { x: event.clientX, y: event.clientY } : undefined)
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
  alignWithinViewport()
  window.addEventListener('resize', alignWithinViewport)
  const root = document.getElementById(id.value)
  const first = root?.querySelector<HTMLElement>('[autofocus]')
    ?? root?.querySelector<HTMLElement>('input:not(:disabled):not([type="hidden"]), textarea:not(:disabled)')
    ?? root?.querySelector<HTMLElement>('button:not(:disabled), a[href], [tabindex="0"]')
  first?.focus({ preventScroll: true })
  returnFocus = true
  document.addEventListener('focusin', trackFocus, true)
  document.addEventListener('pointerdown', onPointerDown, true)
  document.addEventListener('click', onOutsideClick)
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
function showAt(target: HTMLElement, point?: PopoverOrigin) {
  trigger = target
  origin = point ?? overlayOrigin(target)
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
      shown = true
      panel.value?.show(event, target)
    })
  })
}
function alignWithinViewport() {
  const root = document.getElementById(id.value)
  if (!root || !trigger) return
  const anchor = trigger.getBoundingClientRect()
  const point = origin ?? { x: anchor.left + anchor.width / 2, y: anchor.top + anchor.height / 2 }
  // Measure untransformed dimensions: entry animation scales the bounding rect.
  root.style.maxWidth = ''
  root.style.maxHeight = ''
  const placement = popoverPlacement(point, { width: window.innerWidth, height: window.innerHeight }, {
    width: root.offsetWidth, height: root.offsetHeight,
  })
  root.style.maxWidth = `${placement.maxWidth}px`
  root.style.maxHeight = `${placement.maxHeight}px`
  // Width constraints can wrap content and change height. Measure once more.
  const final = popoverPlacement(point, { width: window.innerWidth, height: window.innerHeight }, {
    width: root.offsetWidth, height: root.offsetHeight,
  })
  root.style.left = `${final.left + window.scrollX}px`
  root.style.top = `${final.top + window.scrollY}px`
  root.style.marginTop = '0'
  root.style.transformOrigin = final.origin
}
defineExpose({ toggle, showAt, hide })
</script>

<template>
  <PrimePopover ref="panel" :id="id" :dismissable="false" unstyled :pt="theme" :pt-options="{mergeProps: ptViewMerge}"
    @show="onShow" @hide="onHide" @keydown.capture="onKeydown">
    <slot />
  </PrimePopover>
</template>
