<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Bars3Icon, ChevronDownIcon, MagnifyingGlassIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import type { MenuGroup } from '../navigation/menu'
import { isMenuActive } from '../navigation/menu'
import NavUtilities from './NavUtilities.vue'

const props = defineProps<{ menuItems: MenuGroup[] }>()
defineEmits<{ search: [] }>()
const route = useRoute()
const primaryPaths = ['overview', 'outbounds/subscriptions', 'outbounds/list', 'inbounds', 'route', 'dns', 'logs'].map(path => '/dashboard/' + path)
const allItems = computed(() => props.menuItems.flatMap(group => group.items))
const primaryItems = computed(() => primaryPaths.flatMap(path => allItems.value.filter(item => item.path === path)))
const moreItems = computed(() => allItems.value.filter(item => !primaryPaths.includes(item.path)))
const activeMore = computed(() => moreItems.value.find(item => isMenuActive(item.path, route.path)))
const more = ref<HTMLDetailsElement | null>(null)
const drawer = ref<HTMLDialogElement | null>(null)
const drawerOpen = ref(false)
function closeNavigation() {
  if (more.value) more.value.open = false
  drawer.value?.close()
  drawerOpen.value = false
}
function openDrawer() {
  drawer.value?.showModal()
  drawerOpen.value = true
}
function dismissMore(event: PointerEvent) {
  if (event.target instanceof Node && !more.value?.contains(event.target) && more.value) more.value.open = false
}
function escapeMore(event: KeyboardEvent) {
  if (event.key === 'Escape' && more.value?.open) {
    more.value.open = false
    more.value.querySelector('summary')?.focus()
  }
}
watch(() => route.path, closeNavigation)
onMounted(() => document.addEventListener('pointerdown', dismissMore))
onBeforeUnmount(() => document.removeEventListener('pointerdown', dismissMore))
</script>

<template>
  <header class="navigation-topbar">
    <div class="topbar-brand-row">
      <div class="topbar-brand">
        <button class="nav-icon-button topbar-mobile-toggle" :aria-label="$t('nav.navigation')" :aria-expanded="drawerOpen" aria-controls="topbar-drawer" @click="openDrawer">
          <Bars3Icon class="h-5 w-5" />
        </button>
        <router-link to="/dashboard/overview" class="nav-brand" :aria-label="'Sing Box Easy · ' + $t('nav.overview')" :title="'Sing Box Easy · ' + $t('nav.overview')">
          <img src="/logo.jpg" alt="" width="28" height="28" />
        </router-link>
      </div>
      <nav class="topbar-desktop-nav" :aria-label="$t('nav.navigation')">
        <ul class="topbar-destinations">
          <li v-for="item in primaryItems" :key="item.path">
            <router-link :to="item.path" class="nav-destination" :class="{ selected: isMenuActive(item.path, route.path) }" :aria-current="isMenuActive(item.path, route.path) ? 'page' : undefined">
              <component :is="item.icon" class="h-4 w-4 shrink-0" /><span>{{ item.name }}</span>
            </router-link>
          </li>
          <li v-if="moreItems.length">
            <details ref="more" class="topbar-more" @keydown="escapeMore">
              <summary class="nav-destination" :class="{ 'has-current': activeMore }">
                {{ $t('nav.more') }}<span v-if="activeMore" class="topbar-current-page">· {{ activeMore.name }}</span><ChevronDownIcon class="h-4 w-4" />
              </summary>
              <ul class="topbar-more-panel" :aria-label="$t('nav.pages')">
                <li v-for="item in moreItems" :key="item.path">
                  <router-link :to="item.path" class="nav-destination" :class="{ selected: isMenuActive(item.path, route.path) }" :aria-current="isMenuActive(item.path, route.path) ? 'page' : undefined" @click="closeNavigation">
                    <component :is="item.icon" class="h-4 w-4 shrink-0" /><span>{{ item.name }}</span>
                  </router-link>
                </li>
              </ul>
            </details>
          </li>
        </ul>
      </nav>
      <div class="topbar-tools">
        <button class="nav-search-trigger" aria-keyshortcuts="Control+K Meta+K" @click="$emit('search')" :aria-label="$t('nav.search')">
          <MagnifyingGlassIcon class="h-4 w-4 shrink-0" /><span class="topbar-search-label">{{ $t('nav.search') }}</span><kbd>⌘ / Ctrl K</kbd>
        </button>
        <NavUtilities compact hide-version />
      </div>
    </div>
    <dialog id="topbar-drawer" ref="drawer" class="topbar-drawer" :aria-label="$t('nav.navigation')" @close="drawerOpen = false" @click="event => { if (event.target === drawer) closeNavigation() }">
      <div class="topbar-drawer-heading">
        <strong>{{ $t('nav.navigation') }}</strong>
        <button class="nav-icon-button" :aria-label="$t('nav.close')" @click="closeNavigation"><XMarkIcon class="h-5 w-5" /></button>
      </div>
      <nav :aria-label="$t('nav.navigation')">
        <section v-for="group in menuItems" :key="group.id" :aria-labelledby="'drawer-' + group.id">
          <h2 :id="'drawer-' + group.id" class="nav-group-label">{{ group.name }}</h2>
          <ul>
            <li v-for="item in group.items" :key="item.path">
              <router-link :to="item.path" class="nav-destination" :class="{ selected: isMenuActive(item.path, route.path) }" :aria-current="isMenuActive(item.path, route.path) ? 'page' : undefined" @click="closeNavigation">
                <component :is="item.icon" class="h-4 w-4 shrink-0" /><span>{{ item.name }}</span>
              </router-link>
            </li>
          </ul>
        </section>
      </nav>
    </dialog>
  </header>
</template>
