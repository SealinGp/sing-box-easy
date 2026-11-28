import { onBeforeUnmount, ref, type Ref } from 'vue'

/** Horizontal tab dragging commits once on drop; Escape cancels without reordering. */
export function useWorkspaceDrag(nav: Ref<HTMLElement | undefined>, move: (from: string, to: string, before: boolean) => void) {
  const dragging = ref('')
  const target = ref('')
  const before = ref(true)
  const offset = ref(0)
  let source = ''
  let startX = 0
  let lastX = 0
  let initialScroll = 0
  let frame = 0
  let suppressClick = false
  let captured: HTMLElement | null = null
  let pointerId = -1
  function locate() {
    const strip = nav.value
    if (!strip) return
    const bounds = strip.getBoundingClientRect()
    const velocity = lastX < bounds.left + 28 ? -8 : lastX > bounds.right - 28 ? 8 : 0
    strip.scrollLeft += velocity
    offset.value = lastX - startX + strip.scrollLeft - initialScroll
    const candidates = [...strip.querySelectorAll<HTMLElement>('[data-workspace-path]')].filter(row => row.dataset.workspacePath !== source && row.offsetWidth)
    const closest = candidates.reduce<HTMLElement | undefined>((best, row) => {
      const center = (el: HTMLElement) => { const r = el.getBoundingClientRect(); return r.left + r.width / 2 }
      return !best || Math.abs(center(row) - lastX) < Math.abs(center(best) - lastX) ? row : best
    }, undefined)
    if (closest) {
      const rect = closest.getBoundingClientRect()
      target.value = closest.dataset.workspacePath ?? ''
      before.value = lastX < rect.left + rect.width / 2
    }
    frame = requestAnimationFrame(locate)
  }
  function cleanup() {
    cancelAnimationFrame(frame)
    window.removeEventListener('pointermove', pointerMove)
    window.removeEventListener('pointerup', drop)
    window.removeEventListener('pointercancel', cancel)
    window.removeEventListener('keydown', escape)
    if (captured?.hasPointerCapture(pointerId)) captured.releasePointerCapture(pointerId)
    captured = null
    source = ''; dragging.value = ''; target.value = ''; offset.value = 0
  }
  function pointerMove(event: PointerEvent) {
    if (event.pointerId !== pointerId) return
    lastX = event.clientX
    if (!dragging.value && Math.abs(lastX - startX) > 6) {
      dragging.value = source
      suppressClick = true
      captured?.setPointerCapture(pointerId)
      locate()
    }
    if (dragging.value) event.preventDefault()
  }
  function drop(event: PointerEvent) {
    if (event.pointerId !== pointerId) return
    if (dragging.value && target.value) move(source, target.value, before.value)
    cleanup()
  }
  function cancel() { cleanup() }
  function escape(event: KeyboardEvent) { if (event.key === 'Escape') { event.preventDefault(); cleanup() } }
  function start(event: PointerEvent, path: string) {
    if (event.button !== 0 || (event.target instanceof Element && event.target.closest('.workspace-tab-close'))) return
    cleanup(); suppressClick = false
    source = path; startX = lastX = event.clientX; initialScroll = nav.value?.scrollLeft ?? 0
    pointerId = event.pointerId; captured = event.currentTarget as HTMLElement
    window.addEventListener('pointermove', pointerMove, { passive: false })
    window.addEventListener('pointerup', drop)
    window.addEventListener('pointercancel', cancel)
    window.addEventListener('keydown', escape)
  }
  function consumeClick() { const consumed = suppressClick; suppressClick = false; return consumed }
  onBeforeUnmount(cleanup)
  return { dragging, target, before, offset, start, consumeClick }
}
