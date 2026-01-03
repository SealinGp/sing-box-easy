<script setup lang="ts">
import TabNav from '../../components/TabNav.vue'

// Tabs configuration
const tabs = [
  { path: '/dashboard/experimental/cache-file', label: 'Cache File' },
  { path: '/dashboard/experimental/clash-api', label: 'Clash API' },
  { path: '/dashboard/experimental/v2ray-api', label: 'V2Ray API' },
]
</script>

<template>
  <TabNav title="Experimental Features" :tabs="tabs" />
</template>