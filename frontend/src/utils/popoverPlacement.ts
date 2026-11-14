export interface PopoverOrigin { x: number; y: number }

/** Select the quadrant with the largest usable area, not the longest diagonal. */
export function popoverPlacement(origin: PopoverOrigin, viewport: { width: number; height: number }, size: { width: number; height: number }) {
  const margin = 16, gap = 8
  const x = Math.max(margin + gap, Math.min(origin.x, viewport.width - margin - gap))
  const y = Math.max(margin + gap, Math.min(origin.y, viewport.height - margin - gap))
  const candidates = [
    { horizontal: 'right', vertical: 'bottom', width: viewport.width - margin - x - gap, height: viewport.height - margin - y - gap },
    { horizontal: 'left', vertical: 'bottom', width: x - margin - gap, height: viewport.height - margin - y - gap },
    { horizontal: 'right', vertical: 'top', width: viewport.width - margin - x - gap, height: y - margin - gap },
    { horizontal: 'left', vertical: 'top', width: x - margin - gap, height: y - margin - gap },
  ]
  const best = candidates.sort((a, b) => b.width * b.height - a.width * a.height)[0]!
  const width = Math.min(size.width, best.width)
  const height = Math.min(size.height, best.height)
  return {
    left: best.horizontal === 'right' ? x + gap : x - gap - width,
    top: best.vertical === 'bottom' ? y + gap : y - gap - height,
    maxWidth: best.width,
    maxHeight: best.height,
    origin: `${best.horizontal === 'right' ? 'left' : 'right'} ${best.vertical === 'bottom' ? 'top' : 'bottom'}`,
  }
}
