<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { XMarkIcon, ChevronDownIcon, BookmarkIcon } from '@heroicons/vue/24/outline'
import Popover from '../volt/Popover.vue'
import { useWorkspaceDrag } from '../composables/useWorkspaceDrag'
import type { WorkspaceTab } from '../utils/workspaceTabs'
import { workspaceLabel } from '../navigation/workspaceLabels'
const props = defineProps<{ tabs: WorkspaceTab[]; current: string; dirty: Set<string> }>()
const emit = defineEmits<{ open: [path: string]; close: [path: string]; pin: [path: string]; closeOthers: [path: string]; closeAll: []; reorder: [from: string, to: string, before: boolean] }>()
const { t } = useI18n()
const nav = ref<HTMLElement>()
const menu = ref<InstanceType<typeof Popover>>()
const contextPath = ref('')
const contextOnly = ref(false)
const { dragging, target, before, offset, start, consumeClick } = useWorkspaceDrag(nav, (from, to, before) => emit('reorder', from, to, before))
function openMenu(event: MouseEvent, path: string, context = true) {
  contextPath.value = path; contextOnly.value = context
  if (event.currentTarget instanceof HTMLElement) menu.value?.showAt(event.currentTarget)
}
function activate(path: string) { if (!consumeClick()) emit('open', path) }

watch(() => props.current, async () => {
  await nextTick()
  nav.value?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
})
function move(event: KeyboardEvent, index: number) {
  const step = event.key === 'ArrowRight' ? 1 : event.key === 'ArrowLeft' ? -1 : 0
  if (!step && event.key !== 'Home' && event.key !== 'End') return
  event.preventDefault()
  if (event.altKey && step) {
    const from = props.tabs[index], to = props.tabs[index + step]
    if (from && to) emit('reorder', from.path, to.path, step < 0)
    return
  }
  const target = event.key === 'Home' ? 0 : event.key === 'End' ? props.tabs.length - 1 : (index + step + props.tabs.length) % props.tabs.length
  const tab = props.tabs[target]
  if (tab) { emit('open', tab.path); nav.value?.querySelectorAll<HTMLElement>('[role="tab"]')[target]?.focus() }
}
</script>
<template>
  <div class="workspace-tabs">
    <nav ref="nav" class="workspace-tab-strip" role="tablist" :aria-label="t('nav.workTabs')">
      <div v-for="(tab, index) in tabs" :key="tab.path" class="workspace-tab" :data-workspace-path="tab.path"
        :class="{ 'is-active': tab.path === current, 'is-dragging': dragging === tab.path, 'drop-before': target === tab.path && before, 'drop-after': target === tab.path && !before }"
        :style="dragging === tab.path ? { transform: `translateX(${offset}px)` } : undefined"
        @contextmenu.prevent="openMenu($event, tab.path)">
        <button type="button" role="tab" :id="`work-tab-${index}`" :aria-controls="`work-pane-${index}`"
          :aria-selected="tab.path === current" :tabindex="tab.path === current ? 0 : -1"
          :title="workspaceLabel(tab.path, t)" @click="activate(tab.path)" @pointerdown="start($event, tab.path)" @keydown="move($event, index)"
          @keydown.shift.f10.prevent="openMenu($event as unknown as MouseEvent, tab.path)">
          <BookmarkIcon v-if="tab.pinned" class="h-3 w-3 shrink-0" :aria-label="t('nav.pinned')" />
          <span class="truncate">{{ workspaceLabel(tab.path, t) }}</span>
          <span v-if="dirty.has(tab.path)" class="workspace-dirty" :aria-label="t('nav.unsaved')" :title="t('nav.unsaved')">●</span>
        </button>
        <button v-if="!tab.pinned && tabs.length > 1" type="button" class="workspace-tab-close" :aria-label="t('nav.closeTab', { name: workspaceLabel(tab.path, t) })"
          :title="t('nav.closeTab', { name: workspaceLabel(tab.path, t) })" @click="emit('close', tab.path)"><XMarkIcon class="h-3.5 w-3.5" /></button>
      </div>
    </nav>
    <button type="button" class="workspace-tab-menu" :aria-label="t('nav.workTabs')" :title="t('nav.workTabs')" @click="openMenu($event, current, false)"><ChevronDownIcon class="h-4 w-4" /></button>
    <Popover ref="menu" interaction="menu" panel-class="workspace-actions w-72" content-class="p-2">
      <div v-if="!contextOnly" class="max-h-[50dvh] overflow-auto">
        <button v-for="tab in tabs" :key="tab.path" type="button" class="workspace-menu-item" :aria-current="tab.path === current ? 'page' : undefined"
          @click="emit('open', tab.path); menu?.hide()"><span class="truncate">{{ workspaceLabel(tab.path, t) }}</span><span v-if="dirty.has(tab.path)">●</span><BookmarkIcon v-if="tab.pinned" class="h-3 w-3" /></button>
      </div>
      <div class="mt-1 border-t border-border pt-1">
        <button type="button" class="workspace-menu-item" @click="emit('pin', contextPath); menu?.hide()">{{ t(tabs.find(tab => tab.path === contextPath)?.pinned ? 'nav.unpinTab' : 'nav.pinTab') }}</button>
        <button type="button" class="workspace-menu-item" @click="emit('close', contextPath); menu?.hide()">{{ t('nav.closeThisTab') }}</button>
        <button type="button" class="workspace-menu-item" :disabled="tabs.length < 2" @click="emit('closeOthers', contextPath); menu?.hide()">{{ t('nav.closeOtherTabs') }}</button>
        <button type="button" class="workspace-menu-item" @click="emit('closeAll'); menu?.hide()">{{ t('nav.closeAllTabs') }}</button>
      </div>
    </Popover>
  </div>
</template>
