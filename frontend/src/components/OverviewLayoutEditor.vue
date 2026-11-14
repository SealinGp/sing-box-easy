<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bars3Icon, XMarkIcon } from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import OverviewCardSlot from './OverviewCardSlot.vue'
import { moveCard, overviewSlots, type SortCard } from '../utils/overviewSort'
import { dragScrollSpeed } from '../utils/dragAutoScroll'

const props = defineProps<{ cards: SortCard[]; hidden: string[]; columns: number }>()
const emit = defineEmits<{ move: [from: number, to: number]; hide: [id: string]; add: [slotId: string, id: string] }>()
const { t } = useI18n()
const root = ref<HTMLElement>()
const source = ref<number | null>(null)
const target = ref<number | null>(null)
const pointer = ref<{ x: number; y: number } | null>(null)
const announcement = ref('')
const available = computed(() => props.cards.filter(card => props.hidden.includes(card.id)))
const baseSlots = computed(() => overviewSlots(props.cards, props.columns))
const previewCards = computed(() => source.value !== null && target.value !== null ? moveCard(props.cards, source.value, target.value) : props.cards)
const slots = computed(() => overviewSlots(previewCards.value, props.columns))
const carried = computed(() => source.value === null ? null : props.cards[source.value])
const rows = computed(() => Math.max(1, ...slots.value.map(s => s.row), ...baseSlots.value.map(s => s.row)))
const cells = computed(() => Array.from({ length: rows.value * props.columns }, (_, i) => ({ row: Math.floor(i / props.columns) + 1, column: i % props.columns + 1 })))
let capture: HTMLElement | undefined
let pointerId: number | undefined
let frame = 0
let startPoint = { x: 0, y: 0 }
let armed: number | null = null
let originRects: { index: number; x: number; y: number }[] = []
let scrollParent: HTMLElement | undefined
let initialScroll = 0
let moved = false

function clear() {
  cancelAnimationFrame(frame)
  armed = null
  source.value = target.value = null
  pointer.value = null
  if (capture && pointerId !== undefined && capture.hasPointerCapture(pointerId)) capture.releasePointerCapture(pointerId)
  capture = undefined
  pointerId = undefined
}
function hitTest() {
  if (!pointer.value || !root.value) return
  const box = root.value.getBoundingClientRect()
  if (pointer.value.x < box.left || pointer.value.x > box.right || pointer.value.y < box.top || pointer.value.y > box.bottom) { target.value = null; return }
  const scrollDelta = (scrollParent?.scrollTop ?? 0) - initialScroll
  const nearest = [...originRects].sort((a, b) => Math.hypot(pointer.value!.x - a.x, pointer.value!.y - a.y + scrollDelta) - Math.hypot(pointer.value!.x - b.x, pointer.value!.y - b.y + scrollDelta))[0]
  target.value = nearest?.index ?? null
}
function tick() {
  if (!pointer.value) return
  if (scrollParent) {
    const box = scrollParent === document.scrollingElement ? { top: 0, bottom: innerHeight } : scrollParent.getBoundingClientRect()
    const speed = dragScrollSpeed(pointer.value.y, Math.max(0, box.top), Math.min(innerHeight, box.bottom))
    if (speed) scrollParent.scrollBy({ top: speed / 60, behavior: 'instant' })
  }
  hitTest()
  frame = requestAnimationFrame(tick)
}
function down(index: number, event: PointerEvent) {
  if (event.button !== 0 || armed !== null) return
  capture = root.value!
  pointerId = event.pointerId
  armed = index
  moved = false
  startPoint = { x: event.clientX, y: event.clientY }
  capture.setPointerCapture(event.pointerId)
  // Freeze hit regions for this gesture: animated preview cards must not
  // chase the pointer or cause the destination to oscillate.
  originRects = [...root.value!.querySelectorAll<HTMLElement>('[data-sort-card]')].map(el => {
    const box = el.getBoundingClientRect()
    return { index: Number(el.dataset.sortCard), x: box.left + box.width / 2, y: box.top + box.height / 2 }
  })
  scrollParent = root.value?.parentElement ?? undefined
  while (scrollParent && !/(auto|scroll)/.test(getComputedStyle(scrollParent).overflowY)) scrollParent = scrollParent.parentElement ?? undefined
  scrollParent ??= document.scrollingElement as HTMLElement
  initialScroll = scrollParent.scrollTop
}
function move(event: PointerEvent) {
  if (armed === null || event.pointerId !== pointerId) return
  if (!moved && Math.hypot(event.clientX - startPoint.x, event.clientY - startPoint.y) < 5) return
  event.preventDefault()
  pointer.value = { x: event.clientX, y: event.clientY }
  if (!moved) { moved = true; source.value = armed; target.value = armed; frame = requestAnimationFrame(tick) }
  hitTest()
}
function up(event: PointerEvent) {
  if (event.pointerId !== pointerId) return
  const from = source.value, to = target.value
  clear()
  if (from !== null && to !== null) commit(from, to)
}
function commit(from: number, to: number) {
  const id = props.cards[from]?.id
  if (!id || to < 0 || to >= props.cards.length || from === to) return
  announcement.value = t('overview.layout.moved', { name: t(props.cards[from]!.title), position: to + 1 })
  emit('move', from, to)
  void nextTick(() => root.value?.querySelector<HTMLElement>(`[data-handle="${id}"]`)?.focus({ preventScroll: true }))
}
function remove(id: string) {
  clear()
  emit('hide', id)
  void nextTick(() => root.value?.querySelector<HTMLElement>(`[data-card-id="${id}"] [data-add-card]`)?.focus())
}
function add(slotId: string, id: string) {
  emit('add', slotId, id)
  void nextTick(() => root.value?.querySelector<HTMLElement>(`[data-handle="${id}"]`)?.focus())
}
function key(index: number, event: KeyboardEvent) {
  if (event.key === 'Escape') { clear(); event.preventDefault(); return }
  const delta = ({ ArrowLeft: -1, ArrowRight: 1, ArrowUp: -props.columns, ArrowDown: props.columns } as Record<string, number>)[event.key]
  if (delta !== undefined) { event.preventDefault(); commit(index, index + delta) }
}
function escape(event: KeyboardEvent) { if (event.key === 'Escape') clear() }
window.addEventListener('keydown', escape)
window.addEventListener('blur', clear)
watch(() => props.columns, clear)
onBeforeUnmount(() => { clear(); window.removeEventListener('keydown', escape); window.removeEventListener('blur', clear) })
</script>

<template>
  <div class="layout-editor">
    <p class="text-sm text-gray-500 dark:text-gray-400 mb-3">{{ t('overview.layout.capacity', { count: columns }) }}</p>
    <div ref="root" class="sort-grid" @pointermove="move" @pointerup="up" @pointercancel="clear" @lostpointercapture="clear" :style="{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`, gridTemplateRows: `repeat(${rows}, 132px)` }" :aria-label="t('overview.layout.arranging')">
      <div v-for="(cell, index) in cells" :key="`cell-${index}`" class="sort-cell" :style="{ gridColumn: cell.column, gridRow: cell.row }" aria-hidden="true"><span>{{ cell.column }}</span></div>
      <TransitionGroup name="sort-card">
        <div v-for="slot in slots" :key="slot.id" class="sort-card" :data-sort-card="props.cards.findIndex(c => c.id === slot.id)" :data-card-id="slot.id"
          :class="{ 'is-landing': carried?.id === slot.id, 'is-empty': hidden.includes(slot.id) }" :style="{ gridColumn: `${slot.column} / span ${slot.span}`, gridRow: slot.row }">
          <OverviewCardSlot v-if="hidden.includes(slot.id)" :cards="available" @select="id => add(slot.id, id)" />
          <div v-else-if="carried?.id === slot.id" class="landing-label">{{ t('overview.layout.dropHere') }}<strong>{{ t(carried.title) }}</strong></div>
          <div v-if="!hidden.includes(slot.id)" class="sort-content" :style="{ visibility: carried?.id === slot.id ? 'hidden' : undefined }">
            <button type="button" class="remove-card icon-action" :disabled="source !== null" :aria-label="t('overview.layout.remove', { name: t(previewCards[slot.index]!.title) })" @click="remove(slot.id)">
              <XMarkIcon class="h-4 w-4" />
            </button>
            <button type="button" class="sort-handle" :data-handle="slot.id" :aria-label="t('overview.layout.handle', { name: t(previewCards[slot.index]!.title) })"
              @pointerdown="down(slot.index, $event)" @keydown="key(slot.index, $event)" @dragstart.prevent>
              <Bars3Icon class="h-5 w-5 shrink-0" /><span>{{ t(previewCards[slot.index]!.title) }}</span>
            </button>
            <div class="sort-details"><span>{{ slot.span === columns && slot.id === 'route-topology' ? t('overview.layout.fullWidth') : t('overview.layout.position', { row: slot.row, column: slot.column }) }}</span><div class="flex">
              <Button variant="ghost" size="sm" :disabled="slot.index === 0 || source !== null" :aria-label="t('overview.layout.earlier', { name: t(previewCards[slot.index]!.title) })" @click="commit(slot.index, slot.index - 1)">←</Button>
              <Button variant="ghost" size="sm" :disabled="slot.index === cards.length - 1 || source !== null" :aria-label="t('overview.layout.later', { name: t(previewCards[slot.index]!.title) })" @click="commit(slot.index, slot.index + 1)">→</Button>
            </div></div>
          </div>
        </div>
      </TransitionGroup>
    </div>
    <p class="sr-only" role="status">{{ announcement }}</p>
    <Teleport to="body"><div v-if="carried && pointer" class="sort-ghost" :style="{ left: `${pointer.x + 16}px`, top: `${pointer.y + 16}px` }" aria-hidden="true"><Bars3Icon class="h-5 w-5" />{{ t(carried.title) }}</div></Teleport>
  </div>
</template>

<style scoped>
.sort-grid { display: grid; gap: 16px; position: relative; }
.sort-cell { border: 1px dashed var(--color-border, #a8a29e); border-radius: var(--radius-surface); background: color-mix(in srgb, var(--color-primary) 3%, transparent); display: flex; align-items: center; justify-content: center; color: var(--color-text-secondary); opacity: .55; }
.sort-card { z-index: 1; min-width: 0; border: 1px solid var(--glass-border-muted); border-radius: var(--radius-surface); background: var(--glass-bg-strong); box-shadow: var(--shadow-surface); padding: 12px; position: relative; display: flex; flex-direction: column; justify-content: space-between; }
.remove-card { position: absolute; right: 8px; top: 8px; }
.sort-card.is-empty { border-style: dashed; box-shadow: none; background: transparent; padding: 0; }
.sort-content { display: flex; flex: 1; flex-direction: column; justify-content: space-between; }
.sort-handle { padding-right: 30px; display: flex; align-items: center; gap: 10px; width: 100%; flex: 1; text-align: left; font-weight: 600; cursor: grab; touch-action: none; user-select: none; border-radius: 5px; }
.sort-handle:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 3px; }
.sort-details { display: flex; justify-content: space-between; align-items: center; font-size: 12px; color: var(--color-text-secondary); }
.is-landing { border: 2px dashed var(--color-primary); background: color-mix(in srgb, var(--color-primary) 13%, transparent); box-shadow: inset 0 2px 12px color-mix(in srgb, var(--color-primary) 15%, transparent), 0 4px 18px #00000012; pointer-events: none; }
.landing-label { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; text-align: center; font-size: 12px; color: var(--color-text-secondary); }
.landing-label strong { display: block; margin-top: 6px; color: var(--color-text-primary); }
.sort-ghost { position: fixed; z-index: 9999; pointer-events: none; display: flex; align-items: center; gap: 12px; padding: 20px; max-width: 240px; border: 1px solid var(--glass-border-muted); border-radius: var(--radius-surface); background: var(--glass-bg-strong); box-shadow: 0 16px 40px #00000035; transform: rotate(2deg); font-weight: 600; }
@media (prefers-reduced-motion: no-preference) { .sort-card-move { transition: transform 180ms ease; } }
</style>
