import { inject, onBeforeUnmount, ref, watch, type InjectionKey, type Ref } from 'vue'

export interface WorkspaceContext {
  active: Readonly<Ref<boolean>>
  dirty: Map<symbol, boolean>
  inlineDirty: Ref<boolean>
  editVersion: Ref<number>
}
export const workspaceKey: InjectionKey<WorkspaceContext> = Symbol('workspace')
export function useWorkspaceActive() { return inject(workspaceKey, null)?.active ?? ref(true) }
/** Drafts live in memory only; tab metadata never contains form values. */
export function useWorkspaceDirty(value: () => boolean) {
  const workspace = inject(workspaceKey, null)
  const key = Symbol('draft')
  if (!workspace) return
  watch(value, dirty => workspace.dirty.set(key, dirty), { immediate: true, flush: 'sync' })
  onBeforeUnmount(() => workspace.dirty.delete(key))
}

export const workspaceEditKey: InjectionKey<(target?: Element | null) => void> = Symbol('workspace-edit')
export function useWorkspaceEdit() { return inject(workspaceEditKey, () => {}) }
