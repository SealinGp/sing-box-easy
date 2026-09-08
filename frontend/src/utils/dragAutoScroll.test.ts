import { expect, test } from 'bun:test'
import { dragScrollSpeed, startDragAutoScroll } from './dragAutoScroll'

test('edge speed respects clipped scroll bounds and stops in the middle', () => {
  expect(dragScrollSpeed(100, 100, 700)).toBe(-720)
  expect(dragScrollSpeed(700, 100, 700)).toBe(720)
  expect(dragScrollSpeed(660, 100, 700)).toBe(360)
  expect(dragScrollSpeed(400, 100, 700)).toBe(0)
  expect(dragScrollSpeed(99, 100, 700)).toBe(0)
  expect(dragScrollSpeed(701, 100, 700)).toBe(0)
})

test('stationary drag scrolls the main container, reverses, and cleans up on drop', () => {
  const listeners = new Map<string, (event: any) => void>()
  let tick: FrameRequestCallback | undefined
  const view = {
    innerHeight: 800, innerWidth: 1200,
    getComputedStyle: () => ({ overflowY: 'auto' }),
    requestAnimationFrame: (callback: FrameRequestCallback) => { tick = callback; return 1 },
    cancelAnimationFrame: () => { tick = undefined },
    addEventListener: (name: string, fn: (event: any) => void) => listeners.set(name, fn),
    removeEventListener: (name: string) => listeners.delete(name),
  }
  const main = {
    parentElement: null, scrollTop: 300,
    getBoundingClientRect: () => ({ top: 60, bottom: 900, left: 200, right: 1200 }),
    scrollBy({ top }: { top: number }) { this.scrollTop += top },
  }
  const doc = {
    defaultView: view, scrollingElement: null,
    addEventListener: view.addEventListener, removeEventListener: view.removeEventListener,
  }
  const stop = startDragAutoScroll({ ownerDocument: doc, parentElement: main } as unknown as HTMLElement)
  listeners.get('dragover')!({ clientX: 600, clientY: 795 })
  tick!(0)
  const first = main.scrollTop
  tick!(16)
  expect(main.scrollTop).toBeGreaterThan(first)
  listeners.get('dragover')!({ clientX: 600, clientY: 65 })
  tick!(32)
  expect(main.scrollTop).toBeCloseTo(first)
  listeners.get('dragleave')!({ relatedTarget: null })
  const paused = main.scrollTop
  tick!(48)
  expect(main.scrollTop).toBe(paused)
  listeners.get('drop')!({})
  expect(tick).toBeUndefined()
  expect(listeners.size).toBe(0)
  stop()
})
