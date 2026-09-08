<script setup lang="ts">
import { ref, watch } from 'vue'
import { useNavIndicator } from '../composables/useNavIndicator'
import { useRoute } from 'vue-router'
import { MagnifyingGlassIcon } from '@heroicons/vue/24/outline'
import type { MenuGroup } from '../navigation/menu'
import { isMenuActive } from '../navigation/menu'
import NavUtilities from './NavUtilities.vue'
const props = defineProps<{ menuItems: MenuGroup[] }>()
defineEmits<{ search: [] }>()
const route = useRoute()
const groups = ref<HTMLElement | null>(null)
const indicator = ref<HTMLElement | null>(null)
const { measure } = useNavIndicator({
  scroller: groups, content: groups, indicator, selector: '[aria-current="page"]',
})
watch([() => route.path, () => props.menuItems], measure, { deep: true, flush: 'post' })
</script>

<template>
  <aside class="navigation-sidebar">
    <router-link to="/dashboard/overview" class="nav-brand">
      <img src="/logo.jpg" alt="" width="30" height="30" />
      <span>Sing Box Easy<span class="nav-brand-caption">{{ $t('nav.workspace') }}</span></span>
    </router-link>
    <button class="nav-search-trigger" aria-keyshortcuts="Control+K Meta+K" @click="$emit('search')">
      <MagnifyingGlassIcon class="h-4 w-4 shrink-0" />
      <span>{{ $t('nav.search') }}</span><kbd>⌘ / Ctrl K</kbd>
    </button>
    <nav ref="groups" class="sidebar-groups" :aria-label="$t('nav.navigation')">
      <span ref="indicator" class="nav-active-indicator sidebar-active-indicator" aria-hidden="true" />
      <section v-for="group in menuItems" :key="group.id" :aria-labelledby="'sidebar-' + group.id">
        <h2 :id="'sidebar-' + group.id" class="nav-group-label">{{ group.name }}</h2>
        <ul>
          <li v-for="item in group.items" :key="item.path">
            <router-link :to="item.path" class="nav-destination" :class="{ selected: isMenuActive(item.path, route.path) }" :aria-current="isMenuActive(item.path, route.path) ? 'page' : undefined">
              <component :is="item.icon" class="h-4 w-4 shrink-0" /><span>{{ item.name }}</span>
            </router-link>
          </li>
        </ul>
      </section>
    </nav>
    <NavUtilities />
  </aside>
</template>
