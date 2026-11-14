export type PopoverSide = 'top' | 'bottom'
export interface PopoverAnchor { left: number; right: number; top: number; bottom: number }

/** Stay attached to the control; flip only when the preferred side cannot fit. */
export function popoverPlacement(anchor: PopoverAnchor, viewport: { width: number; height: number }, size: { width: number; height: number }, preferred?: PopoverSide) {
  const margin = 16, gap = 8
  const below = Math.max(0, viewport.height - margin - anchor.bottom - gap)
  const above = Math.max(0, anchor.top - margin - gap)
  const side = preferred ?? (size.height <= below || below >= above ? 'bottom' : 'top')
  const maxWidth = Math.max(0, viewport.width - margin * 2)
  const maxHeight = side === 'bottom' ? below : above
  const width = Math.min(size.width, maxWidth)
  const height = Math.min(size.height, maxHeight)
  const center = (anchor.left + anchor.right) / 2
  const left = Math.max(margin, Math.min(center - width / 2, viewport.width - width - margin))
  return {
    left,
    top: side === 'bottom' ? anchor.bottom + gap : anchor.top - gap - height,
    maxWidth, maxHeight, side,
    origin: `${Math.max(0, Math.min(width, center - left))}px ${side === 'bottom' ? 'top' : 'bottom'}`,
  }
}
