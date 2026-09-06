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
import { computed, ref } from 'vue'
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
}>()

const { locale } = useI18n()

/**
 * A fixed viewBox scaled by CSS, exactly as the route diagram does it: the
 * alternative is a ResizeObserver, a repaint per reflow, and a wrong first
 * frame.
 */
const WIDTH = 720
const PANEL_HEIGHT = 132

const availabilityLayout = computed(() =>
  layoutProbeChart(props.points, { width: WIDTH, height: PANEL_HEIGHT }),
)

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

function onPointerMove(event: PointerEvent) {
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
    <div v-if="loading" class="flex h-[260px] items-center justify-center">
      <div class="h-8 w-8 animate-spin rounded-pill border-b-2 border-primary-600"></div>
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
      <div>
        <p class="mb-0.5 text-[11px] font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
          {{ $t('subProbe.chart.availability') }}
        </p>
        <svg
          :viewBox="`0 0 ${WIDTH} ${PANEL_HEIGHT}`"
          class="w-full"
          role="img"
          :aria-label="$t('subProbe.chart.availabilityAria')"
          @pointermove="onPointerMove"
          @pointerleave="hovered = null"
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
          {{ $t('subProbe.chart.latency') }}
        </p>
        <svg
          :viewBox="`0 0 ${WIDTH} ${PANEL_HEIGHT}`"
          class="w-full"
          role="img"
          :aria-label="$t('subProbe.chart.latencyAria')"
          @pointermove="onPointerMove"
          @pointerleave="hovered = null"
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

          <g v-for="(segment, i) in availabilityLayout.latency.segments" :key="`l-${i}`">
            <path
              :d="segment.line"
              fill="none"
              class="stroke-amber-500 dark:stroke-amber-400"
              stroke-width="1.75"
              stroke-linejoin="round"
              stroke-linecap="round"
            />
            <circle
              v-for="(dot, d) in segment.points.length === 1 ? segment.points : []"
              :key="`ld-${i}-${d}`"
              :cx="dot.x"
              :cy="dot.y"
              r="2.5"
              class="fill-amber-500 dark:fill-amber-400"
            />
          </g>

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
              class="fill-amber-500 dark:fill-amber-400"
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

      <!--
        The readout is a fixed row rather than a floating tooltip: it never
        covers the line it describes, and it keeps the last hovered values on
        screen after the pointer leaves the chart, which is what makes two
        moments comparable at all.
      -->
      <div
        class="flex min-h-[2rem] flex-wrap items-center gap-x-4 gap-y-1 border-t border-gray-100 pt-1.5 text-xs dark:border-gray-700"
      >
        <template v-if="hovered">
          <span class="font-mono text-gray-500 dark:text-gray-400">{{ hoveredTime }}</span>
          <span :class="availabilityClass(hovered.point)">
            <span class="font-semibold">{{ formatAvailability(hovered.point) }}</span>
            <span class="ml-1 text-gray-500 dark:text-gray-400">
              ({{ hovered.point.reachable }}/{{ hovered.point.total }})
            </span>
          </span>
          <span class="text-gray-600 dark:text-gray-300">
            {{ hovered.point.reachable > 0 ? formatLatency(hovered.point.avg_ms) : $t('subProbe.chart.noLatency') }}
          </span>
          <span
            v-if="hovered.point.reachable > 0 && hovered.point.max_ms > hovered.point.min_ms"
            class="text-gray-400 dark:text-gray-500"
          >
            {{ formatLatency(hovered.point.min_ms) }} – {{ formatLatency(hovered.point.max_ms) }}
          </span>
        </template>
        <span v-else class="text-gray-400 dark:text-gray-500">{{ $t('subProbe.chart.hoverHint') }}</span>
      </div>
    </div>
  </div>
</template>
