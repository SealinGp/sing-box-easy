import { expect, test } from 'bun:test'
import { popoverPlacement } from './popoverPlacement'

test('opens toward the largest quadrant for each corner', () => {
  const viewport = { width: 1200, height: 800 }
  const size = { width: 400, height: 300 }
  for (const [x, y, origin] of [[100, 100, 'left top'], [1100, 100, 'right top'], [100, 700, 'left bottom'], [1100, 700, 'right bottom']] as const) {
    const result = popoverPlacement({ x, y }, viewport, size)
    expect(result.origin).toBe(origin)
    expect(result.left).toBeGreaterThanOrEqual(16)
    expect(result.top).toBeGreaterThanOrEqual(16)
    expect(result.left + Math.min(size.width, result.maxWidth)).toBeLessThanOrEqual(1184)
    expect(result.top + Math.min(size.height, result.maxHeight)).toBeLessThanOrEqual(784)
  }
})
test('oversized panels remain bounded on narrow screens', () => {
  const result = popoverPlacement({ x: 370, y: 780 }, { width: 390, height: 844 }, { width: 900, height: 1000 })
  expect(result.left).toBe(16)
  expect(result.top).toBe(16)
  expect(result.maxWidth).toBeLessThan(390)
  expect(result.maxHeight).toBeLessThan(844)
})
