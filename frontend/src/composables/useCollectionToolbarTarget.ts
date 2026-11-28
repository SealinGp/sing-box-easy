import { onMounted, ref, watch, type Ref } from 'vue'

/** Place collection controls beside the nearest existing page/panel actions. */
export function useCollectionToolbarTarget(root: Ref<HTMLElement | null>) {
  const target = ref<HTMLElement | null>(null)
  function locate() {
    let current = root.value
    const boundary = current?.closest('[role="dialog"], #dashboard-content')
    while (current && current !== boundary) {
      let previous = current.previousElementSibling
      while (previous) {
        // Never attach to another collection or to an editable form.
        if (!previous.matches('.collection-records, form') && !previous.querySelector('.collection-records, form')) {
          const buttons = [...previous.querySelectorAll<HTMLButtonElement>('button')]
            .filter(button => !button.closest('[role="dialog"], .collection-toolbar'))
          const last = buttons.at(-1)
          const actions = last?.parentElement
          if (actions && getComputedStyle(actions).display.includes('flex')) {
            target.value = actions
            return
          }
        }
        previous = previous.previousElementSibling
      }
      current = current.parentElement
    }
    target.value = null
  }
  onMounted(locate)
  watch(root, locate, { flush: 'post' })
  return target
}
