<script setup lang="ts">
/**
 * A subscription's availability and latency over time.
 *
 * Two stacked panels sharing one x-axis rather than one chart with two y-axes:
 * the quantities are unrelated and in unrelated units, and a dual-axis chart
 * makes the reader identify which scale each line belongs to before either
 * number means anything.
 *
 * All geometry comes from `utils/probeChart` — a pure, tested module — so this
 * file is markup. Same split as the route flow diagram.
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import Tooltip from './Tooltip.vue'
import Skeleton from './Skeleton.vue'
import { useI18n } from 'vue-i18n'
import type { ProbePoint } from '../types/subprobe'
import {
  availabilityRatio,
  formatAvailability,
  formatLatency,
  layoutProbeChart,
  type PlacedPoint,
} from '../utils/probeChart'

const props = defineProps<{
  points: ProbePoint[]
  /** Shown in place of the chart while the first read is in flight. */
  loading?: boolean
  /** Node history has latency observations rather than subscription totals. */
  latencyOnly?: boolean
  latencyThreshold?: number
}>()

const { t, locale } = useI18n()

/**
 * A fixed viewBox scaled by CSS, exactly as the route diagram does it: the
 * alternative is a ResizeObserver, a repaint per reflow, and a wrong first
 * frame.
 */
const WIDTH = 720
const PANEL_HEIGHT = 132

const availabilityLayout = computed(() =>
  layoutProbeChart(props.points, { width: WIDTH, height: PANEL_HEIGHT, minLatencyMax: props.latencyThreshold }),
)

const thresholdLine = computed(() => {
  const value = props.latencyThreshold
  if (value === undefined || !Number.isFinite(value) || value <= 0) return null
  const { plot, latency } = availabilityLayout.value
  if (latency.max <= 0) return null
  const y = plot.y + plot.height * (1 - value / latency.max)
  return { value, y, labelY: y < 20 ? y + 11 : y - 4 }
})

/** Percentage gridlines. 0/50/100 only — more lines than data is noise. */
const availabilityGrid = computed(() => {
  const { plot } = availabilityLayout.value
  return [0, 50, 100].map((value) => ({
    value,
    y: plot.y + plot.height - (value / 100) * plot.height,
    label: `${value}%`,
  }))
})

/** Latency gridlines at 0 / half / full of the auto-scaled axis. */
const latencyGrid = computed(() => {
  const layout = availabilityLayout.value
  const { plot } = layout
  const max = layout.latency.max
  if (max <= 0) return []
  return [0, max / 2, max].map((value) => ({
    value,
    y: plot.y + plot.height - (value / max) * plot.height,
    label: formatLatency(value),
  }))
})

/**
 * The point the pointer is nearest, or null.
 *
 * Hover is on the SHARED x position rather than per-panel, so pointing at
 * either chart reads out both numbers for the same moment — which is the
 * comparison the two panels exist to support.
 */
const hovered = ref<PlacedPoint | null>(null)
const cursor = ref({ x: 0, y: 0 })
const keyboardHover = ref(false)
let leaveTimer: ReturnType<typeof setTimeout> | undefined
let overTooltip = false
function cancelLeave() { clearTimeout(leaveTimer) }
function clearHover() { cancelLeave(); overTooltip = false; hovered.value = null; keyboardHover.value = false }
function scheduleLeave() { cancelLeave(); if (!overTooltip) leaveTimer = setTimeout(clearHover, 600) }
function tooltipHover(hovering: boolean) { overTooltip = hovering; hovering ? cancelLeave() : scheduleLeave() }
onBeforeUnmount(cancelLeave)
watch(() => [props.points, props.loading], () => {
  if (props.loading) { clearHover(); return }
  const previous = hovered.value?.point
  if (!previous) return
  const sameSample = (point: ProbePoint) => point.at === previous.at && point.source === previous.source
    && point.avg_ms === previous.avg_ms && point.min_ms === previous.min_ms && point.max_ms === previous.max_ms
    && point.total === previous.total && point.reachable === previous.reachable
  const replacement = availabilityLayout.value.availability.segments
    .flatMap(segment => segment.points).find(dot => sameSample(dot.point))
  if (replacement) hovered.value = replacement
  else clearHover()
})

const failedPoints = computed(() => availabilityLayout.value.availability.segments
  .flatMap(segment => segment.points).filter(dot => dot.point.reachable === 0))

function sampleLabel(dot: PlacedPoint) {
  const value = dot.point.reachable > 0 ? formatLatency(dot.point.avg_ms) : t('proxies.noResult')
  return `${new Date(dot.point.at).toLocaleString(locale.value)} · ${value}`
}

function focusPoint(event: FocusEvent, dot: PlacedPoint) {
  cancelLeave()
  const box = (event.currentTarget as SVGElement).getBoundingClientRect()
  cursor.value = { x: box.left + box.width / 2, y: box.top }
  keyboardHover.value = true
  hovered.value = dot
}

function onPointerMove(event: PointerEvent) {
  cancelLeave()
  keyboardHover.value = false
  const layout = availabilityLayout.value
  if (layout.isEmpty) return

  const target = event.currentTarget as SVGSVGElement
  const rect = target.getBoundingClientRect()
  if (rect.width === 0) return
  // The SVG is scaled by CSS, so client pixels must be converted back into
  // viewBox units before they can be compared with laid-out coordinates.
  const x = ((event.clientX - rect.left) / rect.width) * WIDTH

  let nearest: PlacedPoint | null = null
  let bestDistance = Infinity
  for (const segment of layout.availability.segments) {
    for (const placed of segment.points) {
      const distance = Math.abs(placed.x - x)
      if (distance < bestDistance) {
        bestDistance = distance
        nearest = placed
      }
    }
  }
  // Keep a sample's bubble stationary so the pointer can move into it.
  if (hovered.value !== nearest) cursor.value = { x: event.clientX, y: event.clientY }
  hovered.value = nearest
}

/** The latency point matching the hovered moment, if that run had one. */
const hoveredLatency = computed(() => {
  const target = hovered.value
  if (!target) return null
  for (const segment of availabilityLayout.value.latency.segments) {
    for (const placed of segment.points) {
      if (placed.point.at === target.point.at) return placed
    }
  }
  return null
})

const hoveredTime = computed(() => {
  const target = hovered.value
  if (!target) return ''
  return new Date(target.point.at).toLocaleString(locale.value)
})

/** Colour follows the value: this is a health chart, so a bad number is red. */
function availabilityClass(point: ProbePoint): string {
  const ratio = availabilityRatio(point)
  if (ratio >= 0.9) return 'text-green-600 dark:text-green-400'
  if (ratio >= 0.5) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}
</script>

<template>
  <div class="relative">
    <div v-if="loading" class="flex h-[260px] flex-col justify-between gap-4 py-4" role="status" :aria-label="$t('common.loading')">
      <Skeleton class="h-3 w-20" />
      <Skeleton class="h-20 w-full" />
      <Skeleton class="h-3 w-24" />
      <Skeleton class="h-20 w-full" />
    </div>

    <!--
      An empty chart has to say WHY it is empty. The commonest reason is that
      the prober has not run yet (it waits ~45s after start, then samples on
      its interval), which is indistinguishable from a broken feature unless
      the panel says so.
    -->
    <div
      v-else-if="availabilityLayout.isEmpty"
      class="flex h-[260px] flex-col items-center justify-center gap-1 text-center"
    >
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ $t('subProbe.chart.empty') }}</p>
      <p class="text-xs text-gray-400 dark:text-gray-500">{{ $t('subProbe.chart.emptyHint') }}</p>
    </div>

    <div v-else class="space-y-1">
      <!-- Availability panel -->
      <div v-if="!latencyOnly">
        <p class="mb-0.5 text-[11px] font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
          {{ $t('subProbe.chart.availability') }}
        </p>
        <svg
          :viewBox="`0 0 ${WIDTH} ${PANEL_HEIGHT}`"
          class="w-full"
          role="img"
          :aria-label="$t('subProbe.chart.availabilityAria')"
          @pointermove="onPointerMove"
          @pointerleave="scheduleLeave"
          @pointercancel="clearHover"
          @pointerdown="onPointerMove"
        >
          <line
            v-for="line in availabilityGrid"
            :key="`ag-${line.value}`"
            :x1="availabilityLayout.plot.x"
            :x2="availabilityLayout.plot.x + availabilityLayout.plot.width"
            :y1="line.y"
            :y2="line.y"
            class="stroke-gray-200 dark:stroke-gray-700"
            stroke-width="1"
          />
          <text
            v-for="line in availabilityGrid"
            :key="`agl-${line.value}`"
            :x="availabilityLayout.plot.x - 6"
            :y="line.y + 3"
            text-anchor="end"
            class="fill-gray-400 text-[9px] dark:fill-gray-500"
          >
            {{ line.label }}
          </text>

          <g v-for="(segment, i) in availabilityLayout.availability.segments" :key="`a-${i}`">
            <path :d="segment.area" class="fill-primary-500/12" />
            <path
              :d="segment.line"
              fill="none"
              class="stroke-primary-600 dark:stroke-primary-400"
              stroke-width="1.75"
              stroke-linejoin="round"
              stroke-linecap="round"
            />
            <!-- A lone sample has no line to see, so it is drawn as a dot. -->
            <circle
              v-for="(dot, d) in segment.points.length === 1 ? segment.points : []"
              :key="`ad-${i}-${d}`"
              :cx="dot.x"
              :cy="dot.y"
              r="2.5"
              class="fill-primary-600 dark:fill-primary-400"
            />
          </g>

          <g v-if="hovered">
            <line
              :x1="hovered.x"
              :x2="hovered.x"
              :y1="availabilityLayout.plot.y"
              :y2="availabilityLayout.plot.y + availabilityLayout.plot.height"
              class="stroke-gray-400 dark:stroke-gray-500"
              stroke-width="1"
              stroke-dasharray="3 3"
            />
            <circle :cx="hovered.x" :cy="hovered.y" r="3.5" class="fill-primary-600 dark:fill-primary-400" />
          </g>
        </svg>
      </div>

      <!-- Latency panel, sharing the x-axis above -->
      <div>
        <p class="mb-0.5 text-[11px] font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
          {{ latencyOnly ? $t('proxies.history.latency') : $t('subProbe.chart.latency') }}
        </p>
        <svg
          :viewBox="`0 0 ${WIDTH} ${PANEL_HEIGHT}`"
          class="w-full"
          role="img"
          :aria-label="$t('subProbe.chart.latencyAria')"
          @pointermove="onPointerMove"
          @pointerleave="scheduleLeave"
          @pointercancel="clearHover"
          @pointerdown="onPointerMove"
        >
          <line
            v-for="line in latencyGrid"
            :key="`lg-${line.value}`"
            :x1="availabilityLayout.plot.x"
            :x2="availabilityLayout.plot.x + availabilityLayout.plot.width"
            :y1="line.y"
            :y2="line.y"
            class="stroke-gray-200 dark:stroke-gray-700"
            stroke-width="1"
          />
          <text
            v-for="line in latencyGrid"
            :key="`lgl-${line.value}`"
            :x="availabilityLayout.plot.x - 6"
            :y="line.y + 3"
            text-anchor="end"
            class="fill-gray-400 text-[9px] dark:fill-gray-500"
          >
            {{ line.label }}
          </text>

          <g v-if="thresholdLine" data-latency-threshold>
            <line :x1="availabilityLayout.plot.x" :x2="availabilityLayout.plot.x + availabilityLayout.plot.width"
              :y1="thresholdLine.y" :y2="thresholdLine.y" class="stroke-primary-500" stroke-width="1" stroke-dasharray="5 4" />
            <text :x="availabilityLayout.plot.x + availabilityLayout.plot.width" :y="thresholdLine.labelY"
              text-anchor="end" class="fill-primary-600 text-[9px] dark:fill-primary-400">{{ thresholdLine.value }} ms</text>
          </g>
          <g v-for="(segment, i) in availabilityLayout.latency.segments" :key="`l-${i}`">
            <path
              :d="segment.line"
              fill="none"
              class="stroke-gray-500 dark:stroke-gray-400"
              stroke-width="1.75"
              stroke-linejoin="round"
              stroke-linecap="round"
            />
            <circle
              v-for="(dot, d) in latencyOnly || segment.points.length === 1 ? segment.points : []"
              :key="`ld-${i}-${d}`"
              :cx="dot.x"
              :cy="dot.y"
              r="2.5"
              class="fill-gray-500 dark:fill-gray-400"
              :tabindex="latencyOnly ? 0 : undefined"
              :aria-label="latencyOnly ? sampleLabel(dot) : undefined"
              @focus="focusPoint($event, dot)"
              @blur="scheduleLeave"
            />
          </g>

          <circle v-for="(dot, i) in latencyOnly ? failedPoints : []" :key="`failed-${i}`"
            :cx="dot.x" :cy="availabilityLayout.plot.y + availabilityLayout.plot.height"
            r="3" class="fill-red-500" tabindex="0" :aria-label="sampleLabel(dot)"
            @focus="focusPoint($event, dot)" @blur="scheduleLeave" />

          <g v-if="hoveredLatency">
            <line
              :x1="hoveredLatency.x"
              :x2="hoveredLatency.x"
              :y1="availabilityLayout.plot.y"
              :y2="availabilityLayout.plot.y + availabilityLayout.plot.height"
              class="stroke-gray-400 dark:stroke-gray-500"
              stroke-width="1"
              stroke-dasharray="3 3"
            />
            <circle
              :cx="hoveredLatency.x"
              :cy="hoveredLatency.y"
              r="3.5"
              class="fill-gray-500 dark:fill-gray-400"
            />
          </g>

          <!-- The shared time axis, labelled once under the lower panel. -->
          <text
            v-for="(tick, i) in availabilityLayout.xTicks"
            :key="`x-${i}`"
            :x="tick.x"
            :y="PANEL_HEIGHT - 4"
            :text-anchor="i === 0 ? 'start' : i === availabilityLayout.xTicks.length - 1 ? 'end' : 'middle'"
            class="fill-gray-400 text-[9px] dark:fill-gray-500"
          >
            {{ tick.label }}
          </text>
        </svg>
      </div>

      <Tooltip :visible="!!hovered && !loading" :point="cursor" :width="240" :hover-delay="keyboardHover ? 0 : 320"
        interactive :hide-delay="600" @hover-change="tooltipHover" @dismiss="clearHover">
        <template #content><div v-if="hovered" data-probe-tooltip>
        <div class="flex flex-col gap-1">
          <span class="font-mono text-gray-500 dark:text-gray-400">{{ hoveredTime }}</span>
          <span v-if="!latencyOnly" :class="availabilityClass(hovered.point)">
            <span class="font-semibold">{{ formatAvailability(hovered.point) }}</span>
            <span class="ml-1 text-gray-500 dark:text-gray-400">
              ({{ hovered.point.reachable }}/{{ hovered.point.total }})
            </span>
          </span>
          <slot name="latency-value" :point="hovered.point">
            <span class="text-gray-600 dark:text-gray-300">
              {{ latencyOnly ? $t('proxies.history.latency') + ': ' : '' }}{{ hovered.point.reachable > 0 ? (hovered.point.avg_ms === 0 ? '0 ms' : formatLatency(hovered.point.avg_ms)) : $t(latencyOnly ? 'proxies.noResult' : 'subProbe.chart.noLatency') }}
            </span>
          </slot>
          <span
            v-if="hovered.point.reachable > 0 && hovered.point.max_ms > hovered.point.min_ms"
            class="text-gray-400 dark:text-gray-500"
          >
            {{ formatLatency(hovered.point.min_ms) }} – {{ formatLatency(hovered.point.max_ms) }}
          </span>
          <slot name="point-details" :point="hovered.point" />
        </div>
        </div></template>
      </Tooltip>
    </div>
  </div>
</template>
