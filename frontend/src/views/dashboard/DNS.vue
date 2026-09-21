<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import TabNav from '../../components/TabNav.vue'

const { t } = useI18n()

// Diagnostics is deliberately NOT a tab here any more: it lives in its own
// nav section, and a tab that navigates OUT of the section it sits in — into a
// different tab bar — reads as a bug. The old path still redirects, so
// bookmarks keep working.
const tabs = computed(() => [
  { path: '/dashboard/dns/servers', label: t('dns.tabs.servers') },
  { path: '/dashboard/dns/rules', label: t('dns.tabs.rules') },
  { path: '/dashboard/dns/settings', label: t('dns.tabs.settings') },
])
</script>

<template>
  <TabNav :tabs="tabs" />
</template>
