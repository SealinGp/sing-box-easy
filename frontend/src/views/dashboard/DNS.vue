<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import TabNav from '../../components/TabNav.vue'

const { t } = useI18n()

// Tabs configuration
const tabs = computed(() => [
  { path: '/dashboard/dns/servers', label: t('dns.tabs.servers') },
  { path: '/dashboard/dns/rules', label: t('dns.tabs.rules') },
  { path: '/dashboard/dns/settings', label: t('dns.tabs.settings') },
])
</script>

<template>
  <TabNav :title="$t('dns.title')" :tabs="tabs" />
</template>
