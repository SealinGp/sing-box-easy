<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import TabNav from '../../components/TabNav.vue'

const { t } = useI18n()

// Tabs configuration — Outbounds is the parent surface that groups the node
// list, subscriptions, and node rules (mirrors the DNS tab nav).
const tabs = computed(() => [
  { path: '/dashboard/outbounds/list', label: t('outbounds.tabs.list') },
  { path: '/dashboard/outbounds/subscriptions', label: t('outbounds.tabs.subscriptions') },
  { path: '/dashboard/outbounds/node-rules', label: t('outbounds.tabs.nodeRules') },
])
</script>

<template>
  <TabNav :tabs="tabs" />
</template>
