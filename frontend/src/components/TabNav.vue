<script setup lang="ts">
import { ref, watch } from 'vue'
import { useNavIndicator } from '../composables/useNavIndicator'
import { useRoute } from 'vue-router'
import PageTransition from './PageTransition.vue'

interface Tab {
  path: string
  label: string
}

const props = defineProps<{
  tabs: Tab[]
}>()

const route = useRoute()
const nav = ref<HTMLElement | null>(null)
const indicator = ref<HTMLElement | null>(null)
const { measure } = useNavIndicator({
  scroller: nav, content: nav, indicator, selector: '[aria-current="page"]',
})
watch([() => route.path, () => props.tabs], measure, { deep: true, flush: 'post' })

// Check if a tab is active based on current route
const isActiveTab = (path: string) => {
  return route.path === path
}
</script>

<template>
  <div class="page-shell space-y-5">
    <!-- Keep the navigation separate from the page's heading and actions. -->
    <div class="border-b border-gray-200 dark:border-gray-700">
      <nav ref="nav" class="tab-navigation -mb-px flex gap-4 overflow-x-auto">
        <span ref="indicator" class="nav-active-indicator tab-active-indicator" aria-hidden="true" />
        <RouterLink
          v-for="tab in tabs"
          :key="tab.path"
          :to="tab.path"
          :aria-current="isActiveTab(tab.path) ? 'page' : undefined"
          :class="[
            'shrink-0 whitespace-nowrap py-1 px-0.5 border-b-2 border-transparent font-medium text-sm transition-colors',
            isActiveTab(tab.path)
              ? 'text-primary-600 dark:text-primary-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
          ]"
        >
          {{ tab.label }}
        </RouterLink>
      </nav>
    </div>

    <!-- Router view for rendering child components -->
    <RouterView v-slot="{ Component, route: childRoute }">
      <PageTransition :view-key="childRoute.path"><component :is="Component" /></PageTransition>
    </RouterView>
  </div>
</template>
