<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { MagnifyingGlassIcon } from '@heroicons/vue/24/outline'
import type { MenuGroup } from '../navigation/menu'
import { isMenuActive } from '../navigation/menu'
import NavUtilities from './NavUtilities.vue'
const props = defineProps<{ menuItems: MenuGroup[] }>()
defineEmits<{ search: [] }>()
const route = useRoute()
const selected = ref(props.menuItems[0]?.id ?? '')
watch(() => route.path, () => {
  const group = props.menuItems.find(group => group.items.some(item => isMenuActive(item.path, route.path)))
  selected.value = group?.id ?? 'system'
}, { immediate: true })
watch([() => route.path, selected], async () => {
  await nextTick()
  destinations.value?.querySelector('[aria-current="page"]')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
}, { flush: 'post' })
const destinations = ref<HTMLElement>()
const currentGroup = computed(() => props.menuItems.find(group => group.id === selected.value))
</script>

<template>
  <header class="navigation-topbar">
    <div class="topbar-brand-row">
      <router-link to="/dashboard/overview" class="nav-brand">
        <img src="/logo.jpg" alt="" width="28" height="28" /><span>Sing Box Easy</span>
      </router-link>
      <div class="topbar-tools">
        <button class="nav-search-trigger" aria-keyshortcuts="Control+K Meta+K" @click="$emit('search')" :aria-label="$t('nav.search')">
          <MagnifyingGlassIcon class="h-4 w-4 shrink-0" /><span class="topbar-search-label">{{ $t('nav.search') }}</span><kbd>⌘ / Ctrl K</kbd>
        </button>
        <NavUtilities compact />
      </div>
    </div>
    <nav :aria-label="$t('nav.navigation')">
      <div class="topbar-groups">
        <button v-for="group in menuItems" :key="group.id" @click="selected = group.id" :aria-pressed="selected === group.id" aria-controls="topbar-destinations" :class="{ selected: selected === group.id }">
          {{ group.name }}<span v-if="group.items.some(item => isMenuActive(item.path, route.path))" class="group-current-dot" aria-hidden="true" />
        </button>
      </div>
      <ul id="topbar-destinations" ref="destinations" class="topbar-destinations" :aria-label="currentGroup?.name">
        <li v-for="item in currentGroup?.items" :key="item.path">
          <router-link :to="item.path" class="nav-destination" :class="{ selected: isMenuActive(item.path, route.path) }" :aria-current="isMenuActive(item.path, route.path) ? 'page' : undefined">
            <component :is="item.icon" class="h-4 w-4 shrink-0" /><span>{{ item.name }}</span>
          </router-link>
        </li>
      </ul>
    </nav>
  </header>
</template>
