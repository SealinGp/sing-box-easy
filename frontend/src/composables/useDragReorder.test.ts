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
})
