import { expect, test } from 'bun:test'
import { moveCard, overviewSlots } from './overviewSort'
const cards = ['a', 'b', 'route-topology', 'c', 'd'].map(id => ({ id, title: id }))
test('full-width cards start a fresh row and leave explicit empty cells', () => {
  expect(overviewSlots(cards, 3).map(({ row, column, span }) => [row, column, span])).toEqual([[1,1,1],[1,2,1],[2,1,3],[3,1,1],[3,2,1]])
})
test('preview does not mutate the draft; moving a wide card repacks predictably', () => {
  const preview = moveCard(cards, 2, 0)
  expect(cards[0]!.id).toBe('a')
  expect(preview.map(c => c.id)).toEqual(['route-topology','a','b','c','d'])
  expect(overviewSlots(preview, 3).map(({ row, column, span }) => [row,column,span])).toEqual([[1,1,3],[2,1,1],[2,2,1],[2,3,1],[3,1,1]])
})
test('one-column layout and moves to the final position preserve every card', () => {
  const preview = moveCard(cards, 0, 4)
  expect(preview.map(c => c.id)).toEqual(['b','route-topology','c','d','a'])
  expect(overviewSlots(preview, 1).map(s => s.row)).toEqual([1,2,3,4,5])
  expect(moveCard(cards, 0, -1)).toEqual(cards)
})
