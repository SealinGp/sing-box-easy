<script setup lang="ts">
/**
 * Compact route simulator for the Overview page.
 *
 * The counterpart to DnsProbeCard, and the other half of the same question:
 * that one answers "what address does this name resolve to?", this one answers
 * "and which outbound would the connection then leave through?" — before any
 * connection is made, which is what separates it from the Clash dashboard.
 *
 * The full rule ladder lives on Route → Simulator, which this links to.
 */
import { ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/outline'
import { RouterLink } from 'vue-router'
import RouteProbePanel from './RouteProbePanel.vue'

const SIMULATOR_ROUTE = '/dashboard/route/diagnostics'
</script>

<template>
  <div class="bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <div class="flex items-start justify-between gap-3 mb-4">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">
        {{ $t('routeProbe.title') }}
      </h3>
      <RouterLink
        :to="SIMULATOR_ROUTE"
        class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors flex-shrink-0"
      >
        {{ $t('routeProbe.openSimulator') }}
        <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
      </RouterLink>
    </div>

    <RouteProbePanel compact />
  </div>
</template>
