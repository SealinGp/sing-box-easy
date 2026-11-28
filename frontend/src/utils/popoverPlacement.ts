export type PopoverSide = 'top' | 'bottom'
export interface PopoverAnchor { left: number; right: number; top: number; bottom: number }

/** Stay attached to the control; flip only when the preferred side cannot fit. */
export function popoverPlacement(anchor: PopoverAnchor, viewport: { width: number; height: number }, size: { width: number; height: number }, preferred?: PopoverSide) {
  const margin = 16, gap = 8
  const clampY = (value: number) => Math.max(margin, Math.min(value, viewport.height - margin))
  const anchorTop = clampY(anchor.top), anchorBottom = clampY(anchor.bottom)
  const below = Math.max(0, viewport.height - margin - anchorBottom - gap)
  const above = Math.max(0, anchorTop - margin - gap)
  let side = preferred ?? (size.height <= below || below >= above ? 'bottom' : 'top')
  // Keep the direction stable as fields grow, but keep at least a usable
  // portion of the form visible when scrolling moves its anchor offscreen.
  const room = side === 'bottom' ? below : above
  const oppositeRoom = side === 'bottom' ? above : below
  if (room < Math.min(size.height, 96) && oppositeRoom > room) side = side === 'bottom' ? 'top' : 'bottom'
  const maxWidth = Math.max(0, viewport.width - margin * 2)
  const maxHeight = side === 'bottom' ? below : above
  const width = Math.min(size.width, maxWidth)
  const height = Math.min(size.height, maxHeight)
  const center = (anchor.left + anchor.right) / 2
  const left = Math.max(margin, Math.min(center - width / 2, viewport.width - width - margin))
  return {
    left,
    top: Math.max(margin, side === 'bottom' ? anchorBottom + gap : anchorTop - gap - height),
    maxWidth, maxHeight, side,
    origin: `${Math.max(0, Math.min(width, center - left))}px ${side === 'bottom' ? 'top' : 'bottom'}`,
  }
}
