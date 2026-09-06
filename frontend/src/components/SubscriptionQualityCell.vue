<script setup lang="ts">
/**
 * The Subscriptions table's quality cell: latest availability + latency, and
 * the way into the history.
 *
 * Its own component rather than markup inside the table, because the row's
 * three states (measured / never probed / probing off) each need their own
 * treatment, and a single figure is not enough to judge a provider by — so the
 * cell is a button into the trend rather than a read-only readout.
 */
import { computed } from 'vue'
import { ChartBarIcon } from '@heroicons/vue/24/outline'
import type { ProbePoint } from '../types/subprobe'
import { availabilityRatio, formatAvailability, formatLatency } from '../utils/probeChart'
import { qualityStepColors } from '../utils/qualitySteps'
import SegmentedProgress from './SegmentedProgress.vue'

const props = defineProps<{
  point?: ProbePoint
  /** False when the operator turned probing off for this subscription. */
  enabled: boolean
  name: string
}>()
defineEmits<{ open: [] }>()

/** Availability colour: this is a health figure, so a bad one is red. */
const toneClass = computed(() => {
  if (!props.point) return 'text-gray-400 dark:text-gray-500'
  const ratio = availabilityRatio(props.point)
  if (ratio >= 0.9) return 'text-green-600 dark:text-green-400'
  if (ratio >= 0.5) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
})

const latency = computed(() =>
  props.point && props.point.reachable > 0 ? formatLatency(props.point.avg_ms) : '—',
)

/**
 * Blocks coloured by how the nodes split, not by a percentage.
 *
 * percent=100 so every block renders; the colours carry the breakdown. See
 * qualityStepColors for why the green count floors rather than rounds.
 */
const stepColors = computed(() =>
  props.point ? qualityStepColors(props.point.reachable, props.point.total) : [],
)
</script>

<template>
  <button
    v-if="point"
    type="button"
    class="group/q flex cursor-pointer flex-col items-start gap-0.5 rounded p-1 -m-1 text-left transition-colors hover:bg-gray-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:hover:bg-gray-700/40"
    :title="$t('subProbe.openDetail', { name })"
    @click="$emit('open')"
  >
    <span class="flex items-center gap-1.5">
      <span class="text-sm font-semibold" :class="toneClass">
        {{ formatAvailability(point) }}
      </span>
      <ChartBarIcon
        class="h-3.5 w-3.5 text-gray-300 transition-colors group-hover/q:text-primary-500 dark:text-gray-600"
      />
    </span>
    <SegmentedProgress
      :percent="100"
      :steps="5"
      :stroke-color="stepColors"
      size="sm"
      :aria-label="$t('subProbe.nodesTested', { reachable: point.reachable, total: point.total })"
    />
    <span class="text-xs text-gray-500 dark:text-gray-400">{{ latency }}</span>
    <span class="text-xs text-gray-400 dark:text-gray-500">
      {{ $t('subProbe.nodesTested', { reachable: point.reachable, total: point.total }) }}
    </span>
  </button>

  <!--
    Never probed and probing-off are different states. Collapsing them would
    hide the fact that the operator turned it off, leaving a column that looks
    permanently broken.
  -->
  <button
    v-else
    type="button"
    class="cursor-pointer rounded p-1 -m-1 text-xs text-gray-300 transition-colors hover:text-primary-600 dark:text-gray-600 dark:hover:text-primary-400"
    :title="$t('subProbe.openDetail', { name })"
    @click="$emit('open')"
  >
    {{ enabled ? $t('subProbe.never') : $t('subProbe.disabled') }}
  </button>
</template>
