<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ArrowPathIcon, EyeIcon, EyeSlashIcon } from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import { ipProviders, lookupNetwork, type NetworkInfo } from '../utils/browserNetwork'

const rows = ref(ipProviders.map(provider => ({ provider, info: null as NetworkInfo | null, loading: true, failed: false })))
const hidden = ref(true)
const running = ref(false)
const controllers = new Set<AbortController>()
let disposed = false
async function refresh() {
  if (running.value) return
  running.value = true
  await Promise.all(rows.value.map(async row => {
    row.loading = true
    row.failed = false
    const controller = new AbortController()
    controllers.add(controller)
    const timer = setTimeout(() => controller.abort(), 8000)
    try {
      const info = await lookupNetwork(row.provider, controller.signal)
      if (!disposed) row.info = info
    } catch {
      if (!disposed) { row.failed = true; row.info = null }
    } finally {
      clearTimeout(timer)
      controllers.delete(controller)
      if (!disposed) row.loading = false
    }
  }))
  if (!disposed) running.value = false
}
onMounted(refresh)
onBeforeUnmount(() => { disposed = true; controllers.forEach(controller => controller.abort()) })
</script>

<template>
  <section data-testid="network-info-card" class="overview-card-frame bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface" :aria-busy="running">
    <header class="flex items-center justify-between gap-3 mb-2">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">{{ $t('overview.networkChecks.networkInfo') }}</h3>
      <div class="flex gap-1">
        <Button variant="ghost" size="sm" :aria-label="$t(hidden ? 'overview.networkChecks.showIP' : 'overview.networkChecks.hideIP')" :aria-pressed="!hidden" @click="hidden = !hidden"><component :is="hidden ? EyeIcon : EyeSlashIcon" class="h-4 w-4" /></Button>
        <Button variant="ghost" size="sm" :disabled="running" :aria-label="$t('overview.networkChecks.refreshNetwork')" @click="refresh"><ArrowPathIcon class="h-4 w-4" :class="{ 'motion-safe:animate-spin': running }" /></Button>
      </div>
    </header>
    <p class="text-xs text-gray-500 dark:text-gray-400 mb-4">{{ $t('overview.networkChecks.networkHelp') }}</p>
    <div class="overview-card-body quiet-scrollbar space-y-3">
      <div v-for="row in rows" :key="row.provider.id" class="min-h-28 rounded-lg bg-gray-50 dark:bg-slate-900/40 p-3">
        <div class="text-xs font-medium text-gray-500 dark:text-gray-400 mb-2">{{ row.provider.name }}</div>
        <div v-if="row.loading" class="space-y-2" role="status" :aria-label="$t('overview.networkChecks.measuring')"><div class="h-5 w-3/4 rounded bg-gray-200 dark:bg-slate-700 motion-safe:animate-pulse" /><div class="h-4 w-1/2 rounded bg-gray-200 dark:bg-slate-700 motion-safe:animate-pulse" /></div>
        <p v-else-if="row.failed" class="text-xs text-red-600 dark:text-red-400">{{ $t('overview.networkChecks.lookupFailed') }}</p>
        <template v-else-if="row.info">
          <p class="font-mono text-sm break-all" data-testid="network-ip">{{ hidden ? '••••••••' : row.info.ip }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1">{{ row.info.location || '—' }}</p>
          <p class="text-xs text-gray-500 dark:text-gray-400 mt-1 break-words">{{ row.info.organization || '—' }}</p>
        </template>
      </div>
    </div>
  </section>
</template>
