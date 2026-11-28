export interface WorkspaceTab { path: string; pinned: boolean }
export function restoreTabs(raw: string | null, allowed: (path: string) => boolean): WorkspaceTab[] {
  try {
    const data: unknown = JSON.parse(raw ?? '[]')
    if (!Array.isArray(data)) return []
    const seen = new Set<string>()
    return data.filter((tab): tab is WorkspaceTab => {
      if (!tab || typeof tab.path !== 'string' || !allowed(tab.path) || seen.has(tab.path)) return false
      seen.add(tab.path)
      return true
    }).map(tab => ({ path: tab.path, pinned: tab.pinned === true }))
  } catch { return [] }
}
export function nextTab(tabs: WorkspaceTab[], removed: Set<string>, current: string) {
  if (!removed.has(current)) return current
  const index = tabs.findIndex(tab => tab.path === current)
  return tabs.slice(0, index).reverse().find(tab => !removed.has(tab.path))?.path
    ?? tabs.slice(index + 1).find(tab => !removed.has(tab.path))?.path
    ?? '/dashboard/overview'
}
