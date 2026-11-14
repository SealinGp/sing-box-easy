import { expect, test } from 'bun:test'
import { popoverPlacement } from './popoverPlacement'
const viewport = { width: 1200, height: 800 }
const size = { width: 560, height: 300 }
test('prefers below and centers on the triggering control', () => {
  const result = popoverPlacement({ left: 500, right: 600, top: 100, bottom: 130 }, viewport, size)
  expect(result.side).toBe('bottom')
  expect(result.left).toBe(270)
  expect(result.top).toBe(138)
})
test('flips above lower rows, shifts near edges without narrowing', () => {
  const result = popoverPlacement({ left: 1100, right: 1180, top: 700, bottom: 730 }, viewport, size)
  expect(result.side).toBe('top')
  expect(result.top).toBe(392)
  expect(result.left).toBe(624)
  expect(result.maxWidth).toBe(1168)
})
test('retains side as asynchronous details grow', () => {
  const anchor = { left: 500, right: 600, top: 400, bottom: 430 }
  const initial = popoverPlacement(anchor, viewport, { width: 560, height: 100 })
  const expanded = popoverPlacement(anchor, viewport, { width: 560, height: 700 }, initial.side)
  expect(expanded.side).toBe(initial.side)
  expect(expanded.top).toBe(initial.top)
  expect(expanded.maxHeight).toBe(346)
})
