import { computed, ref, shallowRef, type Ref } from 'vue'

/**
 * Drag-to-reorder for an index-addressed list, shared by the route rules
 * (`<List>`) and the DNS rules (`<Table>`).
 *
 * Both lists are evaluated top-down by sing-box, so their order IS policy — a
 * specific rule placed under a broad one never runs. Reordering used to mean
 * deleting and re-adding rules in the right sequence.
 *
 * Two levels of "not yet committed", which is the whole shape of this thing:
 *
 *   1. WITHIN a drag, the list sorts LIVE: as the pointer crosses a neighbour,
 *      the carried row takes its slot and the neighbour slides into the
 *      vacated one, so the gap under the cursor always shows where the row
 *      will land.
 *   2. ACROSS drags, nothing is sent. Reorder mode is a batch edit — drag as
 *      many rows as you like, then `save()` writes ONE permutation. Every
 *      write to config.json costs a validation pass, a version snapshot and a
 *      file swap, so a five-row cleanup should not be five of those.
 *
 * `cancel()` throws the session away and restores the list as it was when the
 * mode was entered.
 *
 * Usage — the two `*Attrs` helpers carry every DOM handler, so a call site
 * only decides WHERE the handle and the row are:
 *
 *     const reorder = useDragReorder(rules, persistOrder)
 *     …
 *     <tr v-for="(r, i) in rules" :key="reorder.keyAt(i)" v-bind="reorder.rowAttrs(i)">
 *       <td><button v-bind="reorder.handleAttrs(i)">☰</button></td>
 *
 * @param items   The rendered list. Reordered in place (immutably) as the
 *                pointer moves, so the caller's template needs no drag state.
 * @param persist Sends a permutation of the SERVER's current indices. It owns
 *                success/failure reporting; rejecting is enough to tell this
 *                composable to stop.
 */
export function useDragReorder<T>(items: Ref<T[]>, persist: (order: number[]) => Promise<void>) {
  /** Reorder is a MODE: handles appear, per-row edit/delete gets out of the way. */
  const enabled = ref(false)

  /**
   * Stable per-row identity, held alongside the items.
   *
   * Rows keyed by array index make reordering invisible to Vue's diff: it
   * patches each row's text in place, no element ever moves, and a FLIP
   * transition has nothing to animate. These keys travel WITH the row through
   * every permutation, so moving row 5 to the top moves that DOM node.
   */
  const keys = ref<number[]>([])

  /** Live index of the row being carried, and the row armed for dragging. */
  const dragPos = ref<number | null>(null)
  const armedIndex = ref<number | null>(null)

  /**
   * Everything the session has done so far, expressed in the SERVER's indices
   * — the single permutation `save()` sends. Null outside reorder mode.
   *
   * Accumulated rather than recomputed: each move composes onto it, so ten
   * drags still add up to one array the server can apply in one pass.
   */
  const sessionOrder = ref<number[] | null>(null)

  /**
   * The list as it was when the mode was entered, for `cancel()`.
   *
   * `shallowRef` because a deep ref would unwrap `T` into `UnwrapRefSimple<T>`
   * — a snapshot needs to hold the rows exactly as they are, not a reactive
   * rewrite of them.
   */
  const snapshot = shallowRef<{ items: T[]; keys: number[] } | null>(null)

  const saving = ref(false)

  /** Has the session actually moved anything? Drives the Save/Done label. */
  const dirty = computed(() => !!sessionOrder.value && !isIdentity(sessionOrder.value))

  function clearDrag() {
    dragPos.value = null
    armedIndex.value = null
  }

  function clearSession() {
    clearDrag()
    sessionOrder.value = null
    snapshot.value = null
  }

  /** Enters reorder mode, baselining the session against what is on screen. */
  function start() {
    enabled.value = true
    sessionOrder.value = items.value.map((_, i) => i)
    snapshot.value = { items: [...items.value], keys: [...keys.value] }
  }

  /**
   * Leaves reorder mode, sending the accumulated permutation if there is one.
   *
   * Exits either way — on failure `persist` refetches, so the list on screen
   * is the config's, and staying in a mode whose pending state was just thrown
   * away would only be confusing.
   */
  async function save() {
    const order = sessionOrder.value
    enabled.value = false
    clearSession()
    if (!order || isIdentity(order)) return
    await send(order)
  }

  /** Leaves reorder mode, restoring the list as it was on entry. */
  function cancel() {
    const restore = snapshot.value
    if (restore) {
      items.value = restore.items
      keys.value = restore.keys
    }
    enabled.value = false
    clearSession()
  }

  /**
   * Reuses keys positionally so a refetch does not re-key the whole list — an
   * edit would otherwise unmount and remount every row. After an add the extra
   * row is the only one that gets a fresh key; after a delete it truncates.
   *
   * Call it wherever the list is (re)loaded. A load replaces the rows
   * wholesale, so any session in progress is rebaselined against it.
   */
  function syncKeys(count: number) {
    const previous = keys.value
    keys.value = Array.from({ length: count }, (_, i) => previous[i] ?? nextKey())
    if (enabled.value) start()
  }

  function keyAt(index: number): number {
    return keys.value[index] ?? index
  }

  function isIdentity(order: number[]): boolean {
    return order.every((value, i) => value === i)
  }

  /**
   * Applies a permutation of the CURRENTLY rendered rows, composing it onto
   * the session.
   *
   * The composition is what lets a second drag build on the first: `view` says
   * where each on-screen row goes, `sessionOrder` says which server row each
   * on-screen row is, so reading the latter through the former keeps it
   * expressed in server indices throughout.
   */
  function applyView(view: number[]) {
    const currentItems = items.value
    const currentKeys = keys.value
    const session = sessionOrder.value

    items.value = view.map((i) => currentItems[i] as T)
    keys.value = view.map((i) => currentKeys[i] as number)
    if (session) sessionOrder.value = view.map((i) => session[i] as number)
  }

  /** Moves the row at `from` to position `target`, both in view coordinates. */
  function moveInView(from: number, target: number) {
    const view = items.value.map((_, i) => i)
    const [moved] = view.splice(from, 1)
    view.splice(target, 0, moved as number)
    applyView(view)
  }

  /**
   * HTML5 drag-and-drop has no notion of a handle: whatever carries
   * `draggable` is what you can grab. Arming on pointerdown over the handle —
   * and disarming when the drag ends — makes the handle the only grab point
   * while leaving the rest of the row selectable.
   */
  function arm(index: number) {
    if (enabled.value && !saving.value) armedIndex.value = index
  }

  function disarm() {
    if (dragPos.value === null) armedIndex.value = null
  }

  function onDragStart(index: number, event: DragEvent) {
    if (!enabled.value || armedIndex.value !== index) {
      event.preventDefault()
      return
    }
    // Firefox refuses to start a drag unless the payload is set. The value is
    // never read back — the order lives in `sessionOrder`.
    event.dataTransfer?.setData('text/plain', String(index))
    if (event.dataTransfer) event.dataTransfer.effectAllowed = 'move'
    dragPos.value = index
  }

  /**
   * The pointer is over row `index`. Move the carried row above or below it
   * depending on which half of the row the pointer is in.
   */
  function onDragOver(index: number, event: DragEvent) {
    if (!enabled.value) return
    // Without preventDefault the browser treats the row as a non-drop target
    // and fires no `drop` at all.
    event.preventDefault()
    if (event.dataTransfer) event.dataTransfer.dropEffect = 'move'

    const from = dragPos.value
    if (from === null) return

    const box = (event.currentTarget as HTMLElement).getBoundingClientRect()
    const after = event.clientY >= box.top + box.height / 2

    // `insertAt` is a slot in the pre-move list; removing the row first shifts
    // every later slot down by one, hence the decrement.
    const insertAt = after ? index + 1 : index
    const target = insertAt > from ? insertAt - 1 : insertAt
    if (target === from || target < 0 || target >= items.value.length) return

    moveInView(from, target)
    dragPos.value = target
    armedIndex.value = target
  }

  /**
   * Drag over. Keeps the result and sends nothing — the rows have been sliding
   * into their new places the whole way, and `save()` is what commits them.
   *
   * This also means a row dropped outside the list keeps its new position
   * rather than snapping back: treating that as a cancel would silently undo
   * what the user just watched happen, and `cancel()` is right there.
   */
  function onDragEnd() {
    clearDrag()
  }

  /** Keyboard path: ArrowUp / ArrowDown on a focused handle. */
  function nudge(index: number, delta: number) {
    if (!enabled.value || saving.value) return
    const target = index + delta
    if (target < 0 || target >= items.value.length) return
    moveInView(index, target)
  }

  async function send(order: number[]) {
    saving.value = true
    try {
      await persist(order)
    } catch {
      // `persist` owns the reporting — and the refetch that puts the rows back
      // in step with the config. Nothing useful to add here.
    } finally {
      saving.value = false
    }
  }

  /**
   * Everything the draggable row element needs. Empty off reorder mode, so a
   * `v-bind` of it adds nothing to the normal list.
   */
  function rowAttrs(index: number): Record<string, unknown> {
    if (!enabled.value) return {}
    return {
      draggable: armedIndex.value === index,
      class: dragPos.value === index ? 'is-dragging' : undefined,
      onDragstart: (event: DragEvent) => onDragStart(index, event),
      onDragover: (event: DragEvent) => onDragOver(index, event),
      // The list has been sorting itself on dragover, so the row is already
      // where it belongs. This only cancels the browser default, which would
      // try to navigate to the drag payload.
      onDrop: (event: DragEvent) => event.preventDefault(),
      onDragend: onDragEnd,
    }
  }

  /** Everything the handle button needs, including the keyboard fallback. */
  function handleAttrs(index: number): Record<string, unknown> {
    return {
      type: 'button',
      onPointerdown: () => arm(index),
      onPointerup: disarm,
      onKeydown: (event: KeyboardEvent) => {
        if (event.key !== 'ArrowUp' && event.key !== 'ArrowDown') return
        event.preventDefault()
        nudge(index, event.key === 'ArrowDown' ? 1 : -1)
      },
    }
  }

  return {
    enabled,
    saving,
    dirty,
    start,
    save,
    cancel,
    syncKeys,
    keyAt,
    rowAttrs,
    handleAttrs,
  }
}

/*
 * Module-scoped so two lists on one page cannot mint the same key — Vue only
 * requires keys to be unique per v-for, but a shared counter costs nothing and
 * removes the question.
 */
let keySeq = 0
function nextKey(): number {
  return ++keySeq
}
