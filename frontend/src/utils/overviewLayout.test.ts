import { describe, expect, test } from 'bun:test'
import { placeTiles, type TileInput } from './overviewLayout'

const tile = (id: string, height: number, wide = false): TileInput => ({ id, height, wide })

describe('placeTiles', () => {
  test('a card starts directly under the card above it in its own column', () => {
    const placed = placeTiles([tile('a', 400), tile('b', 100), tile('c', 100), tile('d', 50), tile('e', 50)], 3, 16)
    // Row 1: a, b, c. `d` goes back to column 1 and waits for the tall `a`…
    expect(placed.get('d')).toEqual({ column: 1, columnSpan: 1, rowStart: 417, rowSpan: 66 })
    // …but `e`, in column 2, sits right under the short `b`, not level with `d`.
    expect(placed.get('e')).toEqual({ column: 2, columnSpan: 1, rowStart: 117, rowSpan: 66 })
  })

  test('a wide card spans every column and nothing after it rises above it', () => {
    const placed = placeTiles([tile('a', 300), tile('b', 40), tile('wide', 200, true), tile('c', 10), tile('d', 10)], 2, 10)
    // Starts under the TALLEST column so far (300 + 10), not the shortest.
    expect(placed.get('wide')).toEqual({ column: 1, columnSpan: 2, rowStart: 311, rowSpan: 210 })
    expect(placed.get('c')).toEqual({ column: 1, columnSpan: 1, rowStart: 521, rowSpan: 20 })
    expect(placed.get('d')).toEqual({ column: 2, columnSpan: 1, rowStart: 521, rowSpan: 20 })
  })

  test('the rotation restarts after a wide card, so the next card is at the left', () => {
    const placed = placeTiles([tile('a', 10), tile('wide', 10, true), tile('b', 10)], 3, 0)
    expect(placed.get('a')?.column).toBe(1)
    expect(placed.get('b')?.column).toBe(1)
  })

  test('a growing card never moves another card to a different column', () => {
    const before = placeTiles([tile('a', 100), tile('b', 100), tile('c', 100), tile('d', 100)], 3, 16)
    const after = placeTiles([tile('a', 900), tile('b', 100), tile('c', 100), tile('d', 100)], 3, 16)
    for (const id of ['a', 'b', 'c', 'd']) {
      expect(after.get(id)?.column).toBe(before.get(id)?.column)
    }
  })

  test('one column is a plain stack in order', () => {
    const placed = placeTiles([tile('a', 100), tile('wide', 50, true), tile('b', 20)], 1, 10)
    expect([placed.get('a')?.rowStart, placed.get('wide')?.rowStart, placed.get('b')?.rowStart]).toEqual([1, 111, 171])
    expect(placed.get('wide')?.columnSpan).toBe(1)
  })

  test('an unmeasured card still occupies a row, so two never share a cell', () => {
    const placed = placeTiles([tile('a', 0), tile('b', 0), tile('c', 0)], 1, 0)
    expect([placed.get('a')?.rowStart, placed.get('b')?.rowStart, placed.get('c')?.rowStart]).toEqual([1, 2, 3])
  })

  test('fractional heights round up so a card is never clipped by the next', () => {
    expect(placeTiles([tile('a', 100.2)], 1, 0).get('a')?.rowSpan).toBe(101)
  })
})
