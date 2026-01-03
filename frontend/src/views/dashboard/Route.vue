<script setup lang="ts">
import TabNav from '../../components/TabNav.vue'

// Tabs configuration
const tabs = [
  { path: '/dashboard/route/rules', label: 'Routing Rules' },
  { path: '/dashboard/route/rule-sets', label: 'Rule Sets' },
  { path: '/dashboard/route/final-policy', label: 'Final Policy' },
]
</script>

<template>
  <TabNav title="Route Configuration" :tabs="tabs" />
</template>