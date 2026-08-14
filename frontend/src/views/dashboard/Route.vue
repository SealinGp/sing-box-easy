<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import TabNav from '../../components/TabNav.vue'

const { t } = useI18n()

// Tabs configuration
const tabs = computed(() => [
  { path: '/dashboard/route/rules', label: t('route.tabs.rules') },
  { path: '/dashboard/route/rule-sets', label: t('route.tabs.ruleSets') },
  { path: '/dashboard/route/final-policy', label: t('route.tabs.finalPolicy') },
])
</script>

<template>
  <TabNav :tabs="tabs" />
</template>