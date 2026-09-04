/**
 * Zoom for the route-flow diagram.
 *
 * WHY THE DIAGRAM NEEDS ONE
 * ─────────────────────────
 * The canvas is a fixed 1290px (`flowLayout.ts`). Docked, the SVG carries
 * `max-w-full`, so in a narrower card the browser SHRINKS the whole picture —
 * an 11px rule label renders at 6px and the card still looks like it fits.
 * Full-window, `fit` squeezes the whole ladder into the viewport height, which
 * on a 28-rule config is also below 1:1. Both are the right DEFAULT (the shape
 * of the config is the first thing to see) and both are unreadable when the
 * question becomes "what does rule #17 actually say".
 *
 * SCALE, NOT A viewBox TRANSFORM
 * ──────────────────────────────
 * Zooming here means rendering the same `viewBox` at a larger pixel size and
 * letting the wrapper scroll. Panning is then the browser's: two-finger
 * scroll, shift+wheel, scrollbars, arrow keys — all of it already correct,
 * including at the edges, where a hand-rolled viewBox transform tends to
 * clip `<title>` hit areas. SVG scales without resampling, so the labels get
 * genuinely sharper rather than magnified.
 *
 * `scale === null` means "leave the default alone" — fit-to-width docked,
 * fit-to-height expanded. The first zoom action commits to a number, and
 * `reset()` hands it back. Nothing about the default view changes for an
 * operator who never touches the control.
 *
 * PANNING IS THE SAME SCROLL
 * ──────────────────────────
 * Ctrl/⌘+drag moves the wrapper's `scrollLeft`/`scrollTop` — exactly what the
 * scrollbars move. Nothing is translated, so the drag, the scrollbars, the
 * wheel and the keyboard all address one position and cannot disagree; the
 * scrollbar thumbs track the drag because they ARE the state being dragged.
 * The modifier is shared with zoom on purpose: one held key turns the diagram
 * into a canvas, and an unmodified drag still selects and clicks as before.
 */
import { computed, onBeforeUnmount, ref } from 'vue'

/**
 * The zoom ladder.
 *
 * Fixed steps rather than a multiplier, so repeated clicks land on the same
 * values in both directions and 100% is always reachable — with `scale *= 1.2`
 * it is not, and "back to actual size" becomes a guess.
 */
export const ZOOM_STEPS = [0.5, 0.67, 0.8, 1, 1.25, 1.5, 2, 2.5, 3] as const

export const MIN_ZOOM = ZOOM_STEPS[0]
export const MAX_ZOOM = ZOOM_STEPS[ZOOM_STEPS.length - 1]!

/** Guards float comparison against the ladder — 0.8 measured is never exactly 0.8. */
const EPSILON = 0.001

/** The first ladder step above `current`, or `null` at the top. */
export function stepUp(current: number): number | null {
  return ZOOM_STEPS.find((step) => step > current + EPSILON) ?? null
}

/** The last ladder step below `current`, or `null` at the bottom. */
export function stepDown(current: number): number | null {
  const below = ZOOM_STEPS.filter((step) => step < current - EPSILON)
  return below.length > 0 ? below[below.length - 1]! : null
}

export function clampZoom(value: number): number {
  return Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, value))
}

export interface ScrollAnchor {
  /** The scroll container's viewport box, in client coordinates. */
  rect: { left: number; top: number }
  /** Where the pointer is, in client coordinates. */
  cursor: { x: number; y: number }
  /** The container's scroll offsets before the scale change. */
  scroll: { left: number; top: number }
}

/**
 * Scroll offsets that keep the content point under the cursor under the cursor.
 *
 * Zooming about the container's top-left instead is what makes wheel-zoom feel
 * like the picture is running away: the operator points at rule #17, zooms, and
 * #17 is off-screen. Pure so the arithmetic is testable without a DOM.
 *
 * Only exact while the content overflows the container in that axis. Below
 * that the content is centred (`mx-auto`) and there is no scroll to speak of,
 * so the clamp at 0 is the whole correction needed.
 */
export function anchoredScroll(anchor: ScrollAnchor, from: number, to: number): { left: number; top: number } {
  if (from <= 0) return { left: anchor.scroll.left, top: anchor.scroll.top }
  const offsetX = anchor.cursor.x - anchor.rect.left
  const offsetY = anchor.cursor.y - anchor.rect.top
  const ratio = to / from
  return {
    left: Math.max(0, (anchor.scroll.left + offsetX) * ratio - offsetX),
    top: Math.max(0, (anchor.scroll.top + offsetY) * ratio - offsetY),
  }
}

export interface PanOrigin {
  /** Where the drag started, in client coordinates. */
  pointer: { x: number; y: number }
  /** The container's scroll offsets when the drag started. */
  scroll: { left: number; top: number }
}

/**
 * Scroll offsets for a pan in progress.
 *
 * Grab-the-canvas, not drag-the-scrollbar: moving the pointer right moves the
 * picture right, which means scrolling LEFT. That is the direction every map,
 * PDF viewer and design tool uses, and inverting it makes the diagram feel
 * like it is fleeing the cursor.
 *
 * Recomputed from the ORIGIN on every move rather than accumulated per delta.
 * Accumulating drifts against the DOM's own clamping: drag past an edge and
 * back, and the accumulated total is ahead of where the element actually is by
 * however far it refused to go, leaving a dead zone before the picture moves
 * again.
 */
export function pannedScroll(origin: PanOrigin, cursor: { x: number; y: number }): { left: number; top: number } {
  return {
    left: Math.max(0, origin.scroll.left - (cursor.x - origin.pointer.x)),
    top: Math.max(0, origin.scroll.top - (cursor.y - origin.pointer.y)),
  }
}

export function useDiagramZoom() {
  /** null = the component's own sizing (fit-to-width, or fit-to-height expanded). */
  const scale = ref<number | null>(null)

  const viewport = ref<HTMLElement | null>(null)

  const zoomed = computed(() => scale.value !== null)
  const percent = computed(() => (scale.value === null ? null : Math.round(scale.value * 100)))

  /**
   * What the diagram is rendering at right now, measured rather than assumed.
   *
   * With `scale === null` the factor is the browser's, decided by the card's
   * width or the window's height, and stepping up from a guess of 1 would jump
   * a shrunk diagram to 125% on the first click. The SVG knows both numbers —
   * its `viewBox` width and its rendered width — so the DOM is asked.
   */
  const measuredScale = (): number => {
    const svg = viewport.value?.querySelector('svg')
    if (!svg) return scale.value ?? 1
    const base = svg.viewBox.baseVal.width
    const rendered = svg.getBoundingClientRect().width
    if (base <= 0 || rendered <= 0) return scale.value ?? 1
    return rendered / base
  }

  const effectiveScale = (): number => scale.value ?? measuredScale()

  const apply = (next: number, anchor?: ScrollAnchor) => {
    const from = effectiveScale()
    const to = clampZoom(next)
    scale.value = to
    const el = viewport.value
    if (!anchor || !el) return
    // After Vue paints the new width — the scroll range does not exist until
    // the element is wider, and setting scrollLeft past the current range is
    // silently clamped to it.
    requestAnimationFrame(() => {
      const offsets = anchoredScroll(anchor, from, to)
      el.scrollLeft = offsets.left
      el.scrollTop = offsets.top
    })
  }

  const zoomIn = (anchor?: ScrollAnchor) => {
    const next = stepUp(effectiveScale())
    if (next !== null) apply(next, anchor)
  }

  const zoomOut = (anchor?: ScrollAnchor) => {
    const next = stepDown(effectiveScale())
    if (next !== null) apply(next, anchor)
  }

  /** Back to the component's own sizing — not to 100%, which is a different thing. */
  const reset = () => {
    scale.value = null
    const el = viewport.value
    if (el) {
      el.scrollLeft = 0
      el.scrollTop = 0
    }
  }

  const actualSize = () => apply(1)

  const canZoomIn = computed(() => scale.value === null || scale.value < MAX_ZOOM - EPSILON)
  const canZoomOut = computed(() => scale.value === null || scale.value > MIN_ZOOM + EPSILON)

  const anchorFrom = (event: { clientX: number; clientY: number }, el: HTMLElement): ScrollAnchor => {
    const rect = el.getBoundingClientRect()
    return {
      rect: { left: rect.left, top: rect.top },
      cursor: { x: event.clientX, y: event.clientY },
      scroll: { left: el.scrollLeft, top: el.scrollTop },
    }
  }

  /**
   * Ctrl/⌘+wheel, which is also what a trackpad pinch arrives as — so pinch
   * works without a gesture handler.
   *
   * `preventDefault` is the entire point and it only works on a NON-passive
   * listener, which is why this is `addEventListener` rather than an `@wheel`
   * binding: browsers default wheel listeners to passive, and without the
   * cancel the browser zooms the whole page instead of the diagram.
   */
  const onWheel = (event: WheelEvent) => {
    if (!event.ctrlKey && !event.metaKey) return
    const el = viewport.value
    if (!el) return
    event.preventDefault()
    const anchor = anchorFrom(event, el)
    if (event.deltaY < 0) zoomIn(anchor)
    else if (event.deltaY > 0) zoomOut(anchor)
  }

  /* ── Pan ────────────────────────────────────────────────────────────────── */

  /** A drag is in progress. */
  const panning = ref(false)
  /** The modifier is down, so a drag WOULD pan — drives the grab cursor. */
  const panReady = ref(false)

  let panOrigin: PanOrigin | null = null
  let panPointerId: number | null = null

  const onPointerDown = (event: PointerEvent) => {
    if (event.button !== 0 || !event.isPrimary) return
    if (!event.ctrlKey && !event.metaKey) return
    const el = viewport.value
    if (!el) return

    // Stops the browser starting its own selection or image drag from under us.
    event.preventDefault()
    panOrigin = {
      pointer: { x: event.clientX, y: event.clientY },
      scroll: { left: el.scrollLeft, top: el.scrollTop },
    }
    panPointerId = event.pointerId
    panning.value = true
    // Capture, so a drag that leaves the card — easy to do at 300% — keeps
    // panning instead of stopping at the edge and stranding the picture.
    //
    // Optional, not load-bearing: the spec lets this throw if the pointer is
    // no longer active, and the pan is already armed above, so a failure costs
    // the off-element part of the drag rather than the gesture.
    try {
      el.setPointerCapture(event.pointerId)
    } catch {
      panPointerId = null
    }
  }

  const onPointerMove = (event: PointerEvent) => {
    if (!panning.value || panOrigin === null) return
    const el = viewport.value
    if (!el) return
    event.preventDefault()
    const offsets = pannedScroll(panOrigin, { x: event.clientX, y: event.clientY })
    el.scrollLeft = offsets.left
    el.scrollTop = offsets.top
  }

  /**
   * Ends a pan. Bound to pointerup AND pointercancel — the browser or OS taking
   * the pointer over (a system gesture, a drag hand-off) delivers a cancel and
   * never an up, and treating only `up` as the end leaves the diagram stuck to
   * a cursor with no button held.
   *
   * Note the modifier is deliberately NOT re-checked while dragging: it arms
   * the gesture, and once the button is down the drag belongs to the pointer.
   * Requiring the key to stay held would abandon the pan mid-move every time a
   * finger slipped.
   */
  const endPan = (event: PointerEvent) => {
    if (!panning.value) return
    panning.value = false
    panOrigin = null
    const el = viewport.value
    if (el && panPointerId !== null && el.hasPointerCapture(panPointerId)) {
      el.releasePointerCapture(panPointerId)
    }
    panPointerId = null
    void event
  }

  /**
   * On macOS, ctrl+click IS a right-click, so the pan gesture also raises a
   * context menu over the diagram. Suppressed only while the modifier is
   * actually engaged, so an ordinary right-click still works.
   */
  const onContextMenu = (event: MouseEvent) => {
    if (event.ctrlKey || event.metaKey || panning.value) event.preventDefault()
  }

  /**
   * Tracks the modifier so the cursor can offer the grab BEFORE the button
   * goes down — otherwise the gesture is invisible until someone tries it.
   *
   * Read off the event's own modifier state rather than by matching key names,
   * which gets Ctrl, Meta and either-side variants right for free. `blur`
   * resets it because switching apps with ⌘-Tab consumes the keyup, and a
   * stuck grab cursor over a diagram that no longer pans is worse than no
   * cursor hint at all.
   */
  const onModifierChange = (event: KeyboardEvent) => {
    panReady.value = event.ctrlKey || event.metaKey
  }

  const onWindowBlur = () => {
    panReady.value = false
  }

  const onKeydown = (event: KeyboardEvent) => {
    if (event.ctrlKey || event.metaKey || event.altKey) return
    if (event.key === '+' || event.key === '=') {
      event.preventDefault()
      zoomIn()
    } else if (event.key === '-' || event.key === '_') {
      event.preventDefault()
      zoomOut()
    } else if (event.key === '0') {
      event.preventDefault()
      reset()
    }
  }

  const detach = (el: HTMLElement) => {
    el.removeEventListener('wheel', onWheel)
    el.removeEventListener('keydown', onKeydown)
    el.removeEventListener('pointerdown', onPointerDown)
    el.removeEventListener('pointermove', onPointerMove)
    el.removeEventListener('pointerup', endPan)
    el.removeEventListener('pointercancel', endPan)
    el.removeEventListener('contextmenu', onContextMenu)
    window.removeEventListener('keydown', onModifierChange)
    window.removeEventListener('keyup', onModifierChange)
    window.removeEventListener('blur', onWindowBlur)
    panning.value = false
    panReady.value = false
  }

  /**
   * Template ref callback for the scrolling wrapper.
   *
   * A callback rather than a plain ref because the wrapper is re-parented by
   * the card's `<Teleport :disabled>` between docked and full-window: this
   * fires with the element on both sides of the move, so the listeners follow
   * it instead of being left on a detached node.
   *
   * `unknown` because Vue types a ref callback as receiving an element OR a
   * component instance; the `instanceof` below is the narrowing, and it is the
   * check that would be needed anyway.
   */
  const bindViewport = (el: unknown) => {
    const previous = viewport.value
    if (previous && previous !== el) detach(previous)
    const next = el instanceof HTMLElement ? el : null
    viewport.value = next
    if (next && next !== previous) {
      next.addEventListener('wheel', onWheel, { passive: false })
      next.addEventListener('keydown', onKeydown)
      // Non-passive for the same reason as the wheel: both preventDefault, and
      // a passive listener silently cannot.
      next.addEventListener('pointerdown', onPointerDown, { passive: false })
      next.addEventListener('pointermove', onPointerMove, { passive: false })
      next.addEventListener('pointerup', endPan)
      next.addEventListener('pointercancel', endPan)
      next.addEventListener('contextmenu', onContextMenu)
      window.addEventListener('keydown', onModifierChange)
      window.addEventListener('keyup', onModifierChange)
      window.addEventListener('blur', onWindowBlur)
    }
  }

  onBeforeUnmount(() => {
    if (viewport.value) detach(viewport.value)
  })

  return {
    scale,
    zoomed,
    percent,
    canZoomIn,
    canZoomOut,
    zoomIn,
    zoomOut,
    reset,
    actualSize,
    panning,
    panReady,
    bindViewport,
  }
}
