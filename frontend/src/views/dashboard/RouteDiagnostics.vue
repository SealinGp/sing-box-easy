<script setup lang="ts">
/**
 * Route diagnostics: what a destination would do, before it does it.
 *
 * The counterpart to DNS diagnostics, and deliberately the counterpart to the
 * Clash dashboard too — that shows connections after they exist, which is the
 * wrong half of the loop when the question is whether a config edit was right.
 */
import Card from '../../components/Card.vue'
import RouteProbePanel from '../../components/RouteProbePanel.vue'
</script>

<template>
  <div class="page-shell">
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
        {{ $t('routeProbe.title') }}
      </h3>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">{{ $t('routeProbe.desc') }}</p>
      <RouteProbePanel />
    </Card>
  </div>
</template>
