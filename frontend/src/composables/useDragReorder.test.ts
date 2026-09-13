import { describe, expect, test } from 'bun:test'
import { ref } from 'vue'
import { useDragReorder } from './useDragReorder'

describe('overview arrangement sessions', () => {
  test('several moves save one order, and cancel restores the saved layout', async () => {
    const items = ref(['topology', 'status', 'dns'])
    const writes: number[][] = []
    const reorder = useDragReorder(items, async order => { writes.push(order) })
    reorder.syncKeys(items.value.length)
    reorder.start()
    reorder.nudge(2, -2)
    reorder.nudge(1, 1)
    expect(items.value).toEqual(['dns', 'status', 'topology'])
    expect(writes).toEqual([])
    await reorder.save()
    expect(writes).toEqual([[2, 1, 0]])
    reorder.start()
    reorder.nudge(0, 2)
    reorder.cancel()
    expect(items.value).toEqual(['dns', 'status', 'topology'])
    expect(writes).toHaveLength(1)
  })

  test('a drag can move a full-width card after a smaller grid card', () => {
    const items = ref(['topology', 'status', 'dns'])
    const reorder = useDragReorder(items, async () => {})
    reorder.syncKeys(items.value.length)
    reorder.start()
    const arm = reorder.handleAttrs(0).onPointerdown as () => void
    arm()
    const start = reorder.rowAttrs(0).onDragstart as (event: unknown) => void
    start({ dataTransfer: { setData() {} }, preventDefault() {} })
    const over = reorder.rowAttrs(2).onDragover as (event: unknown) => void
    over({ clientY: 90, currentTarget: { getBoundingClientRect: () => ({ top: 0, height: 100 }) }, preventDefault() {} })
    expect(items.value).toEqual(['status', 'dns', 'topology'])
    reorder.cancel()
    expect(items.value).toEqual(['topology', 'status', 'dns'])
  })

  test('an arrange surface arms the whole item but ignores its controls', () => {
    const items = ref(['topology', 'status'])
    const reorder = useDragReorder(items, async () => {})
    reorder.syncKeys(items.value.length)
    reorder.start()

    const surface = reorder.surfaceAttrs(0)
    const pointerDown = surface.onPointerdown as (event: unknown) => void
    const pointerUp = surface.onPointerup as () => void

    pointerDown({ target: { closest: () => null } })
    expect(reorder.rowAttrs(0).draggable).toBe(true)

    pointerUp()
    expect(reorder.rowAttrs(0).draggable).toBe(false)

    pointerDown({
      target: {
        closest: (selector: string) => selector === '[data-reorder-control]' ? {} : null,
      },
    })
    expect(reorder.rowAttrs(0).draggable).toBe(false)
  })

  test('pressing and holding a card surface enters arrangement mode', async () => {
    const items = ref(['topology', 'status'])
    const reorder = useDragReorder(items, async () => {}, { longPressMs: 5 })
    reorder.syncKeys(items.value.length)

    const surface = reorder.activationAttrs(0)
    const pointerDown = surface.onPointerdown as (event: unknown) => void

    pointerDown({
      button: 0,
      pointerId: 1,
      clientX: 24,
      clientY: 40,
      target: { closest: () => null },
    })
    expect(reorder.holdingIndex.value).toBe(0)

    await Bun.sleep(10)
    expect(reorder.enabled.value).toBe(true)
    expect(reorder.holdingIndex.value).toBeNull()
    expect(reorder.rowAttrs(0).draggable).toBe(true)
  })

  test('releasing or moving before the hold delay preserves normal card interaction', async () => {
    const items = ref(['topology', 'status'])
    const reorder = useDragReorder(items, async () => {}, { longPressMs: 10 })
    reorder.syncKeys(items.value.length)

    const releasedSurface = reorder.activationAttrs(0)
    ;(releasedSurface.onPointerdown as (event: unknown) => void)({
      button: 0,
      pointerId: 1,
      clientX: 10,
      clientY: 10,
      target: { closest: () => null },
    })
    ;(releasedSurface.onPointerup as (event: unknown) => void)({ pointerId: 1 })
    await Bun.sleep(15)
    expect(reorder.enabled.value).toBe(false)

    const movedSurface = reorder.activationAttrs(0)
    ;(movedSurface.onPointerdown as (event: unknown) => void)({
      button: 0,
      pointerId: 2,
      clientX: 10,
      clientY: 10,
      target: { closest: () => null },
    })
    ;(movedSurface.onPointermove as (event: unknown) => void)({
      pointerId: 2,
      clientX: 24,
      clientY: 10,
    })
    await Bun.sleep(15)
    expect(reorder.enabled.value).toBe(false)
  })

  test('pressing an interactive card control never starts the hold gesture', async () => {
    const items = ref(['topology', 'status'])
    const reorder = useDragReorder(items, async () => {}, { longPressMs: 5 })
    reorder.syncKeys(items.value.length)

    const surface = reorder.activationAttrs(0)
    ;(surface.onPointerdown as (event: unknown) => void)({
      button: 0,
      pointerId: 1,
      clientX: 0,
      clientY: 0,
      target: { closest: () => ({ tagName: 'BUTTON' }) },
    })

    await Bun.sleep(10)
    expect(reorder.enabled.value).toBe(false)
    expect(reorder.holdingIndex.value).toBeNull()
  })
})
