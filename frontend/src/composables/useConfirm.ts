import { reactive, readonly } from 'vue'

/**
 * Promise-based confirmation dialog, a drop-in replacement for the native
 * `window.confirm()`. A single <ConfirmDialog /> instance (mounted globally in
 * App.vue) renders the shared state below; calling `confirm(options)` opens it
 * and resolves to `true`/`false` once the user picks an action.
 *
 * Usage in a component:
 *   const { confirm } = useConfirm()
 *   if (!(await confirm({ message: t('...'), tone: 'danger' }))) return
 *
 * The state is a module-level singleton so any caller shares the one dialog.
 */

export type ConfirmTone = 'danger' | 'primary'

export interface ConfirmOptions {
  /** Body text shown to the user. */
  message: string
  /** Header text. Falls back to a generic "Please confirm" in the dialog. */
  title?: string
  /** Confirm button label. Falls back to a generic "Confirm". */
  confirmLabel?: string
  /** Cancel button label. Falls back to a generic "Cancel". */
  cancelLabel?: string
  /** Visual emphasis of the confirm button. Destructive actions use 'danger'. */
  tone?: ConfirmTone
}

interface ConfirmState extends ConfirmOptions {
  visible: boolean
}

// Singleton state shared by the global <ConfirmDialog /> and every caller.
const state = reactive<ConfirmState>({
  visible: false,
  message: '',
  title: undefined,
  confirmLabel: undefined,
  cancelLabel: undefined,
  tone: 'primary',
})

// The resolver for the in-flight confirm() promise, captured when opened and
// invoked exactly once on confirm/cancel. Null when no dialog is open.
let resolver: ((value: boolean) => void) | null = null

function settle(result: boolean) {
  // Guard against double-settle (e.g. confirm click + mask close racing).
  if (!resolver) return
  const resolve = resolver
  resolver = null
  state.visible = false
  resolve(result)
}

function confirm(options: ConfirmOptions): Promise<boolean> {
  // If a previous dialog is still resolving, cancel it before reusing state.
  if (resolver) settle(false)

  state.message = options.message
  state.title = options.title
  state.confirmLabel = options.confirmLabel
  state.cancelLabel = options.cancelLabel
  state.tone = options.tone ?? 'primary'
  state.visible = true

  return new Promise<boolean>((resolve) => {
    resolver = resolve
  })
}

function handleConfirm() {
  settle(true)
}

function handleCancel() {
  settle(false)
}

export function useConfirm() {
  return {
    // Read-only view of the shared state for the global dialog component.
    state: readonly(state),
    confirm,
    handleConfirm,
    handleCancel,
  }
}
