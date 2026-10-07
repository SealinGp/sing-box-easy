import { computed, nextTick, onBeforeUnmount, watch } from 'vue'
import { useRoute } from 'vue-router'
import { focusFromQuery } from '../utils/focusTarget'

/**
 * The receiving half of a `?focus=<name>` link (see utils/focusTarget.ts).
 *
 * A page calls this once, puts `:class="{ 'focus-target': isFocused(name) }"`
 * on each card that can be linked to, and the matching card is tinted and
 * scrolled into view.
 *
 * `ready` says when the cards exist. Every one of these pages loads its list
 * after mounting, so scrolling on mount would find nothing — and the scroll
 * has to be retried when the data arrives, not just when the query changes.
 *
 * The highlight lasts as long as the query does. It is not timed out: the
 * link is also what a reload or a pasted URL shows, and a tint that had faded
 * by the time the operator looked would have told them nothing.
 */
/** Up to ~3s of looking for the card before giving up quietly. */
const SCROLL_ATTEMPTS = 30
const SCROLL_RETRY_MS = 100

export function useFocusTarget(ready: () => boolean) {
  const route = useRoute()
  const focus = computed(() => focusFromQuery(route.query.focus))

  const isFocused = (name: string | undefined | null): boolean =>
    focus.value !== '' && name === focus.value

  /**
   * Scroll to the focused card once it exists.
   *
   * Retried, because "the data is loaded" and "the card is in the DOM" are
   * different moments: the page may be a lazily loaded route chunk, its list
   * may sit behind a loading state, and its store may ALREADY hold the data
   * from another page — in which case `ready` is true before anything has
   * rendered (arriving on Rule Sets from a connection did exactly that, and
   * the first version of this scrolled to nothing).
   */
  let timer: ReturnType<typeof setTimeout> | null = null
  const stop = () => {
    if (timer !== null) clearTimeout(timer)
    timer = null
  }
  const scrollWhenRendered = (attempt = 0) => {
    stop()
    const target = document.querySelector('.focus-target')
    if (target) {
      target.scrollIntoView({ block: 'center' })
      return
    }
    if (attempt < SCROLL_ATTEMPTS) {
      timer = setTimeout(() => scrollWhenRendered(attempt + 1), SCROLL_RETRY_MS)
    }
  }

  watch(
    [focus, ready],
    async ([name, isReady]) => {
      stop()
      if (!name || !isReady) return
      await nextTick()
      scrollWhenRendered()
    },
    { immediate: true },
  )
  onBeforeUnmount(stop)

  return { focus, isFocused }
}
