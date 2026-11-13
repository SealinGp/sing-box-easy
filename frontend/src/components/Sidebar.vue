<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { ChevronDownIcon, MagnifyingGlassIcon } from '@heroicons/vue/24/outline'
import type { MenuGroup } from '../navigation/menu'
import { isMenuActive } from '../navigation/menu'
import NavUtilities from './NavUtilities.vue'
const props = defineProps<{ menuItems: MenuGroup[] }>()
defineEmits<{ search: [] }>()
const route = useRoute()
const { t } = useI18n()
const groups = ref<HTMLElement | null>(null)
// These are the existing tab destinations, also reachable from their section.
const subpages = computed(() => {
  const pages = (section: string, entries: [string, string][]) => entries.map(([path, key]) => ({ path: `/dashboard/${section}/${path}`, name: t(key) }))
  return {
    '/dashboard/lab': [{ path: '/dashboard/lab/functionality', name: t('nav.labFunctionality') }, { path: '/dashboard/lab/theme', name: t('nav.labTheme') }],
    '/dashboard/dns': pages('dns', [['groups', 'lab.resolverGroups'], ['servers', 'dns.tabs.servers'], ['rules', 'dns.tabs.rules'], ['settings', 'dns.tabs.settings']]),
    '/dashboard/route': pages('route', [['policies', 'lab.trafficPolicies'], ['rules', 'route.tabs.rules'], ['rule-sets', 'route.tabs.ruleSets'], ['final-policy', 'route.tabs.finalPolicy']]),
    '/dashboard/diagnostics': pages('diagnostics', [['dual-stack', 'diagnostics.tabs.dualStack'], ['dns', 'diagnostics.tabs.dns'], ['route', 'diagnostics.tabs.route']]),
    '/dashboard/experimental': pages('experimental', [['cache-file', 'experimental.tabs.cacheFile'], ['clash-api', 'experimental.tabs.clashApi'], ['v2ray-api', 'experimental.tabs.v2rayApi']]),
  } as Record<string, { path: string; name: string }[]>
})
const activeSection = () => props.menuItems.flatMap(group => group.items).find(item => subpages.value[item.path] && isMenuActive(item.path, route.path))?.path ?? null
const expanded = ref<string | null>(activeSection())
const motionReady = ref(false)
const keepCurrentVisible = () => groups.value?.querySelector<HTMLElement>('[aria-current="page"]')?.scrollIntoView({ block: 'nearest', behavior: 'instant' })
function onSubnavTransitionEnd(event: TransitionEvent) {
  if (event.propertyName === 'grid-template-rows' && event.target instanceof HTMLElement && event.target.classList.contains('is-open') && event.target.querySelector('[aria-current="page"]')) keepCurrentVisible()
}
const sectionId = (path: string) => `sidebar-pages-${path.split('/').pop()}`
const isCurrent = (path: string) => isMenuActive(path, route.path) && (!subpages.value[path] || expanded.value !== path)
watch(() => route.path, () => { expanded.value = activeSection() })
watch(expanded, (next, previous) => {
  // A closed submenu must not retain keyboard focus inside its inert content.
  const closing = previous ? document.getElementById(sectionId(previous)) : null
  if (closing?.contains(document.activeElement)) {
    const target = next ?? previous
    groups.value?.querySelector<HTMLElement>(`[data-sidebar-path="${target}"]`)?.focus({ preventScroll: true })
  }
  if (next && next === activeSection() && window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    void nextTick(() => { if (expanded.value === next) keepCurrentVisible() })
  }
}, { flush: 'pre' })
let readyFrame = 0
let viewportObserver: ResizeObserver | undefined
onMounted(() => {
  keepCurrentVisible()
  viewportObserver = new ResizeObserver(keepCurrentVisible)
  if (groups.value) viewportObserver.observe(groups.value)
  // First paint is already at the current route, with no entrance replay.
  readyFrame = requestAnimationFrame(() => { motionReady.value = true })
})
onBeforeUnmount(() => { cancelAnimationFrame(readyFrame); viewportObserver?.disconnect() })
</script>

<template>
  <aside class="navigation-sidebar" :class="{ 'sidebar-motion-ready': motionReady }">
    <router-link to="/dashboard/overview" class="nav-brand">
      <img src="/logo.svg" alt="" width="30" height="30" />
      <span>Sing Box Easy<span class="nav-brand-caption">{{ $t('nav.workspace') }}</span></span>
    </router-link>
    <button class="nav-search-trigger" aria-keyshortcuts="Control+K Meta+K" @click="$emit('search')">
      <MagnifyingGlassIcon class="h-4 w-4 shrink-0" />
      <span>{{ $t('nav.search') }}</span><kbd>⌘ / Ctrl K</kbd>
    </button>
    <nav ref="groups" class="sidebar-groups" :aria-label="$t('nav.navigation')">
      <section v-for="group in menuItems" :key="group.id" :aria-labelledby="'sidebar-' + group.id">
        <h2 :id="'sidebar-' + group.id" class="nav-group-label">{{ group.name }}</h2>
        <ul>
          <li v-for="item in group.items" :key="item.path">
            <div class="sidebar-item-row" :class="{ 'is-current': isCurrent(item.path) }">
              <router-link :to="item.path" class="nav-destination" :class="{ selected: isCurrent(item.path), 'is-section-active': isMenuActive(item.path, route.path) }"
                :data-sidebar-path="item.path" :aria-current="isCurrent(item.path) ? 'page' : undefined">
                <component :is="item.icon" class="sidebar-item-icon h-4 w-4 shrink-0" aria-hidden="true" /><span>{{ item.name }}</span>
              </router-link>
              <button v-if="subpages[item.path]" type="button" class="sidebar-disclosure" :aria-expanded="expanded === item.path"
                :aria-controls="sectionId(item.path)" :aria-label="t(expanded === item.path ? 'nav.collapsePages' : 'nav.expandPages', { name: item.name })"
                @click="expanded = expanded === item.path ? null : item.path">
                <ChevronDownIcon class="h-3.5 w-3.5" aria-hidden="true" />
              </button>
            </div>
            <div v-if="subpages[item.path]" :id="sectionId(item.path)" class="sidebar-subnav" :class="{ 'is-open': expanded === item.path }"
              :style="{ '--sidebar-child-count': subpages[item.path]!.length }" :inert="expanded !== item.path" :aria-hidden="expanded !== item.path" @transitionend="onSubnavTransitionEnd">
              <div class="sidebar-subnav-clip">
                <ul class="sidebar-subnav-list">
                  <li v-for="(child, index) in subpages[item.path]" :key="child.path" class="sidebar-child" :style="{ '--sidebar-child-index': index }">
                    <router-link :to="child.path" class="nav-destination sidebar-child-link" :class="{ selected: isMenuActive(child.path, route.path) }"
                      :aria-current="expanded === item.path && isMenuActive(child.path, route.path) ? 'page' : undefined">
                      <span>{{ child.name }}</span>
                    </router-link>
                  </li>
                </ul>
              </div>
            </div>
          </li>
        </ul>
      </section>
    </nav>
    <NavUtilities />
  </aside>
</template>
