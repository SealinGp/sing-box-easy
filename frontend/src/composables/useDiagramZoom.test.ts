import { describe, expect, test } from 'bun:test'
import {
  MAX_ZOOM,
  MIN_ZOOM,
  ZOOM_STEPS,
  anchoredScroll,
  clampZoom,
  pannedScroll,
  stepDown,
  stepUp,
} from './useDiagramZoom'

describe('zoom ladder', () => {
  test('steps up and down through the same values', () => {
    expect(stepUp(1)).toBe(1.25)
    expect(stepDown(1.25)).toBe(1)
    expect(stepUp(stepDown(1.25)!)).toBe(1.25)
  })

  // A shrunk diagram measures at something like 0.54; the first click must land
  // on the next rung up, not jump to 125%.
  test('steps from an off-ladder measured scale to the neighbouring rung', () => {
    expect(stepUp(0.54)).toBe(0.67)
    expect(stepDown(0.54)).toBe(0.5)
  })

  test('ends rather than wrapping at the extremes', () => {
    expect(stepUp(MAX_ZOOM)).toBeNull()
    expect(stepDown(MIN_ZOOM)).toBeNull()
  })

  // Floats measured off the DOM are never exactly a ladder value; a naive `>`
  // would step 1.0000001 up to 1.25 and back down to 1, which reads as a stuck
  // control.
  test('does not step on a float that is a rung within epsilon', () => {
    expect(stepUp(1.0000001)).toBe(1.25)
    expect(stepDown(0.9999999)).toBe(0.8)
  })

  test('100% is on the ladder, so actual size is always reachable', () => {
    expect(ZOOM_STEPS).toContain(1)
  })

  test('clamps outside the ladder', () => {
    expect(clampZoom(99)).toBe(MAX_ZOOM)
    expect(clampZoom(0.01)).toBe(MIN_ZOOM)
    expect(clampZoom(1.5)).toBe(1.5)
  })
})

describe('anchoredScroll', () => {
  const anchor = (cursorX: number, scrollLeft = 0) => ({
    rect: { left: 100, top: 50 },
    cursor: { x: cursorX, y: 250 },
    scroll: { left: scrollLeft, top: 0 },
  })

  // The point under the cursor must stay under the cursor: at 300px into the
  // content, doubling puts it at 600, so the container has to scroll 300 more.
  test('keeps the point under the cursor fixed', () => {
    const out = anchoredScroll(anchor(400), 1, 2)
    expect(out.left).toBe(300)
  })

  test('accounts for an existing scroll offset', () => {
    const out = anchoredScroll(anchor(400, 500), 1, 2)
    expect(out.left).toBe(1300)
  })

  test('is an identity when the scale does not change', () => {
    const out = anchoredScroll(anchor(400, 120), 1.5, 1.5)
    expect(out.left).toBe(120)
  })

  // Zooming out past the point where the content overflows leaves nothing to
  // scroll; a negative offset would be clamped by the DOM anyway, and reporting
  // it would make the pure function disagree with the element.
  test('never returns a negative offset', () => {
    const out = anchoredScroll(anchor(400), 2, 0.5)
    expect(out.left).toBe(0)
    expect(out.top).toBe(0)
  })

  test('is inert rather than dividing by zero on an unmeasurable scale', () => {
    const out = anchoredScroll(anchor(400, 42), 0, 2)
    expect(out.left).toBe(42)
  })
})

describe('pannedScroll', () => {
  const origin = (left = 400, top = 200) => ({
    pointer: { x: 500, y: 300 },
    scroll: { left, top },
  })

  // Grab-the-canvas: the picture follows the pointer, so the scroll offset
  // moves the other way. Inverting this makes the diagram flee the cursor.
  test('moving the pointer right scrolls left, and vice versa', () => {
    expect(pannedScroll(origin(), { x: 560, y: 300 }).left).toBe(340)
    expect(pannedScroll(origin(), { x: 440, y: 300 }).left).toBe(460)
  })

  test('pans both axes at once', () => {
    const out = pannedScroll(origin(), { x: 450, y: 260 })
    expect(out).toEqual({ left: 450, top: 240 })
  })

  test('no movement is no scroll', () => {
    expect(pannedScroll(origin(), { x: 500, y: 300 })).toEqual({ left: 400, top: 200 })
  })

  test('stops at the start rather than going negative', () => {
    expect(pannedScroll(origin(10, 10), { x: 900, y: 700 })).toEqual({ left: 0, top: 0 })
  })

  /**
   * Recomputed from the origin, never accumulated. Dragging to an edge and
   * back must land exactly where it started — an accumulating implementation
   * runs ahead of the element by however far the DOM refused to scroll, so the
   * return trip has a dead zone before the picture moves again.
   */
  test('returning the pointer to the origin returns the scroll, even after hitting an edge', () => {
    const start = origin(50, 50)
    expect(pannedScroll(start, { x: 5000, y: 5000 })).toEqual({ left: 0, top: 0 })
    expect(pannedScroll(start, { x: 500, y: 300 })).toEqual({ left: 50, top: 50 })
  })
})
