<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch, useAttrs, useId } from 'vue'
import PrimePopover, { type PopoverPassThroughOptions, type PopoverProps } from 'primevue/popover'
import { popoverMotion } from '../composables/useInterfaceMotion'
import { ptViewMerge } from './utils'
import { twMerge } from 'tailwind-merge'
import '../style/panel-forms.css'
import { useWorkspaceActive } from '../composables/workspaceContext'
import { popoverPlacement, type PopoverSide } from '../utils/popoverPlacement'

interface Props extends /* @vue-ignore */ PopoverProps {
  dismissable?: boolean
  closeOnEscape?: boolean
  /** Action menus dismiss even when the outside control stops click bubbling. */
  interaction?: 'form' | 'menu'
  panelClass?: string
  contentClass?: string
}
const props = withDefaults(defineProps<Props>(), {
  dismissable: true,
  closeOnEscape: true,
  interaction: 'form',
})
const emit = defineEmits<{ show: []; hide: [] }>()
const workspaceActive = useWorkspaceActive()
watch(workspaceActive, async active => {
  if (!active) { returnFocus = false; removeListeners() }
  else if (shown) { await nextTick(); onShow(true) }
})
const attrs = useAttrs()
const autoId = useId()
const id = computed(() => String(attrs.id ?? autoId))
const panel = ref<InstanceType<typeof PrimePopover> | null>(null)
let side: PopoverSide | undefined
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
  // Teleported nested popovers (including node history and its own panels)
  // remain part of the parent interaction even though they live under body.
  let nested = (target instanceof Element ? target : target.parentElement)?.closest<HTMLElement>('[data-overlay-owner]')
  const visited = new Set<string>()
  while (nested?.dataset.overlayOwner) {
    const owner = nested.dataset.overlayOwner
    if (owner === id.value) return true
    if (visited.has(owner)) break
    visited.add(owner)
    nested = document.getElementById(owner)?.closest<HTMLElement>('[data-overlay-owner]')
  }
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
  if (!workspaceActive.value) return
  // Confirmation dialogs belong to the current operation, not an outside dismissal.
  if (props.interaction === 'form' && event.target instanceof Element && event.target.closest('[aria-modal="true"], .shared-confirmation')) return
  // Workspace navigation suspends contextual forms instead of cancelling their drafts.
  if (props.interaction === 'form' && event.target instanceof Element && event.target.closest('.workspace-tabs, .workspace-actions, .nav-destination, .nav-brand, .navigation-search')) return
  const inside = ownsTarget(event.target) || (event.detail > 0 && pointerStartedInside)
  pointerStartedInside = false
  if (!inside && props.dismissable !== false) hide()
}

function trackFocus(event: Event) {
  returnFocus = ownsTarget(event.target)
}
function onScroll(event: Event) {
  if (document.getElementById(id.value)?.contains(event.target as Node)) return
  cancelAnimationFrame(alignmentFrame)
  alignmentFrame = requestAnimationFrame(alignWithinViewport)
}
function removeListeners() {
  document.removeEventListener('focusin', trackFocus, true)
  document.removeEventListener('pointerdown', onPointerDown, true)
  document.removeEventListener('click', onOutsideClick, props.interaction === 'menu')
  window.removeEventListener('resize', alignWithinViewport)
  document.removeEventListener('scroll', onScroll, true)
  resizeObserver?.disconnect()
  cancelAnimationFrame(alignmentFrame)
}
onBeforeUnmount(removeListeners)
onBeforeUnmount(() => cancelAnimationFrame(openingFrame))

const theme = computed<PopoverPassThroughOptions>(() => ({
  root: {
    class: twMerge('liquid-glass-float rounded-surface w-80 max-w-[calc(100vw-2rem)] max-h-[70dvh] overflow-y-auto', props.panelClass),
    'aria-modal': 'false',
    'aria-hidden': workspaceActive.value ? undefined : 'true',
    inert: !workspaceActive.value,
    style: workspaceActive.value ? undefined : { display: 'none' },

  },
  content: twMerge('panel-form-body p-4', props.contentClass),
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
  if (!props.closeOnEscape) { event.stopPropagation(); return }
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
function onShow(resuming = false) {
  // PrimeVue otherwise hides on anchor scroll or desktop resize, even when
  // dismissable=false. Keep drafts mounted and reposition with the anchor.
  const overlay = panel.value as (InstanceType<typeof PrimePopover> & {
    unbindScrollListener(): void
    unbindResizeListener(): void
  }) | null
  overlay?.unbindScrollListener()
  overlay?.unbindResizeListener()
  alignWithinViewport()
  window.addEventListener('resize', alignWithinViewport)
  document.addEventListener('scroll', onScroll, true)
  const root = document.getElementById(id.value)
  const owner = trigger?.closest<HTMLElement>('[data-pc-name="popover"]')
  if (root && owner?.id) root.dataset.overlayOwner = owner.id
  else if (root) delete root.dataset.overlayOwner
  const first = root?.querySelector<HTMLElement>('[autofocus]')
    ?? root?.querySelector<HTMLElement>('input:not(:disabled):not([type="hidden"]), textarea:not(:disabled)')
    ?? root?.querySelector<HTMLElement>('button:not(:disabled), a[href], [tabindex="0"]')
  first?.focus({ preventScroll: true })
  returnFocus = true
  document.addEventListener('focusin', trackFocus, true)
  document.addEventListener('pointerdown', onPointerDown, true)
  // A simple action menu must also dismiss when an outside button uses
  // click.stop (copy controls and other contextual triggers do this).
  // Forms retain the existing bubbling listener and nested-picker protection.
  document.addEventListener('click', onOutsideClick, props.interaction === 'menu')
  // Async details and expandable fields can resize a panel after it opens.
  // Reposition at its anchor so growing content stays within the viewport.
  resizeObserver = new ResizeObserver(() => {
    cancelAnimationFrame(alignmentFrame)
    alignmentFrame = requestAnimationFrame(alignWithinViewport)
  })
  if (root) resizeObserver.observe(root)
  if (!resuming) emit('show')
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
  side = undefined
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
      alignWithinViewport()
    })
  })
}
function alignWithinViewport() {
  const root = document.getElementById(id.value)
  if (!root || !trigger || !workspaceActive.value) return
  const anchor = trigger.getBoundingClientRect()
  // Measure untransformed dimensions: entry animation scales the bounding rect.
  root.style.maxWidth = ''
  root.style.maxHeight = ''
  const placement = popoverPlacement(anchor, { width: window.innerWidth, height: window.innerHeight }, {
    width: root.offsetWidth, height: root.offsetHeight,
  }, side)
  side = placement.side
  root.style.maxWidth = `${Math.min(root.offsetWidth, placement.maxWidth)}px`
  root.style.maxHeight = `${placement.maxHeight}px`
  // Width constraints can wrap content and change height. Measure once more.
  const final = popoverPlacement(anchor, { width: window.innerWidth, height: window.innerHeight }, {
    width: root.offsetWidth, height: root.offsetHeight,
  }, side)
  root.style.left = `${final.left + window.scrollX}px`
  root.style.top = `${final.top + window.scrollY}px`
  root.style.marginTop = '0'
  root.style.transformOrigin = final.origin
}
defineExpose({ toggle, showAt, hide })
</script>

<template>
  <PrimePopover ref="panel" :id="id" :dismissable="false" :close-on-escape="false" unstyled :pt="theme" :pt-options="{mergeProps: ptViewMerge}"
    @show="onShow()" @hide="onHide" @keydown.capture="onKeydown">
    <slot />
  </PrimePopover>
</template>
