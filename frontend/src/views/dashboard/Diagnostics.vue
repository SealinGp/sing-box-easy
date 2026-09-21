<script setup lang="ts">
/**
 * Diagnostics: the three tools that answer "is my config doing what I meant?"
 *
 * They were previously tabs under DNS and Route, which put them beside the
 * editors for the thing they inspect. That reads well until you are actually
 * debugging, when the question spans both — the dual-stack test resolves a
 * name AND routes its addresses AND sends traffic — and there is no correct
 * parent to file it under. So the three live together, and the old paths
 * redirect here.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import TabNav from '../../components/TabNav.vue'

const { t } = useI18n()

const tabs = computed(() => [
  { path: '/dashboard/diagnostics/dual-stack', label: t('diagnostics.tabs.dualStack') },
  { path: '/dashboard/diagnostics/dns', label: t('diagnostics.tabs.dns') },
  { path: '/dashboard/diagnostics/route', label: t('diagnostics.tabs.route') },
])
</script>

<template>
  <TabNav :tabs="tabs" />
</template>
