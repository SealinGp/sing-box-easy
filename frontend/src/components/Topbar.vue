<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { Bars3Icon, ChevronDownIcon, MagnifyingGlassIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import type { MenuGroup } from '../navigation/menu'
import { isMenuActive } from '../navigation/menu'
import NavUtilities from './NavUtilities.vue'
import Popover from '../volt/Popover.vue'

const props = defineProps<{ menuItems: MenuGroup[] }>()
defineEmits<{ search: [] }>()
const route = useRoute()
const primaryPaths = ['overview', 'outbounds/subscriptions', 'outbounds/list', 'inbounds', 'route', 'dns', 'logs'].map(path => '/dashboard/' + path)
const allItems = computed(() => props.menuItems.flatMap(group => group.items))
const primaryItems = computed(() => primaryPaths.flatMap(path => allItems.value.filter(item => item.path === path)))
const moreItems = computed(() => allItems.value.filter(item => !primaryPaths.includes(item.path)))
const activeMore = computed(() => moreItems.value.find(item => isMenuActive(item.path, route.path)))
const more = ref<InstanceType<typeof Popover> | null>(null)
const drawer = ref<InstanceType<typeof Popover> | null>(null)
const moreOpen = ref(false)
const drawerOpen = ref(false)
function closeNavigation() {
  more.value?.hide()
  drawer.value?.hide()
  moreOpen.value = false
  drawerOpen.value = false
}
watch(() => route.path, closeNavigation)
</script>

<template>
  <header class="navigation-topbar">
    <div class="topbar-brand-row">
      <div class="topbar-brand">
        <button class="nav-icon-button topbar-mobile-toggle" :aria-label="$t('nav.navigation')" :aria-expanded="drawerOpen" aria-controls="topbar-drawer" @click="drawer?.toggle($event)">
          <Bars3Icon class="h-5 w-5" />
        </button>
        <router-link to="/dashboard/overview" class="nav-brand" :aria-label="'Sing Box Easy · ' + $t('nav.overview')" :title="'Sing Box Easy · ' + $t('nav.overview')">
          <img src="/logo.svg" alt="" width="28" height="28" />
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
            <div class="topbar-more">
              <button type="button" class="nav-destination" :class="{ 'has-current': activeMore }" :aria-expanded="moreOpen" aria-controls="topbar-more-panel" @click="more?.toggle($event)">
                {{ $t('nav.more') }}<span v-if="activeMore" class="topbar-current-page">· {{ activeMore.name }}</span><ChevronDownIcon class="h-4 w-4" />
              </button>
              <Popover id="topbar-more-panel" ref="more" :aria-label="$t('nav.pages')" @show="moreOpen = true" @hide="moreOpen = false">
              <ul class="space-y-1" :aria-label="$t('nav.pages')">
                <li v-for="item in moreItems" :key="item.path">
                  <router-link :to="item.path" class="nav-destination" :class="{ selected: isMenuActive(item.path, route.path) }" :aria-current="isMenuActive(item.path, route.path) ? 'page' : undefined" @click="closeNavigation">
                    <component :is="item.icon" class="h-4 w-4 shrink-0" /><span>{{ item.name }}</span>
                  </router-link>
                </li>
              </ul>
              </Popover>
            </div>
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
    <Popover id="topbar-drawer" ref="drawer" class="topbar-navigation-popover" :aria-label="$t('nav.navigation')" @show="drawerOpen = true" @hide="drawerOpen = false">
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
    </Popover>
  </header>
</template>
