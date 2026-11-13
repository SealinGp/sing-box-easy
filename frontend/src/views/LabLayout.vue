<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

import { provideInterfaceMotion } from '../composables/useInterfaceMotion'
provideInterfaceMotion()

</script>
<template>
  <div class="lab-preview min-w-0 space-y-6 p-4 sm:p-6">
    <nav :aria-label="t('lab.labPages')" class="flex gap-2 border-b border-border pb-3">
      <RouterLink v-for="page in [{path:'functionality',label:t('lab.functionality')},{path:t('lab.theme'),label:t('lab.themeStyle')}]" :key="page.path" :to="`/dashboard/lab/${page.path}`" class="rounded-control px-3 py-2 text-sm font-medium focus-visible:outline-2 focus-visible:outline-primary" :class="$route.path.endsWith('/' + page.path) ? 'bg-bg-tertiary text-text-primary' : 'text-text-secondary hover:bg-bg-tertiary'" :aria-current="$route.path.endsWith('/' + page.path) ? 'page' : undefined">{{ page.label }}</RouterLink>
    </nav>
    <RouterView v-slot="{ Component }"><KeepAlive><component :is="Component" /></KeepAlive></RouterView>
  </div>
</template>
