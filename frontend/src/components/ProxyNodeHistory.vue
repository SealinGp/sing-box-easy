<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ClockIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import Popover from '../volt/Popover.vue'
import TooltipTarget from './TooltipTarget.vue'
import ProbeTrendChart from './ProbeTrendChart.vue'
import { displayName } from '../utils/proxyGroups'
import { nodeDelayPoints } from '../utils/nodeDelayHistory'
import { formatLatency } from '../utils/probeChart'
import type { NodeDelaySample } from '../types/runtime'

const props = defineProps<{ name: string; history?: NodeDelaySample[] }>()
const { t, locale } = useI18n()
const panel = ref<InstanceType<typeof Popover>>()
const open = ref(false)
const label = computed(() => t('proxies.history.open', { node: displayName(props.name) }))
const points = computed(() => nodeDelayPoints(props.history))
const recent = computed(() => [...points.value].reverse())
function toggle(event: Event) { panel.value?.toggle(event) }
</script>

<template>
  <span class="inline-flex shrink-0" @click.stop @keydown.enter.stop @keydown.space.stop>
    <!-- The surrounding selector card is already a button. -->
    <TooltipTarget as="span" role="button" tabindex="0"
      class="inline-flex h-8 w-8 cursor-pointer items-center justify-center rounded-control text-text-secondary hover:bg-bg-tertiary hover:text-primary-600 focus-visible:outline-2 focus-visible:outline-primary-500"
      :text="label" :aria-label="label" :aria-expanded="open" aria-haspopup="dialog"
      @click.stop="toggle" @keydown.enter.stop.prevent="toggle" @keydown.space.stop.prevent="toggle">
      <ClockIcon class="h-4 w-4" aria-hidden="true" />
    </TooltipTarget>
    <Popover ref="panel" interaction="menu" :aria-label="label" panel-class="w-[32rem]"
      @show="open = true" @hide="open = false">
      <div class="space-y-3 text-left">
        <div class="flex items-start gap-2">
          <div class="min-w-0 flex-1">
            <h3 class="text-sm font-semibold text-text-primary">{{ t('proxies.history.title') }}</h3>
            <p class="break-words text-xs text-text-secondary">{{ displayName(name) }}</p>
          </div>
          <button type="button" class="icon-action" :aria-label="t('common.close')" @click="panel?.hide()">
            <XMarkIcon class="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <p v-if="!points.length" class="py-4 text-sm text-text-secondary">{{ t('proxies.history.empty') }}</p>
        <template v-else>
          <ProbeTrendChart :points="points" latency-only />
          <p class="text-xs text-text-secondary">{{ t('proxies.history.retained', { n: points.length }) }}</p>
          <p v-if="points.length === 1" class="text-xs text-text-secondary">{{ t('proxies.history.single') }}</p>
          <details>
            <summary class="cursor-pointer text-xs text-primary-600 focus-visible:outline-2 focus-visible:outline-primary-500">{{ t('proxies.history.samples') }}</summary>
            <ul class="mt-2 max-h-40 space-y-1 overflow-y-auto text-xs tabular-nums">
              <li v-for="(point, index) in recent" :key="`${point.at}-${index}`" class="flex justify-between gap-3">
                <time :datetime="point.at">{{ new Date(point.at).toLocaleString(locale) }}</time>
                <span :class="point.reachable ? 'text-text-primary' : 'text-red-600'">{{ point.reachable ? formatLatency(point.avg_ms) : t('proxies.noResult') }}</span>
              </li>
            </ul>
          </details>
        </template>
      </div>
    </Popover>
  </span>
</template>
