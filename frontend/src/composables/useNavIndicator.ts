import { onBeforeUnmount, onMounted, type Ref } from 'vue'

/**
 * Drives a single "active item" pill that slides between nav entries — the web
 * analogue of Liquid Glass's `glassEffectID` morph, where one glass shape moves
 * from A to B instead of two shapes cross-fading independently.
 *
 * Why a shared element rather than per-item backgrounds:
 *
 *  1. `linear-gradient()` is NOT an interpolatable CSS value. A per-item
 *     `background: linear-gradient(...)` -> `none` cannot transition at all; it
 *     snaps. Moving the gradient onto one persistent element and animating its
 *     `transform` sidesteps that entirely.
 *  2. `transform` and `opacity` are composited on the GPU, so the slide keeps
 *     running while the main thread is busy mounting the incoming route
 *     component. Background/size transitions need a main-thread paint every
 *     frame and therefore stutter at exactly the moment of a route change —
 *     which is what "laggy when switching" actually is.
 *
 * Positions are written straight to the DOM node rather than through a reactive
 * style binding: the ResizeObserver fires on every frame of the submenu
 * expand/collapse, and routing each of those through Vue's render cycle would
 * reintroduce the jank this is meant to remove.
 */
export interface NavIndicatorOptions {
  /** The scrolling nav element. Must be `position: relative`. */
  scroller: Ref<HTMLElement | null>
  /** The content element to observe for size changes (the menu `<ul>`). */
  content: Ref<HTMLElement | null>
  /** The pill element to position. */
  indicator: Ref<HTMLElement | null>
  /** Selector matching the currently active entry, e.g. `[data-nav-pill="primary"]`. */
  selector: string
}

export function useNavIndicator(options: NavIndicatorOptions) {
  const { scroller, content, indicator, selector } = options

  let frame = 0
  let observer: ResizeObserver | null = null
  let placed = false

  const hide = (pill: HTMLElement) => {
    pill.style.opacity = '0'
  }

  /**
   * True when an ancestor clips the target — i.e. the submenu holding it is
   * collapsing. The pill lives outside that clipping box, so without this it
   * would hang in mid-air over unrelated rows while the submenu closes.
   */
  const isClipped = (target: HTMLElement, root: HTMLElement): boolean => {
    const height = target.offsetHeight
    for (let el = target.parentElement; el && el !== root; el = el.parentElement) {
      if (el.offsetHeight < height) return true
    }
    return false
  }

  const paint = () => {
    frame = 0

    const nav = scroller.value
    const pill = indicator.value
    if (!nav || !pill) return

    const target = nav.querySelector<HTMLElement>(selector)
    if (!target) {
      hide(pill)
      return
    }

    const box = target.getBoundingClientRect()
    if (box.width === 0 || box.height === 0 || isClipped(target, nav)) {
      hide(pill)
      return
    }

    // getBoundingClientRect is viewport-relative and therefore already includes
    // the scroll offset; adding it back yields a stable content coordinate, so
    // the pill scrolls with its target without needing a scroll listener.
    const navBox = nav.getBoundingClientRect()
    const x = box.left - navBox.left + nav.scrollLeft
    const y = box.top - navBox.top + nav.scrollTop

    pill.style.transform = `translate3d(${x}px, ${y}px, 0)`
    pill.style.width = `${box.width}px`
    pill.style.height = `${box.height}px`
    pill.style.opacity = '1'

    if (!placed) {
      // Enable the transition only after the first placement, otherwise the
      // pill visibly slides in from the nav's top-left corner on page load.
      placed = true
      requestAnimationFrame(() => pill.classList.add('is-animated'))
    }
  }

  /** Coalesce bursts of measure requests into one write per frame. */
  const measure = () => {
    if (frame) return
    frame = requestAnimationFrame(paint)
  }

  onMounted(() => {
    measure()

    if (content.value) {
      // Observing the list (never the pill — resizing the pill inside its own
      // observer callback would loop) keeps the pill glued to a target that is
      // still moving as a submenu opens, instead of snapping at the end.
      observer = new ResizeObserver(measure)
      observer.observe(content.value)
    }

    window.addEventListener('resize', measure)
  })

  onBeforeUnmount(() => {
    if (frame) cancelAnimationFrame(frame)
    observer?.disconnect()
    window.removeEventListener('resize', measure)
  })

  return { measure }
}
