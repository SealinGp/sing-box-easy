export interface SortCard { id: string; title: string }
export interface SortSlot { id: string; index: number; row: number; column: number; span: number }

export function moveCard<T>(cards: readonly T[], from: number, to: number): T[] {
  const next = [...cards]
  if (from < 0 || to < 0 || from >= next.length || to >= next.length) return next
  const [card] = next.splice(from, 1)
  next.splice(to, 0, card!)
  return next
}

/** Row-major slots; the topology takes a complete row. Holes stay visible. */
export function overviewSlots(cards: readonly SortCard[], columns: number): SortSlot[] {
  let row = 1
  let column = 1
  return cards.map((card, index) => {
    const span = card.id === 'route-topology' ? columns : 1
    if (column + span - 1 > columns) { row++; column = 1 }
    const slot = { id: card.id, index, row, column, span }
    column += span
    if (column > columns) { row++; column = 1 }
    return slot
  })
}
