<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import TabNav from '../../components/TabNav.vue'

const { t } = useI18n()

// Tabs configuration
const tabs = computed(() => [
  { path: '/dashboard/experimental/cache-file', label: t('experimental.tabs.cacheFile') },
  { path: '/dashboard/experimental/clash-api', label: t('experimental.tabs.clashApi') },
  { path: '/dashboard/experimental/v2ray-api', label: t('experimental.tabs.v2rayApi') },
])
</script>

<template>
  <TabNav :title="$t('experimental.tabs.title')" :tabs="tabs" />
</template>