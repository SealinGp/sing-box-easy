/** Pixels per second, increasing as the pointer approaches a visible edge. */
export function dragScrollSpeed(pointer: number, start: number, end: number): number {
  if (pointer < start || pointer > end || end <= start) return 0
  const edge = Math.min(80, (end - start) / 3)
  if (pointer < start + edge) return -720 * (1 - (pointer - start) / edge)
  if (pointer > end - edge) return 720 * (1 - (end - pointer) / edge)
  return 0
}

/** Scroll the dragged row's ancestors, never a diagram nested inside it. */
export function startDragAutoScroll(source: HTMLElement): () => void {
  const doc = source.ownerDocument
  const view = doc.defaultView
  if (!view) return () => {}
  const parents: HTMLElement[] = []
  for (let el = source.parentElement; el; el = el.parentElement) {
    if (/(auto|scroll)/.test(view.getComputedStyle(el).overflowY)) parents.push(el)
  }
  const root = doc.scrollingElement as HTMLElement | null
  if (root && !parents.includes(root)) parents.push(root)

  let pointer: { x: number; y: number } | null = null
  let frame = 0
  let lastTime: number | null = null
  let stopped = false

  function tick(time: number) {
    if (stopped) return
    const elapsed = lastTime === null ? 1 / 60 : Math.min((time - lastTime) / 1000, 0.05)
    lastTime = time
    if (pointer) {
      for (const el of parents) {
        const box = el === root
          ? { top: 0, bottom: view!.innerHeight, left: 0, right: view!.innerWidth }
          : el.getBoundingClientRect()
        if (pointer.x < box.left || pointer.x > box.right) continue
        const speed = dragScrollSpeed(pointer.y, Math.max(0, box.top), Math.min(view!.innerHeight, box.bottom))
        if (!speed) continue
        const before = el.scrollTop
        // Explicit instant scrolling also works when the app enables smooth scrolling.
        el.scrollBy({ top: speed * elapsed, behavior: 'instant' })
        if (el.scrollTop !== before) break
      }
    }
    frame = view!.requestAnimationFrame(tick)
  }

  function track(event: DragEvent) {
    pointer = { x: event.clientX, y: event.clientY }
  }
  function leave(event: DragEvent) {
    if (!event.relatedTarget) pointer = null
  }
  function stop() {
    stopped = true
    view!.cancelAnimationFrame(frame)
    doc.removeEventListener('dragover', track, true)
    doc.removeEventListener('dragleave', leave, true)
    doc.removeEventListener('drop', stop, true)
    doc.removeEventListener('dragend', stop, true)
    view!.removeEventListener('blur', stop)
  }
  doc.addEventListener('dragover', track, true)
  doc.addEventListener('dragleave', leave, true)
  doc.addEventListener('drop', stop, true)
  doc.addEventListener('dragend', stop, true)
  view.addEventListener('blur', stop)
  frame = view.requestAnimationFrame(tick)
  return stop
}
