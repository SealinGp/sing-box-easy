import { onBeforeUnmount } from 'vue'

let users = 0
let lastTrigger: HTMLElement | null = null
let triggeredAt = 0

function rememberTrigger(event: Event) {
  lastTrigger = event.target instanceof Element
    ? event.target.closest<HTMLElement>('button, [role="button"], [data-overlay-anchor]')
    : null
  triggeredAt = performance.now()
}

/** Capture the initiating control before async handlers or focus changes run. */
export function useOverlayAnchor() {
  if (users++ === 0) {
    document.addEventListener('pointerdown', rememberTrigger, true)
    document.addEventListener('click', rememberTrigger, true)
  }
  onBeforeUnmount(() => {
    if (--users === 0) {
      document.removeEventListener('pointerdown', rememberTrigger, true)
      document.removeEventListener('click', rememberTrigger, true)
      lastTrigger = null
    }
  })
  return () => lastTrigger?.isConnected && performance.now() - triggeredAt < 5000
    ? lastTrigger
    : document.activeElement instanceof HTMLElement && document.activeElement.matches('button, [role="button"]')
      ? document.activeElement
      : null
}
