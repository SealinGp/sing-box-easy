<script setup lang="ts">
import { markRaw, nextTick, onBeforeUnmount, reactive, shallowReactive, watch } from 'vue'
import { useRoute, useRouter, type RouteLocationNormalizedLoaded } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useConfirm } from '../composables/useConfirm'
import { nextTab, restoreTabs } from '../utils/workspaceTabs'
import WorkspaceTabs from './WorkspaceTabs.vue'
import WorkspacePane from './WorkspacePane.vue'
import '../style/workspace.css'
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const { confirm } = useConfirm()
const storageKey = 'sbe-workspace-tabs-v1'
const allowed = (path: string) => path.startsWith('/dashboard/') && !path.includes('?') && !path.includes('#') && router.resolve(path).matched.length > 1 && !router.resolve(path).matched.at(-1)?.redirect
let stored: string | null = null
try { stored = sessionStorage.getItem(storageKey) } catch { /* storage is optional */ }
const tabs = reactive(restoreTabs(stored, allowed))
const pages = shallowReactive(new Map<string, RouteLocationNormalizedLoaded>())
const dirty = reactive(new Set<string>())
const ordered = tabs
watch(() => route.fullPath, () => {
  if (!allowed(route.path)) return
  if (!tabs.some(tab => tab.path === route.path)) tabs.push({ path: route.path, pinned: false })
  pages.set(route.path, markRaw({ ...route }))
}, { immediate: true })
watch(tabs, () => {
  // Persist destinations only. Query strings, draft content and credentials never go into storage.
  try { sessionStorage.setItem(storageKey, JSON.stringify(tabs)) } catch { /* optional */ }
}, { deep: true, immediate: true })
function open(path: string) { void router.push(pages.get(path)?.fullPath ?? path) }
function setDirty(path: string, value: boolean) { if (value) dirty.add(path); else dirty.delete(path) }
let closing = false
async function close(paths: string[]) {
  if (closing || !paths.length) return
  closing = true
  try {
    if (paths.some(path => dirty.has(path)) && !await confirm({ message: t('nav.discardTabs'), confirmLabel: t('nav.discardAndClose'), tone: 'danger' })) return
    const removed = new Set(paths)
    const next = nextTab(ordered, removed, route.path)
    if (next !== route.path) {
      const failure = await router.push(pages.get(next)?.fullPath ?? next)
      if (failure) return
    }
    for (let i = tabs.length - 1; i >= 0; i--) if (removed.has(tabs[i]!.path)) tabs.splice(i, 1)
    for (const path of paths) { pages.delete(path); dirty.delete(path) }
    // Closing the last tab returns to a fresh overview, never an empty workspace.
    if (!tabs.length) {
      await nextTick()
      tabs.push({ path: '/dashboard/overview', pinned: false })
      pages.set('/dashboard/overview', markRaw({ ...router.currentRoute.value }))
    }
  } finally { closing = false }
}
function reorder(from: string, to: string, before: boolean) {
  const source = tabs.findIndex(tab => tab.path === from)
  if (source < 0 || from === to) return
  const [tab] = tabs.splice(source, 1)
  const destination = tabs.findIndex(tab => tab.path === to)
  if (tab) tabs.splice(destination + (before ? 0 : 1), 0, tab)
}
function pin(path: string) { const tab = tabs.find(tab => tab.path === path); if (tab) tab.pinned = !tab.pinned }
function beforeUnload(event: BeforeUnloadEvent) { if (dirty.size) { event.preventDefault(); event.returnValue = '' } }
window.addEventListener('beforeunload', beforeUnload)
const removeGuard = router.beforeEach(async to => {
  if (to.path.startsWith('/dashboard')) return true
  if (dirty.size && !await confirm({ message: t('nav.discardTabs'), confirmLabel: t('nav.discardAndClose'), tone: 'danger' })) return false
  try { sessionStorage.removeItem(storageKey) } catch { /* optional */ }
  return true
})
onBeforeUnmount(() => { window.removeEventListener('beforeunload', beforeUnload); removeGuard() })
</script>
<template>
  <div class="workspace">
    <WorkspaceTabs :tabs="ordered" :current="route.path" :dirty="dirty" @open="open" @close="path => close([path])" @pin="pin"
      @close-others="path => close(tabs.filter(tab => tab.path !== path).map(tab => tab.path))" @close-all="close(tabs.map(tab => tab.path))" @reorder="reorder" />
    <template v-for="(tab, index) in ordered" :key="tab.path">
      <WorkspacePane v-if="pages.has(tab.path)" :id="`work-pane-${index}`" role="tabpanel" :aria-labelledby="`work-tab-${index}`"
        :route="pages.get(tab.path)!" :active="route.path === tab.path" @dirty="setDirty(tab.path, $event)" />
    </template>
  </div>
</template>
