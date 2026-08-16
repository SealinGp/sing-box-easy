import { onBeforeUnmount, onMounted, watch, type Ref } from 'vue'

/**
 * Sizes a scroll region to the space actually left below it.
 *
 * WHY THIS IS NOT CSS
 * ───────────────────
 * This started as `max-height: calc(100dvh - <constant>)`, and the constant was
 * wrong three separate times:
 *
 *   1. It was measured on the sidebar layout, so every OpenWrt page — the
 *      deployment that matters most — overran the window by the top bar's
 *      height.
 *   2. Pages differ. A DNS panel with a titled header inside its card spends
 *      ~60px more than the Outbounds table; a `<List>` inside a `<Card>` spends
 *      another 16px on the card's own bottom padding. Each needed its own
 *      override class.
 *   3. Fatally: `Topbar.vue` uses `flex-wrap`. Measured on this app, the header
 *      is 45px on a wide screen and **180px** once it wraps — a 136px swing. No
 *      constant survives that, and a router's LuCI page is exactly where it
 *      wraps.
 *
 * `container-type: size` on `<main>` would have fixed 1 and 2 in pure CSS, but
 * size containment implies `contain: layout`, which makes `<main>` the
 * containing block for the `fixed inset-0` modals that Config, NodeRules and
 * Profile render inside it — they would position against `main`, not the
 * viewport.
 *
 * So: measure. One mechanism, no constants, correct for any layout and any
 * chrome, including a wrapped top bar.
 *
 * NO FEEDBACK LOOP
 * ────────────────
 * Both inputs are deliberately independent of the element's own height:
 * `offsetTop` is set by the content *above* the element, and trailing space by
 * the content *below* it. Applying a new max-height therefore cannot change the
 * next measurement, so re-entrant observer fires converge instead of
 * oscillating.
 */

/** Floor, so a pathological layout can never collapse the region to nothing. */
const MIN_HEIGHT = 120

/** The nearest ancestor that scrolls; the viewport if none does. */
function scrollParent(el: HTMLElement): HTMLElement {
  let node = el.parentElement
  while (node && node !== document.body) {
    const { overflowY } = getComputedStyle(node)
    if (overflowY === 'auto' || overflowY === 'scroll') return node
    node = node.parentElement
  }
  return document.documentElement
}

/**
 * Distance from the container's content top to the element's top.
 *
 * Rect difference plus `scrollTop`, NOT an `offsetTop` walk. `offsetTop` is
 * measured against `offsetParent`, which skips statically positioned ancestors
 * — and `<main>` is static, so the chain steps straight over it to `<body>` and
 * silently folds the top bar's height into the answer. That cost the region ~57
 * extra pixels on the OpenWrt layout.
 *
 * Adding `scrollTop` back makes this independent of where the container happens
 * to be scrolled when we measure, which is what `offsetTop` was chosen for.
 */
function offsetWithin(el: HTMLElement, container: HTMLElement): number {
  const elTop = el.getBoundingClientRect().top
  const containerTop = container.getBoundingClientRect().top
  return elTop - containerTop + container.scrollTop
}

/**
 * Everything that still has to fit below the element: each ancestor's bottom
 * padding and border up to the container, plus any siblings rendered after it.
 * This is what the old per-page `--scroll-chrome-extra` constants were
 * hand-approximating.
 */
function trailingSpace(el: HTMLElement, container: HTMLElement): number {
  let total = 0
  let node: HTMLElement | null = el

  while (node && node !== container) {
    for (let sib = node.nextElementSibling; sib; sib = sib.nextElementSibling) {
      total += sib.getBoundingClientRect().height
    }
    total += parseFloat(getComputedStyle(node).marginBottom) || 0

    const parent: HTMLElement | null = node.parentElement
    if (!parent) break
    if (parent !== container) {
      const cs = getComputedStyle(parent)
      total += parseFloat(cs.paddingBottom) || 0
      total += parseFloat(cs.borderBottomWidth) || 0
    }
    node = parent
  }

  return total
}

/**
 * @param elRef    the scroll region
 * @param disabled skip entirely — the caller pinned an explicit max-height, or
 *                 an ancestor owns the scrolling
 */
export function useFillHeight(elRef: Ref<HTMLElement | null>, disabled: Ref<boolean>) {
  let observer: ResizeObserver | null = null
  let frame = 0

  const apply = () => {
    const el = elRef.value
    if (!el || disabled.value) return

    const container = scrollParent(el)
    const available =
      container.clientHeight - offsetWithin(el, container) - trailingSpace(el, container)
    const next = Math.max(Math.round(available), MIN_HEIGHT)

    // Only write on a real change: a no-op style write still invalidates layout,
    // and this runs from a ResizeObserver.
    if (el.style.getPropertyValue('--scroll-max-h') !== `${next}px`) {
      el.style.setProperty('--scroll-max-h', `${next}px`)
    }
  }

  // Coalesce bursts — a viewport resize fires the observer for the container
  // and the parent in the same frame.
  const schedule = () => {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(apply)
  }

  const attach = () => {
    observer?.disconnect()
    const el = elRef.value
    if (!el || disabled.value) return

    observer = new ResizeObserver(schedule)
    // The container catches the viewport and a wrapping top bar (both change
    // `<main>`'s height). The parent catches chrome above the region growing
    // inside the page — a toolbar wrapping, an alert appearing.
    observer.observe(scrollParent(el))
    if (el.parentElement) observer.observe(el.parentElement)
    schedule()
  }

  onMounted(attach)
  // The region only exists in the data branch, so the ref goes null and back
  // whenever the caller flips to loading or empty.
  watch([elRef, disabled], attach)

  onBeforeUnmount(() => {
    cancelAnimationFrame(frame)
    observer?.disconnect()
    observer = null
  })
}
