<script setup lang="ts">
import TabNav from '../../components/TabNav.vue'

// Tabs configuration
const tabs = [
  { path: '/dashboard/dns/servers', label: 'DNS Servers' },
  { path: '/dashboard/dns/rules', label: 'DNS Rules' },
  { path: '/dashboard/dns/settings', label: 'Settings' },
]
</script>

<template>
  <TabNav title="DNS Configuration" :tabs="tabs" />
</template>