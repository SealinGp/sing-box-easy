<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { ArrowPathIcon } from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import { LATENCY_ROUNDS, latencyTargets, latencyStats, measureImage } from '../utils/browserNetwork'

const samples = ref<(number | null)[][]>(latencyTargets.map(() => []))
const running = ref(false)
let controller: AbortController | undefined
const rows = computed(() => latencyTargets.map((target, index) => {
  const values = samples.value[index] ?? []
  return { ...target, values, stats: latencyStats(values), failed: values.filter(v => v === null).length, scale: Math.max(1, ...values.map(v => v ?? 0)) }
}))
async function refresh() {
  if (running.value) return
  const current = new AbortController()
  controller = current
  running.value = true
  samples.value = latencyTargets.map(() => [])
  try {
    await Promise.all(latencyTargets.map(async (target, index) => {
      for (let round = 0; round < LATENCY_ROUNDS && !current.signal.aborted; round++) {
        const value = await measureImage(target.url, current.signal)
        if (!current.signal.aborted) samples.value[index]!.push(value)
      }
    }))
  } finally { running.value = false }
}
onMounted(refresh)
onBeforeUnmount(() => controller?.abort())
</script>

<template>
  <section data-testid="latency-card" class="overview-card-frame bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface" :aria-busy="running">
    <header class="flex items-center justify-between gap-3 mb-2">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">{{ $t('overview.networkChecks.latency') }}</h3>
      <Button variant="ghost" size="sm" :disabled="running" :aria-label="$t('overview.networkChecks.refreshLatency')" @click="refresh"><ArrowPathIcon class="h-4 w-4" :class="{ 'motion-safe:animate-spin': running }" /></Button>
    </header>
    <p class="text-xs text-gray-500 dark:text-gray-400 mb-4">{{ $t('overview.networkChecks.latencyHelp') }}</p>
    <div class="overview-card-body quiet-scrollbar space-y-4">
      <div v-for="row in rows" :key="row.name">
        <div class="flex justify-between text-sm mb-1"><span>{{ row.name }}</span><span class="text-xs text-gray-500 tabular-nums">{{ row.values.length }}/{{ LATENCY_ROUNDS }}</span></div>
        <div class="flex h-8 items-end gap-1" :aria-label="row.name">
          <span v-for="index in LATENCY_ROUNDS" :key="index" class="flex-1 rounded-sm motion-safe:transition-[height,background-color] motion-safe:duration-200"
            :class="row.values[index - 1] === undefined ? 'bg-gray-100 dark:bg-slate-700' : row.values[index - 1] === null ? 'bg-red-400/60' : row.values[index - 1]! < 400 ? 'bg-teal-500' : row.values[index - 1]! < 800 ? 'bg-amber-500' : 'bg-red-500'"
            :style="{ height: row.values[index - 1] === null ? '100%' : `${Math.max(15, (row.values[index - 1] ?? 0) / row.scale * 100)}%` }"
            :title="row.values[index - 1] === undefined ? $t('overview.networkChecks.pending') : row.values[index - 1] === null ? $t('overview.networkChecks.failed') : `${row.values[index - 1]} ms`" />
        </div>
        <div class="flex flex-wrap gap-x-3 text-xs text-gray-500 dark:text-gray-400 min-h-5 mt-1 tabular-nums">
          <template v-if="row.stats"><span v-for="key in (['min', 'avg', 'max'] as const)" :key="key">{{ $t(`overview.networkChecks.${key}`) }} {{ row.stats[key] }} ms</span></template>
          <span v-else>{{ running ? $t('overview.networkChecks.measuring') : $t('overview.networkChecks.failed') }}</span>
          <span v-if="row.failed" class="text-red-600 dark:text-red-400">{{ $t('overview.networkChecks.failures', { count: row.failed }) }}</span>
        </div>
      </div>
    </div>
  </section>
</template>
