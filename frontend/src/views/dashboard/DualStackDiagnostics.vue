<script setup lang="ts">
/**
 * Dual-stack domain test: does this name work over IPv4 and over IPv6, and
 * does each family go where the config says it should?
 *
 * The server-side form of scripts/validate-ipv6-split.sh, which cannot run
 * where the answer is — its network checks SKIP on a client without global
 * IPv6, and the one machine that can test the router's IPv6 egress is the
 * router.
 */
import Card from '../../components/Card.vue'
import DualStackPanel from '../../components/DualStackPanel.vue'
</script>

<template>
  <div class="space-y-4 pt-2">
    <Card>
      <h3 class="text-lg font-semibold text-gray-900 dark:text-gray-100">
        {{ $t('dualStack.title') }}
      </h3>
      <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">{{ $t('dualStack.desc') }}</p>
      <DualStackPanel />
    </Card>
  </div>
</template>
