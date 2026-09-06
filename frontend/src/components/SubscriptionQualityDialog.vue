<script setup lang="ts">
/**
 * One subscription's quality: the trend, and the latest per-node detail.
 *
 * The trend answers "is this provider worth renewing" — a question only time
 * can answer. The node table answers "what is wrong right now", which the
 * trend cannot, and which is the first thing anyone asks after seeing a dip.
 */
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowPathIcon, BoltIcon, CheckCircleIcon, MinusCircleIcon, XCircleIcon } from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import ProbeTrendChart from './ProbeTrendChart.vue'
import { Dialog } from '../volt'
import { subProbeService } from '../services'
import { useNotify } from '../composables/useNotify'
import type { Subscription } from '../types/api'
import type { ProbeNodeResult, ProbePoint, ProbeRange } from '../types/subprobe'
import { availabilityRatio, formatAvailability, formatLatency } from '../utils/probeChart'
import { qualityStepColors } from '../utils/qualitySteps'
import SegmentedProgress from './SegmentedProgress.vue'

const props = defineProps<{
  modelValue: boolean
  subscription: Subscription | null
}>()
const emit = defineEmits<{ 'update:modelValue': [boolean] }>()

const { t, locale } = useI18n()
const notify = useNotify()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const range = ref<ProbeRange>('24h')
const ranges: ProbeRange[] = ['1h', '6h', '24h', '7d', '30d']

const points = ref<ProbePoint[]>([])
const loadingHistory = ref(false)
const nodes = ref<ProbeNodeResult[]>([])
const nodesAvailable = ref(false)
const nodesAt = ref<string>('')
/**
 * The latest SINGLE run's aggregate, from the nodes endpoint.
 *
 * Not taken from the last chart point: on any bucketed range that point is a
 * sum over several runs, so its counts describe no run that ever happened. It
 * read "331/333 nodes" for a subscription with 37 — the ratio was right and the
 * denominator was nine runs deep.
 */
const nodesSample = ref<Omit<ProbePoint, 'at'> | null>(null)
const loadingNodes = ref(false)
const probing = ref(false)

async function loadHistory() {
  const subscription = props.subscription
  if (!subscription) return
  loadingHistory.value = true
  try {
    const { data } = await subProbeService.getHistory(subscription.id, range.value)
    points.value = data.points ?? []
  } catch (err) {
    notify.apiError(err, t('subProbe.notify.historyFailed'))
  } finally {
    loadingHistory.value = false
  }
}

async function loadNodes() {
  const subscription = props.subscription
  if (!subscription) return
  loadingNodes.value = true
  try {
    const { data } = await subProbeService.getNodes(subscription.id)
    nodesAvailable.value = data.available
    nodes.value = data.results ?? []
    nodesAt.value = data.at ?? ''
    nodesSample.value = data.sample ?? null
  } catch (err) {
    notify.apiError(err, t('subProbe.notify.nodesFailed'))
  } finally {
    loadingNodes.value = false
  }
}

// Opening the dialog, or switching subscription, reloads both halves. Watching
// the subscription too matters: the dialog is reused across rows, so without it
// the second row opened would show the first row's chart.
watch(
  () => [props.modelValue, props.subscription?.id] as const,
  ([open]) => {
    if (!open) return
    void loadHistory()
    void loadNodes()
  },
  { immediate: true },
)

watch(range, () => {
  if (props.modelValue) void loadHistory()
})

/**
 * Probe now, then reload both halves.
 *
 * Can take a minute or more on a large feed of dead nodes (measured: 103
 * unreachable nodes take ~65s at the server's 8-way concurrency), so the
 * button stays in its loading state rather than assuming this is quick.
 */
async function probeNow() {
  const subscription = props.subscription
  if (!subscription || probing.value) return
  probing.value = true
  try {
    const { data } = await subProbeService.run(subscription.id)
    notify.success(
      t('subProbe.notify.probed', {
        reachable: data.sample.reachable,
        total: data.sample.total,
        latency: data.sample.reachable > 0 ? formatLatency(data.sample.avg_ms) : '—',
      }),
    )
    await Promise.all([loadHistory(), loadNodes()])
  } catch (err) {
    notify.apiError(err, t('subProbe.notify.probeFailed'))
  } finally {
    probing.value = false
  }
}

/**
 * The headline figures: the latest actual run.
 *
 * Prefers the in-memory snapshot, which is always ONE run. The last chart point
 * is the fallback for a panel restarted since the last sweep — and it is only a
 * fallback because on a bucketed range its counts are summed over several runs
 * and describe no single measurement.
 */
const latest = computed<Omit<ProbePoint, 'at'> | null>(
  () => nodesSample.value ?? points.value[points.value.length - 1] ?? null,
)

const latestClass = computed(() => {
  const point = latest.value
  if (!point) return 'text-gray-400 dark:text-gray-500'
  const ratio = availabilityRatio(point)
  if (ratio >= 0.9) return 'text-green-600 dark:text-green-400'
  if (ratio >= 0.5) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
})

/**
 * Nodes worst-first: the reason anyone opens this table is to find what broke,
 * and a table sorted by tag makes them read all 103 rows to find the three.
 * Within a group, slowest first for the same reason.
 */
const sortedNodes = computed(() =>
  [...nodes.value].sort((a, b) => {
    const rank = (n: ProbeNodeResult) => (n.skipped ? 1 : n.error ? 0 : 2)
    const diff = rank(a) - rank(b)
    if (diff !== 0) return diff
    return (b.delay_ms ?? 0) - (a.delay_ms ?? 0)
  }),
)

const nodesAtLabel = computed(() =>
  nodesAt.value ? new Date(nodesAt.value).toLocaleString(locale.value) : '',
)

const probeTarget = computed(
  () => props.subscription?.probe_url || 'https://www.gstatic.com/generate_204',
)
</script>

<template>
  <Dialog
    v-model:visible="visible"
    :header="$t('subProbe.dialog.title', { name: subscription?.name ?? '' })"
    modal
    class="w-full max-w-4xl"
  >
    <div class="space-y-4">
      <!-- Headline + controls -->
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div class="flex items-baseline gap-3">
          <template v-if="latest">
            <span class="text-2xl font-bold" :class="latestClass">
              {{ formatAvailability(latest) }}
            </span>
            <SegmentedProgress
              :percent="100"
              :steps="5"
              :stroke-color="qualityStepColors(latest.reachable, latest.total)"
              size="md"
              :aria-label="
                $t('subProbe.nodesTested', { reachable: latest.reachable, total: latest.total })
              "
            />
            <span class="text-sm text-gray-500 dark:text-gray-400">
              {{ latest.reachable }}/{{ latest.total }} · {{ latest.reachable > 0 ? formatLatency(latest.avg_ms) : '—' }}
            </span>
          </template>
          <span v-else class="text-sm text-gray-400 dark:text-gray-500">
            {{ $t('subProbe.dialog.noData') }}
          </span>
        </div>

        <div class="flex items-center gap-2">
          <!--
            Named ranges, not a date picker: the server chooses the bucket
            alongside the window, and a client picking one without the other is
            how a 30-day request arrives unbucketed.
          -->
          <div class="inline-flex rounded-control border border-gray-200 dark:border-gray-700">
            <button
              v-for="option in ranges"
              :key="option"
              type="button"
              class="cursor-pointer px-2 py-1 text-xs font-medium transition-colors first:rounded-l-control last:rounded-r-control"
              :class="
                range === option
                  ? 'bg-primary-600 text-white'
                  : 'text-gray-600 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700'
              "
              @click="range = option"
            >
              {{ $t(`subProbe.range.${option}`) }}
            </button>
          </div>

          <Button variant="secondary" size="sm" :loading="probing" :disabled="probing" @click="probeNow">
            <BoltIcon v-if="!probing" class="h-4 w-4" />
            {{ $t('subProbe.dialog.probeNow') }}
          </Button>
        </div>
      </div>

      <ProbeTrendChart :points="points" :loading="loadingHistory" />

      <!-- What is actually being measured, so the numbers can be interpreted. -->
      <p class="text-xs text-gray-400 dark:text-gray-500">
        {{ $t('subProbe.dialog.target') }}:
        <span class="font-mono">{{ probeTarget }}</span>
      </p>

      <!-- Latest run, per node -->
      <div>
        <div class="mb-2 flex items-center justify-between gap-2">
          <h4 class="text-sm font-semibold text-gray-700 dark:text-gray-300">
            {{ $t('subProbe.dialog.nodes') }}
          </h4>
          <div class="flex items-center gap-2">
            <span v-if="nodesAtLabel" class="text-xs text-gray-400 dark:text-gray-500">
              {{ nodesAtLabel }}
            </span>
            <button
              type="button"
              class="cursor-pointer rounded p-0.5 text-gray-400 transition hover:text-primary-600 dark:hover:text-primary-400"
              :title="$t('common.refresh')"
              :aria-label="$t('common.refresh')"
              @click="loadNodes"
            >
              <ArrowPathIcon class="h-4 w-4" :class="{ 'animate-spin': loadingNodes }" />
            </button>
          </div>
        </div>

        <!--
          The per-node detail is in memory server-side, so it is genuinely
          absent until the first sweep after a restart. Saying so beats an
          empty table, which reads as "all nodes fine".
        -->
        <p v-if="!nodesAvailable && !loadingNodes" class="text-xs text-gray-400 dark:text-gray-500">
          {{ $t('subProbe.dialog.nodesUnavailable') }}
        </p>

        <div v-else class="max-h-64 overflow-y-auto rounded-control border border-gray-100 dark:border-gray-700">
          <table class="w-full text-xs">
            <tbody>
              <tr
                v-for="node in sortedNodes"
                :key="node.tag"
                class="border-b border-gray-50 last:border-b-0 dark:border-gray-700/50"
              >
                <td class="w-6 py-1 pl-2">
                  <CheckCircleIcon
                    v-if="!node.error && !node.skipped"
                    class="h-3.5 w-3.5 text-green-500"
                  />
                  <MinusCircleIcon v-else-if="node.skipped" class="h-3.5 w-3.5 text-gray-400" />
                  <XCircleIcon v-else class="h-3.5 w-3.5 text-red-500" />
                </td>
                <td class="py-1 pr-2">
                  <span class="block truncate text-gray-700 dark:text-gray-300" :title="node.tag">
                    {{ node.tag }}
                  </span>
                </td>
                <td class="w-32 py-1 pr-2 text-right">
                  <span v-if="!node.error && !node.skipped" class="font-mono text-gray-600 dark:text-gray-300">
                    {{ formatLatency(node.delay_ms ?? 0) }}
                  </span>
                  <!--
                    A node the panel could not test is labelled as such, not as
                    down: it is in the config but absent from the running
                    sing-box, which is an unapplied config rather than a
                    provider failure — and it is excluded from the percentage
                    above for the same reason.
                  -->
                  <span v-else-if="node.skipped" class="text-gray-400 dark:text-gray-500">
                    {{ $t('subProbe.dialog.untestable') }}
                  </span>
                  <span v-else class="text-red-500 dark:text-red-400">
                    {{ $t('subProbe.dialog.unreachable') }}
                  </span>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>

    <template #footer>
      <Button variant="secondary" @click="visible = false">{{ $t('common.close') }}</Button>
    </template>
  </Dialog>
</template>
