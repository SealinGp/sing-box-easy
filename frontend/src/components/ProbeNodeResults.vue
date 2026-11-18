<script setup lang="ts">
import { computed } from 'vue'
import { CheckCircleIcon, MinusCircleIcon, XCircleIcon } from '@heroicons/vue/24/outline'
import TooltipTarget from './TooltipTarget.vue'
import { displayName } from '../utils/proxyGroups'
import type { ProbeNodeResult } from '../types/subprobe'

const props = defineProps<{ nodes: ProbeNodeResult[] }>()
const sorted = computed(() => [...props.nodes].sort((a, b) => {
  const rank = (node: ProbeNodeResult) => node.skipped ? 1 : node.error ? 0 : 2
  return rank(a) - rank(b) || (b.delay_ms ?? 0) - (a.delay_ms ?? 0)
}))
</script>

<template>
  <ul class="divide-y divide-border" role="list">
    <li v-for="node in sorted" :key="node.tag" class="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-x-2 gap-y-1 px-3 py-2 text-xs">
      <CheckCircleIcon v-if="!node.error && !node.skipped" class="h-4 w-4 text-green-500" aria-hidden="true" />
      <MinusCircleIcon v-else-if="node.skipped" class="h-4 w-4 text-text-secondary" aria-hidden="true" />
      <XCircleIcon v-else class="h-4 w-4 text-red-500" aria-hidden="true" />
      <TooltipTarget as="span" tabindex="0" class="min-w-0 truncate text-text-primary" :text="node.tag">{{ displayName(node.tag) }}</TooltipTarget>
      <span v-if="!node.error && !node.skipped" class="font-mono tabular-nums text-text-secondary">{{ node.delay_ms ?? 0 }} ms</span>
      <span v-else-if="node.skipped" class="text-text-secondary">{{ $t('subProbe.dialog.untestable') }}</span>
      <span v-else class="text-red-600 dark:text-red-400">{{ $t('subProbe.dialog.unreachable') }}</span>
      <p v-if="node.error" class="col-span-2 col-start-2 break-words text-text-secondary">{{ node.error }}</p>
    </li>
  </ul>
</template>
