<script setup lang="ts">
import { nextTick, onBeforeUnmount, ref, useId, watch } from 'vue'
import { claimTooltip, releaseTooltip } from '../utils/tooltipCoordinator'

const props = withDefaults(defineProps<{
  target?: HTMLElement | SVGElement | null
  visible?: boolean
  point?: { x: number; y: number }
  enabled?: boolean
  shouldShow?: () => boolean
  hoverDelay?: number
  width?: number
  matchAnchorWidth?: boolean
}>(), { visible: undefined, enabled: true, hoverDelay: 320, matchAnchorWidth: false })
const emit = defineEmits<{ dismiss: [] }>()
const id = useId()
const owner = {}
const anchor = ref<HTMLElement>()
const bubble = ref<HTMLElement>()
const open = ref(false)
const position = ref<Record<string, string>>({ visibility: 'hidden' })
let timer: ReturnType<typeof setTimeout> | undefined
let generation = 0
let overAnchor = false
let overBubble = false
let keyboardFocus = false
let pointerType = ''
let pointerPosition: { x: number; y: number } | undefined
let suppressHover = false
const element = () => props.target ?? anchor.value
function cancelTimer() { clearTimeout(timer) }
function close() {
  cancelTimer()
  generation++
  open.value = false
  overBubble = false
  releaseTooltip(owner)
  emit('dismiss')
}
async function show() {
  cancelTimer()
  await nextTick()
  if (!props.enabled || props.visible === false || props.shouldShow?.() === false) return
  if (!element() && !props.point) return
  claimTooltip(owner, close)
  if (open.value && !props.point) return
  const current = ++generation
  const box = props.point
    ? { left: props.point.x + 14, top: props.point.y, bottom: props.point.y + 8, width: 0 }
    : element()!.getBoundingClientRect()
  const available = Math.max(0, innerWidth - 16)
  const fixedWidth = props.width !== undefined || props.matchAnchorWidth
  const width = Math.min(Math.max(props.width ?? 0, props.matchAnchorWidth ? box.width : 0), available)
  const sizing = {
    width: fixedWidth ? `${width}px` : 'max-content',
    maxWidth: `${fixedWidth ? available : Math.min(280, available)}px`,
  }
  if (!open.value) position.value = { visibility: 'hidden', ...sizing }
  open.value = true
  await nextTick()
  if (!open.value || current !== generation || !bubble.value) return
  const height = bubble.value.offsetHeight
  const measuredWidth = bubble.value.offsetWidth
  const above = box.top >= height + 14
  position.value = {
    ...sizing,
    left: `${Math.max(8, Math.min(box.left, innerWidth - measuredWidth - 8))}px`,
    top: `${Math.max(8, Math.min(above ? box.top - height - 6 : box.bottom + 6, innerHeight - height - 8))}px`,
    '--tooltip-offset': above ? '4px' : '-4px',
    transformOrigin: above ? 'bottom left' : 'top left',
  }
}
function queueShow() {
  if (props.shouldShow?.() === false || !props.enabled) return
  claimTooltip(owner, close)
  cancelTimer()
  timer = setTimeout(() => {
    if (!suppressHover && (overAnchor || props.visible)) void show()
  }, props.hoverDelay)
}
function enter() {
  overAnchor = true
  suppressHover = false
  pointerPosition = undefined
  queueShow()
}
function move(event: PointerEvent) {
  if (!overAnchor || open.value || suppressHover || event.pointerType === 'touch') return
  if (!pointerPosition || Math.hypot(event.clientX - pointerPosition.x, event.clientY - pointerPosition.y) > 3) {
    pointerPosition = { x: event.clientX, y: event.clientY }
    queueShow()
  }
}
function scheduleLeave() {
  cancelTimer()
  timer = setTimeout(() => {
    if (!overAnchor && !overBubble && !keyboardFocus) close()
  }, 100)
}
function leave() { overAnchor = false; scheduleLeave() }
function enterBubble() { overBubble = true; cancelTimer() }
function leaveBubble() { overBubble = false; scheduleLeave() }
function focusIn(event: FocusEvent) {
  const focused = event.target as Element
  keyboardFocus = pointerType !== 'mouse' && pointerType !== 'touch' && focused.matches(':focus-visible')
  if (keyboardFocus) void show()
}
function focusOut() { keyboardFocus = false; pointerType = ''; scheduleLeave() }
function pointerDown(event: PointerEvent) {
  pointerType = event.pointerType || 'mouse'
  keyboardFocus = false
  if (pointerType === 'mouse') { suppressHover = true; close() }
}
function click() {
  // Explicit help buttons call the slot's show() themselves. Other mouse
  // clicks perform their action without leaving a tooltip over the result.
  if (pointerType === 'touch' && !element()?.matches('button, a')) void show()
}
function outside(event: PointerEvent) {
  const target = event.target as Node
  if (!element()?.contains(target) && !bubble.value?.contains(target)) close()
}
function escape(event: KeyboardEvent) {
  if (event.key !== 'Escape') return
  event.preventDefault()
  event.stopPropagation()
  suppressHover = true
  close()
}
function listeners(remove: boolean) {
  if (remove) {
    document.removeEventListener('pointerdown', outside, true)
    document.removeEventListener('keydown', escape, true)
    window.removeEventListener('scroll', close, true)
    window.removeEventListener('resize', close)
  } else {
    document.addEventListener('pointerdown', outside, true)
    document.addEventListener('keydown', escape, true)
    window.addEventListener('scroll', close, true)
    window.addEventListener('resize', close)
  }
}
watch(() => [props.visible, props.point] as const, ([visible]) => {
  if (visible) {
    suppressHover = false
    if (open.value) void show()
    else queueShow()
  }
  else if (visible === false) close()
}, { flush: 'post', immediate: true })
watch(() => props.target, (target, _, onCleanup) => {
  if (!target) return
  const bindings = {
    mouseenter: enter, mouseleave: leave, focusin: focusIn, focusout: focusOut,
    pointerdown: pointerDown, pointermove: move, click,
  }
  for (const [event, listener] of Object.entries(bindings)) target.addEventListener(event, listener as EventListener)
  onCleanup(() => {
    for (const [event, listener] of Object.entries(bindings)) target.removeEventListener(event, listener as EventListener)
  })
}, { flush: 'post', immediate: true })
watch(open, value => {
  listeners(!value)
  const target = props.target
  if (target) {
    const ids = new Set((target.getAttribute('aria-describedby') ?? '').split(/\s+/).filter(Boolean))
    value ? ids.add(id) : ids.delete(id)
    if (ids.size) target.setAttribute('aria-describedby', [...ids].join(' '))
    else target.removeAttribute('aria-describedby')
  }
}, { flush: 'sync' })
onBeforeUnmount(() => { close(); listeners(true) })
</script>
<template>
  <div v-if="$slots.default" ref="anchor" class="tooltip-anchor" @mouseenter="enter" @mouseleave="leave"
    @focusin="focusIn" @focusout="focusOut" @pointerdown="pointerDown" @pointermove="move" @click="click">
    <slot :described-by="open ? id : undefined" :show="show" />
  </div>
  <Teleport to="body">
    <Transition name="field-tooltip">
      <div v-if="open" :id="id" ref="bubble" role="tooltip" class="field-tooltip" :class="{ 'field-tooltip-pointer': point }"
        :style="position" @mouseenter="enterBubble" @mouseleave="leaveBubble">
        <slot name="content" />
      </div>
    </Transition>
  </Teleport>
</template>
<style scoped>
.tooltip-anchor { min-width: 0; }
.field-tooltip { position: fixed; z-index: 10000; max-height: calc(100dvh - 16px); overflow: auto; padding: 6px 10px; border: 1px solid var(--color-border); border-radius: var(--radius-control); background: var(--color-bg-secondary); color: var(--color-text-primary); box-shadow: 0 4px 18px rgb(0 0 0 / 12%); font-size: 12px; line-height: 1.5; overflow-wrap: anywhere; }
.field-tooltip-pointer { pointer-events: none; }
.field-tooltip-enter-active { transition: opacity 160ms var(--motion-enter), transform 160ms var(--motion-enter); }
.field-tooltip-leave-active { transition: opacity 160ms var(--motion-exit), transform 160ms var(--motion-exit); pointer-events: none; }
.field-tooltip-enter-from, .field-tooltip-leave-to { opacity: 0; transform: translateY(var(--tooltip-offset, 4px)) scale(.98); }
@media (prefers-reduced-motion: reduce) { .field-tooltip-enter-active, .field-tooltip-leave-active { transition: none; } .field-tooltip-enter-from, .field-tooltip-leave-to { transform: none; } }
</style>
