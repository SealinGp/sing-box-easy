<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ClockIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import Popover from '../volt/Popover.vue'
import TooltipTarget from './TooltipTarget.vue'
import ProbeTrendChart from './ProbeTrendChart.vue'
import { displayName } from '../utils/proxyGroups'
import { nodeDelayPoints } from '../utils/nodeDelayHistory'
import type { NodeDelaySample } from '../types/runtime'
import { runtimeService } from '../services'
import { apiErrorMessage } from '../utils/apiErrorMessage'

const props = defineProps<{ name: string; history?: NodeDelaySample[] }>()
const { t, locale } = useI18n()
const panel = ref<InstanceType<typeof Popover>>()
const open = ref(false)
const label = computed(() => t('proxies.history.open', { node: displayName(props.name) }))
const stored = ref<NodeDelaySample[] | null>(null)
const loading = ref(false)
const error = ref('')
let generation = 0
const points = computed(() => nodeDelayPoints(stored.value ?? props.history))
const recent = computed(() => [...(stored.value ?? props.history ?? [])]
  .filter(sample => Number.isFinite(Date.parse(sample.time)))
  .sort((a, b) => Date.parse(b.time) - Date.parse(a.time)))
function toggle(event: Event) { panel.value?.toggle(event) }
async function load() {
  const current = ++generation
  loading.value = true
  error.value = ''
  try {
    const { data } = await runtimeService.getNodeHistory(props.name)
    if (current === generation) stored.value = data.points
  } catch (err) {
    if (current === generation) error.value = apiErrorMessage(err, t('proxies.loadFailed'))
  } finally { if (current === generation) loading.value = false }
}
watch(() => props.name, () => { generation++; stored.value = null; if (open.value) void load() })
watch(() => props.history, () => { if (open.value) void load() })
function close() { open.value = false; generation++; loading.value = false }
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
      @show="open = true; load()" @hide="close">
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
        <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }} <button type="button" class="underline" @click="load">{{ t('proxies.retry') }}</button></p>
        <p v-if="!loading && !points.length && !error" class="py-4 text-sm text-text-secondary">{{ t('proxies.history.empty') }}</p>
        <template v-if="loading || points.length">
          <ProbeTrendChart :points="points" :loading="loading && !points.length" latency-only />
          <p class="text-xs text-text-secondary">{{ t('proxies.history.retained', { n: points.length }) }}</p>
          <p v-if="points.length === 1" class="text-xs text-text-secondary">{{ t('proxies.history.single') }}</p>
          <details>
            <summary class="cursor-pointer text-xs text-primary-600 focus-visible:outline-2 focus-visible:outline-primary-500">{{ t('proxies.history.samples') }}</summary>
            <ul class="mt-2 max-h-40 space-y-1 overflow-y-auto text-xs tabular-nums">
              <li v-for="(point, index) in recent" :key="`${point.time}-${index}`" class="flex flex-wrap justify-between gap-x-3 gap-y-1">
                <time :datetime="point.time">{{ new Date(point.time).toLocaleString(locale) }}</time>
                <TooltipTarget v-if="point.source" as="span" tabindex="0" :text="point.test_url" class="text-text-secondary">{{ t(`proxies.history.sources.${point.source}`) }}</TooltipTarget>
                <span :class="(point.success ?? point.delay > 0) ? 'text-text-primary' : 'text-red-600'">{{ (point.success ?? point.delay > 0) ? `${point.delay} ms` : t('proxies.noResult') }}</span>
              </li>
            </ul>
          </details>
        </template>
      </div>
    </Popover>
  </span>
</template>
