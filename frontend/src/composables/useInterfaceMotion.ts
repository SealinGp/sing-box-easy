import { inject, provide, type InjectionKey } from 'vue'

// Shared interaction policy for the dashboard and /lab.
// Vue injection follows component ownership, including teleported overlays.
const interfaceMotionKey: InjectionKey<boolean> = Symbol('interface-motion')

export function provideInterfaceMotion(enabled = true) {
  provide(interfaceMotionKey, enabled)
}

export function useInterfaceMotion() {
  return inject(interfaceMotionKey, true)
}

// Entry and exit share their geometry. The reverse curve is defined in CSS.
export const popoverMotion = {
  enterFromClass: 'ui-popover-enter-from',
  enterActiveClass: 'ui-popover-enter-active',
  enterToClass: 'ui-popover-enter-to',
  leaveFromClass: 'ui-popover-leave-from',
  leaveActiveClass: 'ui-popover-leave-active',
  leaveToClass: 'ui-popover-leave-to',
}

export const dialogMotion = {
  enterFromClass: 'ui-dialog-enter-from',
  enterActiveClass: 'ui-dialog-enter-active',
  enterToClass: 'ui-dialog-enter-to',
  leaveFromClass: 'ui-dialog-leave-from',
  leaveActiveClass: 'ui-dialog-leave-active',
  leaveToClass: 'ui-dialog-leave-to',
}
