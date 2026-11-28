import { onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { useNavIndicator } from './useNavIndicator'
import { useWorkspaceActive } from './workspaceContext'
import '../style/row-highlight.css'

/** One delegated hover/focus surface per collection, regardless of row count. */
export function useRowHighlight(root: Ref<HTMLElement | null>, selector: string) {
  const indicator = ref<HTMLElement | null>(null)
  const { measure } = useNavIndicator({ scroller: root, content: root, indicator, selector: '[data-moving-highlight-target]' })
  let current: HTMLElement | null = null
  let observer: MutationObserver | undefined
  let host: HTMLElement | null = null
  let enterFrame = 0
  function clear() {
    cancelAnimationFrame(enterFrame)
    current?.removeAttribute('data-moving-highlight-target')
    current = null
    if (indicator.value) {
      indicator.value.style.opacity = '0'
      indicator.value.classList.remove('is-animated')
    }
  }
  function choose(target: EventTarget | null) {
    const row = target instanceof Element ? target.closest<HTMLElement>(selector) : null
    if (!row || !host?.contains(row) || row.closest('[inert]') || row.matches('.is-dragging')) { clear(); return }
    if (row === current) return
    current?.removeAttribute('data-moving-highlight-target')
    current = row
    row.setAttribute('data-moving-highlight-target', '')
    measure()
    // Initial entry is positioned immediately; only travel within the list animates.
    cancelAnimationFrame(enterFrame)
    enterFrame = requestAnimationFrame(() => {
      enterFrame = requestAnimationFrame(() => { if (current) indicator.value?.classList.add('is-animated') })
    })
  }
  function pointer(event: PointerEvent) { if (event.pointerType !== 'touch') choose(event.target) }
  function focus(event: FocusEvent) {
    if (event.target instanceof Element && event.target.matches(':focus-visible')) choose(event.target)
  }
  function blur(event: FocusEvent) { if (!host?.contains(event.relatedTarget as Node)) clear() }
  function cleanup() {
    clear(); observer?.disconnect()
    host?.removeEventListener('pointermove', pointer)
    host?.removeEventListener('pointerleave', clear)
    host?.removeEventListener('focusin', focus)
    host?.removeEventListener('focusout', blur)
    document.removeEventListener('scroll', clear, true)
    host?.classList.remove('has-moving-highlight')
    indicator.value?.remove(); indicator.value = null
  }
  function attach(element: HTMLElement | null) {
    cleanup(); host = element
    if (!host) return
    host.classList.add('has-moving-highlight')
    const pill = document.createElement('div')
    pill.className = 'nav-active-indicator row-moving-highlight'
    pill.setAttribute('aria-hidden', 'true')
    host.append(pill); indicator.value = pill
    host.addEventListener('pointermove', pointer)
    host.addEventListener('pointerleave', clear)
    host.addEventListener('focusin', focus)
    host.addEventListener('focusout', blur)
    document.addEventListener('scroll', clear, true)
    observer = new MutationObserver(() => { if (current && !current.isConnected) clear() })
    observer.observe(host, { childList: true, subtree: true })
  }
  onMounted(() => attach(root.value))
  watch(root, attach, { flush: 'post' })
  watch(useWorkspaceActive(), active => { if (!active) clear() })
  onBeforeUnmount(cleanup)
}
