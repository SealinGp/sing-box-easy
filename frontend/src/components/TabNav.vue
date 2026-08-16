<script setup lang="ts">
import { useRoute } from 'vue-router'

interface Tab {
  path: string
  label: string
}

defineProps<{
  tabs: Tab[]
}>()

const route = useRoute()

// Check if a tab is active based on current route
const isActiveTab = (path: string) => {
  return route.path === path
}
</script>

<template>
  <!--
    `page-shell` (style/density.css) is the gutter for every tabbed section —
    Outbounds, DNS, Route and Experimental all render through here, so this one
    element sets the page padding for 12 views. It was `p-8`.

    `pt-3` trims the top gutter to 12px for tabbed pages only, which lines the
    tab strip up with the top edge of the sidebar / top bar card (both `m-3`).
    The utility wins over `.page-shell`'s own padding because density.css sits
    in `@layer components` — see DESIGN.md §10.2.
  -->
  <div class="page-shell pt-3">
    <!--
      The strip is chrome, not content: it gets the minimum that still reads as
      a divider. Was `mb-4` + `py-1.5` links, which cost 64px before a page's
      first row; now 40px (12px gutter + a 28px strip). Link boxes stay at
      ~28px, above the 24px WCAG 2.2 minimum target size.

      No bottom margin: every panel below already leads with its own toolbar
      row carrying `mb-2`, so a margin here stacked with it. The panel owns the
      gap.
    -->
    <div class="border-b border-gray-200 dark:border-gray-700">
      <nav class="-mb-px flex space-x-4">
        <RouterLink
          v-for="tab in tabs"
          :key="tab.path"
          :to="tab.path"
          :class="[
            'py-1 px-0.5 border-b-2 font-medium text-sm transition-colors',
            isActiveTab(tab.path)
              ? 'border-primary-500 text-primary-600 dark:text-primary-400'
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