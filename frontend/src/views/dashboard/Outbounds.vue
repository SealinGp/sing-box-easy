<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
const route = useRoute()
const { t } = useI18n()
const title = computed(() => t(route.path.endsWith('/subscriptions') ? 'nav.subscriptions'
  : route.path.endsWith('/node-rules') ? 'nav.nodeRules' : 'nav.outbounds'))
</script>

<template>
  <!-- These are direct menu destinations now; preserve their nested URLs. -->
  <div class="page-shell">
    <h1 class="text-lg font-semibold text-gray-900 dark:text-white mb-2">{{ title }}</h1>
    <RouterView />
  </div>
</template>
