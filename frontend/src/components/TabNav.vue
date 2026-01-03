<script setup lang="ts">
import { useRoute } from 'vue-router'

interface Tab {
  path: string
  label: string
}

defineProps<{
  title: string
  tabs: Tab[]
}>()

const route = useRoute()

// Check if a tab is active based on current route
const isActiveTab = (path: string) => {
  return route.path === path
}
</script>

<template>
  <div class="p-8">
    <h2 class="text-3xl font-bold text-gray-900 dark:text-gray-100 mb-6">{{ title }}</h2>

    <!-- Tabs -->
    <div class="mb-6 border-b border-gray-200 dark:border-gray-700">
      <nav class="-mb-px flex space-x-8">
        <RouterLink
          v-for="tab in tabs"
          :key="tab.path"
          :to="tab.path"
          :class="[
            'py-2 px-1 border-b-2 font-medium text-sm transition-colors',
            isActiveTab(tab.path)
              ? 'border-violet-500 text-violet-600 dark:text-violet-400'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
          ]"
        >
          {{ tab.label }}
        </RouterLink>
      </nav>
    </div>

    <!-- Router view for rendering child components -->
    <RouterView />
  </div>
</template>