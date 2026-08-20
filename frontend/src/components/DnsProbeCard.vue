<script setup lang="ts">
/**
 * Compact DNS probe for the Overview page.
 *
 * Answers the common question — "where does this domain actually go?" —
 * without leaving the dashboard. The rule ladder, upstream comparison and
 * per-rule evaluation live on the DNS → Diagnostics tab, which this links to.
 */
import { ArrowTopRightOnSquareIcon } from '@heroicons/vue/24/outline'
import { RouterLink } from 'vue-router'
import DnsProbePanel from './DnsProbePanel.vue'

const DIAGNOSTICS_ROUTE = '/dashboard/dns/diagnostics'
</script>

<template>
  <div class="bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <div class="flex items-start justify-between gap-3 mb-4">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">
        {{ $t('dnsProbe.title') }}
      </h3>
      <RouterLink
        :to="DIAGNOSTICS_ROUTE"
        class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors flex-shrink-0"
      >
        {{ $t('dnsProbe.openDiagnostics') }}
        <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
      </RouterLink>
    </div>

    <DnsProbePanel compact />
  </div>
</template>
