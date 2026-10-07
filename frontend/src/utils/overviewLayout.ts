/**
 * Column packing for the Overview's cards.
 *
 * The cards used to sit in an ordinary row-based grid. Every card in a row
 * then shared the row's height, so a tall card left an empty band under each
 * of its shorter neighbours, and the page was mostly gaps.
 *
 * Here each card is given an explicit grid position instead: its column, and
 * the pixel row where that column currently ends. Cards therefore stack
 * directly under the card above them in the SAME column, whatever its
 * neighbours are doing.
 *
 * The DOM stays one flat list in the saved order — only the grid coordinates
 * change — which is what lets the drag-to-reorder mode and its FLIP animation
 * keep working unmodified.
 */

export interface TileInput {
  id: string
  /** Spans every column (the flow diagram). */
  wide: boolean
  /** Measured height in px; 0 before the first measurement. */
  height: number
}

export interface TilePlacement {
  /** 1-based grid column line. */
  column: number
  columnSpan: number
  /** 1-based grid row line; rows are 1px tall. */
  rowStart: number
  rowSpan: number
}

/**
 * Places tiles in order.
 *
 * A normal tile goes to the next column in rotation and starts where that
 * column ends. A wide tile starts below EVERYTHING before it and pushes every
 * column down past itself, so nothing that follows it can slip above it — the
 * saved order stays the reading order. The rotation restarts after a wide
 * tile, so the first card under it is always at the left.
 *
 * Rotation rather than "shortest column": a card's column then depends only on
 * its position in the order, never on heights, so a card that grows (a probe
 * result appearing, a chart filling) cannot make a different card jump across
 * the page.
 */
export function placeTiles(tiles: readonly TileInput[], columns: number, gap: number): Map<string, TilePlacement> {
  const count = Math.max(1, Math.floor(columns))
  const ends: number[] = Array.from({ length: count }, () => 0)
  const placements = new Map<string, TilePlacement>()
  let turn = 0

  for (const tile of tiles) {
    // A row line per pixel: the span covers the card and the gap beneath it.
    const span = Math.max(1, Math.ceil(tile.height) + gap)

    if (tile.wide) {
      const start = Math.max(...ends)
      placements.set(tile.id, { column: 1, columnSpan: count, rowStart: start + 1, rowSpan: span })
      ends.fill(start + span)
      turn = 0
      continue
    }

    const column = turn % count
    const start = ends[column] ?? 0
    placements.set(tile.id, { column: column + 1, columnSpan: 1, rowStart: start + 1, rowSpan: span })
    ends[column] = start + span
    turn += 1
  }
  return placements
}
