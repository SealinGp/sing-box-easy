<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import { RouterLink } from 'vue-router'
import NodeQualityBadge from './NodeQualityBadge.vue'
import SubscriptionQualityCell from './SubscriptionQualityCell.vue'
import { useNodeQualityStore } from '../stores/nodeQuality'
import Popover from '../volt/Popover.vue'
import ProbeTrendChart from './ProbeTrendChart.vue'
import { displayName } from '../utils/proxyGroups'
import { nodeDelayPoints, nodeProbeSource } from '../utils/nodeDelayHistory'
import type { NodeDelaySample } from '../types/runtime'
import { runtimeService, subscriptionService, subProbeService } from '../services'
import type { Subscription } from '../types/api'
import type { ProbePoint } from '../types/subprobe'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { nodeSubscriptionId } from '../utils/nodeSubscription'
import { FOCUS_PAGES, focusLink } from '../utils/focusTarget'
import { useNodeQuality } from '../composables/useNodeQuality'

const props = defineProps<{ name: string; history?: NodeDelaySample[] }>()
const { t, locale } = useI18n()

// Popover state and async request identity.
const panel = ref<InstanceType<typeof Popover>>()
const open = ref(false)
const label = computed(() => t('proxies.quality.open', { node: displayName(props.name) }))
let generation = 0

// Node quality and historical observations.
const stored = ref<NodeDelaySample[] | null>(null)
const qualityHistory = computed(() => stored.value ?? props.history)
const { rating, summary: qualitySummary } = useNodeQuality(() => qualityHistory.value)
const qualityStore = useNodeQualityStore()
const loading = ref(false)
const error = ref('')
const points = computed(() => nodeDelayPoints(qualityHistory.value))
const recent = computed(() => [...(qualityHistory.value ?? [])]
  .filter(sample => Number.isFinite(Date.parse(sample.time)))
  .sort((a, b) => Date.parse(b.time) - Date.parse(a.time)))

// The subscription owning this node, and its aggregate quality.
const subscriptionId = computed(() => nodeSubscriptionId(props.name))
const subscription = ref<Subscription | null>(null)
const subscriptionName = computed(() => subscription.value?.name ?? '')
const subscriptionPoint = ref<ProbePoint>()
const subscriptionQualityOpen = ref(false)
const subscriptionLoading = ref(false)
const subscriptionError = ref(false)

async function loadSubscription(current = generation) {
  const id = subscriptionId.value
  if (!id || subscriptionName.value) return

  subscriptionLoading.value = true
  subscriptionError.value = false
  try {
    const { data } = await subscriptionService.getSubscriptionByID(id)
    if (current === generation) subscription.value = data
  } catch {
    if (current === generation) subscriptionError.value = true
  } finally {
    if (current === generation) subscriptionLoading.value = false
  }
}

async function loadSubscriptionQuality(current = generation) {
  const id = subscriptionId.value
  if (!id) return

  try {
    const { data } = await subProbeService.getStatus()
    if (current === generation) subscriptionPoint.value = data.latest?.[id]
  } catch {
    // Missing quality data stays unknown; the shared cell can still open history.
    if (current === generation) subscriptionPoint.value = undefined
  }
}

// Opening and refreshing the node panel also refreshes its subscription summary.
async function load() {
  const current = ++generation
  void loadSubscription(current)
  void loadSubscriptionQuality(current)
  loading.value = true
  error.value = ''

  try {
    const { data } = await runtimeService.getNodeHistory(props.name)
    if (current === generation) stored.value = data.points
  } catch (err) {
    if (current === generation) error.value = apiErrorMessage(err, t('proxies.loadFailed'))
  } finally {
    if (current === generation) loading.value = false
  }
}

function toggle(event: Event) {
  panel.value?.toggle(event)
}

function close() {
  open.value = false
  generation++
  loading.value = false
  subscriptionLoading.value = false
}

// Discard stale data and responses when the represented node changes.
watch(() => props.name, () => {
  generation++
  stored.value = null
  subscription.value = null
  subscriptionPoint.value = undefined
  subscriptionQualityOpen.value = false
  subscriptionError.value = false
  subscriptionLoading.value = false
  if (open.value) void load()
})

watch(() => props.history, () => {
  if (open.value) void load()
})
</script>

<template>
  <span class="inline-flex shrink-0" @click.stop @keydown.enter.stop @keydown.space.stop>
    <!-- The surrounding selector card is already a button. -->
    <NodeQualityBadge
      :history="qualityHistory"
      :action-label="label"
      :expanded="open"
      @click.stop="toggle"
      @keydown.enter.stop.prevent="toggle"
      @keydown.space.stop.prevent="toggle"
    />

    <Popover
      ref="panel"
      interaction="menu"
      :dismissable="!subscriptionQualityOpen"
      :aria-label="label"
      panel-class="w-[32rem]"
      @show="open = true; load()"
      @hide="close"
    >
      <div class="space-y-3 text-left">
        <div class="flex items-start gap-2">
          <div class="min-w-0 flex-1">
            <h3 class="text-sm font-semibold text-text-primary">{{ t('proxies.quality.title') }}</h3>
            <p class="mt-1 select-text break-all text-xs leading-relaxed text-text-secondary">{{ name }}</p>
            <div
              v-if="subscriptionId"
              class="mt-1.5 flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-1 text-xs"
              :aria-busy="subscriptionLoading"
            >
              <span class="text-text-secondary">{{ t('proxies.quality.subscription') }}</span>
              <RouterLink
                :to="focusLink(FOCUS_PAGES.subscriptions, subscriptionId)"
                class="break-all rounded-control text-primary-600 underline underline-offset-2 focus-visible:outline-2 focus-visible:outline-primary-500 dark:text-primary-400"
                @click="panel?.hide()"
              >
                {{ subscriptionName || subscriptionId }}
              </RouterLink>
              <SubscriptionQualityCell
                v-if="subscription"
                :subscription="subscription"
                :point="subscriptionPoint"
                size="small"
                @update:detail-visible="subscriptionQualityOpen = $event"
                @refresh="loadSubscriptionQuality()"
              />
              <span v-if="subscriptionError" class="text-text-secondary">{{ t('proxies.quality.subscriptionLoadFailed') }}
                <button
                  type="button"
                  class="rounded-control underline focus-visible:outline-2 focus-visible:outline-primary-500"
                  @click="loadSubscription()"
                >{{ t('proxies.retry') }}</button>
              </span>
            </div>
          </div>
          <button type="button" class="icon-action" :aria-label="t('common.close')" @click="panel?.hide()">
            <XMarkIcon class="h-4 w-4" aria-hidden="true" />
          </button>
        </div>
        <div
          v-if="qualityStore.ready && rating.count"
          class="flex flex-wrap items-baseline gap-x-2 gap-y-1 text-xs"
        >
          <span
            class="font-medium"
            :class="rating.level === 'good' ? 'text-green-600 dark:text-green-400' : rating.level === 'poor' ? 'text-red-600 dark:text-red-400' : 'text-text-secondary'"
          >{{ t(`proxies.quality.${rating.level}`) }}</span>
          <span class="text-text-secondary">{{ qualitySummary }}</span>
          <span v-if="!rating.complete" class="text-text-secondary">{{ t('proxies.quality.requiredSamples', { count: qualityStore.rules.samples }) }}</span>
        </div>

        <!-- Loading, errors and the latency chart. -->
        <p v-if="error" role="alert" class="text-sm text-red-600">{{ error }} <button type="button" class="underline" @click="load">{{ t('proxies.retry') }}</button></p>
        <p v-if="!loading && !points.length && !error" class="py-4 text-sm text-text-secondary">{{ t('proxies.history.empty') }}</p>
        <template v-if="loading || points.length">
          <ProbeTrendChart
            :points="points"
            :loading="loading && !points.length"
            latency-only
            :latency-threshold="qualityStore.ready ? qualityStore.rules.max_delay_ms : undefined"
          >
            <template #latency-value="{ point }">
              <span
                class="text-base font-semibold"
                :class="!qualityStore.ready ? 'text-text-primary' : point.reachable > 0 && point.avg_ms <= qualityStore.rules.max_delay_ms ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'"
              >
                {{ point.reachable > 0 ? `${point.avg_ms} ms` : t('proxies.noResult') }}
                <span
                  v-if="qualityStore.ready && point.reachable > 0"
                  class="ml-1 text-xs font-normal text-text-secondary"
                >{{ point.avg_ms === qualityStore.rules.max_delay_ms ? '=' : point.avg_ms < qualityStore.rules.max_delay_ms ? '<' : '>' }} {{ qualityStore.rules.max_delay_ms }} ms</span>
              </span>
            </template>
            <template #point-details="{ point }">
              <span
                class="self-start rounded-control bg-bg-tertiary px-1.5 py-0.5 text-[11px] text-text-secondary"
              >{{ t(`proxies.history.sources.${nodeProbeSource(point.source)}`) }}</span>
            </template>
          </ProbeTrendChart>
        </template>

        <!-- Chart legend and quality settings. -->
        <div
          class="flex flex-wrap items-center gap-x-4 gap-y-2 text-xs"
          :aria-label="t('proxies.history.legend')"
        >
          <RouterLink
            to="/dashboard/settings#node-quality"
            class="inline-flex items-center gap-1.5 rounded-control text-text-secondary focus-visible:outline-2 focus-visible:outline-primary-500"
            @click="panel?.hide()"
          >
            <svg
              v-if="qualityStore.ready"
              viewBox="0 0 28 12"
              class="h-3 w-7 shrink-0 text-primary-500"
              aria-hidden="true"
            >
              <line
                x1="0"
                x2="28"
                y1="6"
                y2="6"
                stroke="currentColor"
                stroke-width="1"
                stroke-dasharray="5 4"
              />
            </svg>
            <span class="text-primary-600 underline dark:text-primary-400">{{ t('proxies.quality.settings') }}</span>
            <span v-if="qualityStore.ready">{{ qualityStore.rules.max_delay_ms }} ms</span>
          </RouterLink>
          <span class="inline-flex items-center gap-1.5 text-text-secondary">
            <svg
              viewBox="0 0 28 12"
              class="h-3 w-7 shrink-0 text-gray-500 dark:text-gray-400"
              aria-hidden="true"
            >
              <line x1="0" x2="28" y1="6" y2="6" stroke="currentColor" stroke-width="1.75" />
            </svg>
            {{ t('proxies.history.actualLatency') }}
          </span>
        </div>

        <!-- Individual probe records. -->
        <template v-if="points.length">
          <p v-if="points.length === 1" class="text-xs text-text-secondary">{{ t('proxies.history.single') }}</p>
          <details>
            <summary
              class="cursor-pointer text-xs text-primary-600 focus-visible:outline-2 focus-visible:outline-primary-500"
            >{{ t('proxies.history.samples') }}</summary>
            <ul class="mt-2 max-h-40 space-y-1 overflow-y-auto text-xs tabular-nums">
              <li
                v-for="(point, index) in recent"
                :key="`${point.time}-${index}`"
                class="flex flex-wrap justify-between gap-x-3 gap-y-1"
              >
                <time :datetime="point.time">{{ new Date(point.time).toLocaleString(locale) }}</time>
                <span class="rounded-control bg-bg-tertiary px-1.5 py-0.5 text-text-secondary">{{ t(`proxies.history.sources.${nodeProbeSource(point.source)}`) }}</span>
                <span :class="(point.success ?? point.delay > 0) ? 'text-text-primary' : 'text-red-600'">{{ (point.success ?? point.delay > 0) ? `${point.delay} ms` : t('proxies.noResult') }}</span>
              </li>
            </ul>
          </details>
        </template>
      </div>
    </Popover>
  </span>
</template>
