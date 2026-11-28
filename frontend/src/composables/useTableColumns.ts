import { onBeforeUnmount, watch, type Ref } from 'vue'

/** Opt-in resizing works with generated headers and custom header slots. */
export function useTableColumns(region: Ref<HTMLElement | null>, enabled: () => boolean, storageKey: () => string | undefined) {
  let cleanup = () => {}
  watch([region, enabled, storageKey], ([element, active]) => {
    cleanup()
    if (!element || !active) return
    const table = element.querySelector('table')
    if (!table) return
    const headers = [...table.querySelectorAll<HTMLElement>('thead th')]
    let saved: number[] = []
    try { saved = JSON.parse(localStorage.getItem(storageKey() ?? '') ?? '[]') } catch { /* Defaults remain usable. */ }
    const minimums = headers.map(header => Number(header.dataset.minWidth) || (header.classList.contains('col-actions') ? 160 : header.classList.contains('col-select') ? 56 : 80))
    const widths = headers.map((header, index) => Number.isFinite(saved[index]) ? Math.max(minimums[index]!, Math.min(800, saved[index]!)) : header.getBoundingClientRect().width)
    const group = document.createElement('colgroup')
    const cols = widths.map(width => { const col = document.createElement('col'); col.style.width = `${width}px`; group.append(col); return col })
    table.prepend(group)
    table.style.tableLayout = 'fixed'
    const paint = () => { table.style.width = `${widths.reduce((sum, width) => sum + width, 0)}px`; cols.forEach((col, i) => { col.style.width = `${widths[i]}px` }) }
    const persist = () => { try { if (storageKey()) localStorage.setItem(storageKey()!, JSON.stringify(widths)) } catch { /* Storage is optional. */ } }
    paint()
    const disposers = headers.map((header, index) => {
      const handle = document.createElement('span')
      handle.className = 'table-column-resizer'
      handle.tabIndex = 0
      handle.role = 'separator'
      handle.setAttribute('aria-orientation', 'vertical')
      handle.setAttribute('aria-label', header.textContent?.trim() ?? '')
      header.append(handle)
      const resize = (width: number) => { widths[index] = Math.max(minimums[index]!, Math.min(800, width)); handle.setAttribute('aria-valuenow', String(Math.round(widths[index]!))); paint() }
      handle.setAttribute('aria-valuemin', String(minimums[index]))
      handle.setAttribute('aria-valuemax', '800')
      resize(widths[index]!)
      handle.onpointerdown = event => {
        event.preventDefault(); event.stopPropagation()
        const start = event.clientX, width = widths[index]!
        handle.setPointerCapture(event.pointerId)
        handle.onpointermove = move => resize(width + move.clientX - start)
      }
      handle.onlostpointercapture = () => { handle.onpointermove = null; persist() }
      handle.onkeydown = event => {
        if (!['ArrowLeft', 'ArrowRight', 'Home'].includes(event.key)) return
        event.preventDefault(); event.stopPropagation()
        resize(event.key === 'Home' ? 160 : widths[index]! + (event.key === 'ArrowRight' ? 16 : -16)); persist()
      }
      handle.ondblclick = () => { resize(160); persist() }
      return () => handle.remove()
    })
    cleanup = () => { disposers.forEach(dispose => dispose()); group.remove(); table.style.removeProperty('width'); table.style.removeProperty('table-layout') }
  }, { flush: 'post' })
  onBeforeUnmount(() => cleanup())
}
