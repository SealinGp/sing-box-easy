<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { GlobeAltIcon, MapIcon, BeakerIcon, UserCircleIcon } from '@heroicons/vue/24/outline'
import Sidebar from '../components/Sidebar.vue'
import Topbar from '../components/Topbar.vue'
import NavigationSearch from '../components/NavigationSearch.vue'
import { useDeployment } from '../composables/useDeployment'
import { createMenu, type MenuItem } from '../navigation/menu'
import '../style/navigation.css'

const { t } = useI18n()
const { prefersTopbar, authEnabled } = useDeployment()
const menuItems = computed(() => createMenu(t, authEnabled.value))
const searchGroups = computed(() => {
  const deep = (parent: string, keys: string[], paths: string[], icon: MenuItem['icon']) =>
    keys.map((key, index) => ({ name: t(parent) + ' · ' + t(key), path: '/dashboard/' + paths[index], icon }))
  return [...menuItems.value, { id: 'pages', name: t('nav.pages'), items: [
    ...deep('nav.dns', ['dns.tabs.servers', 'dns.tabs.rules', 'dns.tabs.settings', 'dns.tabs.diagnostics'],
      ['dns/servers', 'dns/rules', 'dns/settings', 'dns/diagnostics'], GlobeAltIcon),
    ...deep('nav.route', ['route.tabs.rules', 'route.tabs.ruleSets', 'route.tabs.finalPolicy', 'route.tabs.diagnostics'],
      ['route/rules', 'route/rule-sets', 'route/final-policy', 'route/diagnostics'], MapIcon),
    ...deep('nav.experimental', ['experimental.tabs.cacheFile', 'experimental.tabs.clashApi', 'experimental.tabs.v2rayApi'],
      ['experimental/cache-file', 'experimental/clash-api', 'experimental/v2ray-api'], BeakerIcon),
    ...(authEnabled.value ? [{ name: t('nav.profile'), path: '/dashboard/profile', icon: UserCircleIcon }] : []),
  ] }]
})
const searchOpen = ref(false)
const media = window.matchMedia('(max-width: 1023px)')
const narrow = ref(media.matches)
const useTopbar = computed(() => prefersTopbar.value || narrow.value)
function resize() { narrow.value = media.matches }
function shortcut(event: KeyboardEvent) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k' && !event.altKey) {
    // Keep editor shortcuts and existing dialogs local to their own workflow.
    if (!searchOpen.value && (document.querySelector('[role="dialog"], dialog[open]') || (event.target instanceof Element && event.target.closest('.monaco-editor')))) return
    event.preventDefault()
    searchOpen.value = !searchOpen.value
  }
}
onMounted(() => {
  media.addEventListener('change', resize)
  window.addEventListener('keydown', shortcut)
})
onBeforeUnmount(() => {
  media.removeEventListener('change', resize)
  window.removeEventListener('keydown', shortcut)
})
</script>

<template>
  <div class="liquid-app dashboard-shell" :class="{ 'horizontal-shell': useTopbar }">
    <a href="#dashboard-content" class="nav-skip-link">{{ $t('nav.skipContent') }}</a>
    <Topbar v-if="useTopbar" :menu-items="menuItems" @search="searchOpen = true" />
    <Sidebar v-else :menu-items="menuItems" @search="searchOpen = true" />
    <main id="dashboard-content" tabindex="-1" class="dashboard-content"><router-view /></main>
    <NavigationSearch v-model="searchOpen" :groups="searchGroups" />
  </div>
</template>
