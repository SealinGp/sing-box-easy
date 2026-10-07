<script setup lang="ts">
/**
 * Overview card: how much traffic sing-box is carrying right now, and over the
 * last minute.
 *
 * TWO PANELS, ONE TIME AXIS — NOT ONE CHART WITH TWO Y-SCALES
 * ──────────────────────────────────────────────────────────
 * Download and upload are the same quantity in the same unit, so they share a
 * panel and one axis: that is what lets the eye compare them. The connection
 * count is a different quantity in a different unit. Drawn on the speed chart
 * with its own right-hand scale it would cross the speed lines at points that
 * mean nothing, and every reader would have to work out which line belongs to
 * which axis before either number meant anything. So it gets its own short
 * panel underneath, on the same time axis — the same decision
 * `ProbeTrendChart` makes for availability and latency.
 *
 * No chart library, for the reason that file gives: these are two line panels,
 * and a dependency to draw them would outweigh the card.
 *
 * The SVGs stretch to the card (`preserveAspectRatio="none"`) with
 * `vector-effect: non-scaling-stroke`, so lines keep their weight at any
 * width. Text is NOT in the SVG for that reason — stretched glyphs — and sits
 * in HTML over the plot instead.
 */
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useConnectionStats } from '../composables/useConnectionStats'
import { useServiceStore } from '../stores/service'
import { formatBytes } from '../utils/formatBytes'
import { formatRate } from '../utils/flowOverlay'
import {
  WINDOW_SAMPLES, countCeiling, rateCeiling, sampleIndexAt, sampleX, seriesPath, valueY,
} from '../utils/trafficChart'

const { t } = useI18n()
const serviceStore = useServiceStore()
const running = computed(() => serviceStore.status?.status === 'running')

const { totals, history, error } = useConnectionStats(running, () => t('overview.connections.streamFailed'))

// viewBox units. Only the ratio between the two heights matters: the speed
// panel is the subject, the count panel is context.
const WIDTH = 600
const SPEED_HEIGHT = 110
const COUNT_HEIGHT = 44
const speedSize = { width: WIDTH, height: SPEED_HEIGHT }
const countSize = { width: WIDTH, height: COUNT_HEIGHT }

const speedTop = computed(() => rateCeiling(Math.max(0, ...history.value.map((s) => Math.max(s.down, s.up)))))
const countTop = computed(() => countCeiling(Math.max(0, ...history.value.map((s) => s.connections))))

const downPath = computed(() => seriesPath(history.value.map((s) => s.down), speedTop.value, speedSize))
const upPath = computed(() => seriesPath(history.value.map((s) => s.up), speedTop.value, speedSize))
const countPath = computed(() => seriesPath(history.value.map((s) => s.connections), countTop.value, countSize))

/**
 * Hover. One crosshair across BOTH panels — they share the time axis, so a
 * moment is a column through the whole card — and a small card beside the
 * crosshair gives that moment's three values.
 *
 * The values appear AT the pointer. An earlier version rewrote the readout row
 * at the top of the card instead, which meant pointing at one place and reading
 * the answer in another; the readout now always means "now" and never changes
 * under a hover.
 */
const hoverIndex = ref(-1)

const onMove = (event: PointerEvent) => {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  if (rect.width <= 0) return
  hoverIndex.value = sampleIndexAt((event.clientX - rect.left) / rect.width, history.value.length)
}
const onLeave = () => {
  hoverIndex.value = -1
}

const hovered = computed(() => (hoverIndex.value >= 0 ? history.value[hoverIndex.value] ?? null : null))
const latest = computed(() => history.value[history.value.length - 1] ?? null)

/**
 * Which side of the crosshair the tooltip opens on. It flips past the middle
 * so it never runs off the card's edge — and since data fills from the right,
 * that also keeps it over the older part of the line, not the newest.
 */
const tipFlipped = computed(() => hoverX.value > WIDTH / 2)

const hoverX = computed(() =>
  hovered.value ? sampleX(hoverIndex.value, history.value.length, WIDTH) : 0)
/** Marker positions as percentages, for the HTML dots over the stretched SVGs. */
const marker = computed(() => {
  const point = hovered.value
  if (!point) return null
  return {
    left: `${(hoverX.value / WIDTH) * 100}%`,
    down: `${(valueY(point.down, speedTop.value, SPEED_HEIGHT) / SPEED_HEIGHT) * 100}%`,
    up: `${(valueY(point.up, speedTop.value, SPEED_HEIGHT) / SPEED_HEIGHT) * 100}%`,
    count: `${(valueY(point.connections, countTop.value, COUNT_HEIGHT) / COUNT_HEIGHT) * 100}%`,
  }
})

const when = computed(() => {
  const point = hovered.value
  const now = latest.value
  if (!point || !now) return t('overview.connections.now')
  const seconds = Math.max(0, Math.round((now.at - point.at) / 1000))
  return seconds === 0 ? t('overview.connections.now') : t('overview.connections.ago', { n: seconds })
})

const rate = (value: number) => `${formatRate(value)}/s`

const summary = computed(() => {
  const point = latest.value
  return point
    ? t('overview.connections.chartSummary', {
        down: rate(point.down), up: rate(point.up), n: point.connections, seconds: WINDOW_SAMPLES,
      })
    : t('overview.connections.waiting')
})
</script>

<template>
  <div class="bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <div class="mb-2 flex items-baseline justify-between gap-3">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">{{ $t('overview.connections.title') }}</h3>
      <RouterLink to="/dashboard/connections" class="text-xs font-medium text-primary-600 hover:underline dark:text-primary-400">
        {{ $t('overview.connections.open') }}
      </RouterLink>
    </div>

    <p v-if="!running" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
      {{ $t('overview.connections.stopped') }}
    </p>

    <template v-else>
      <!--
        The legend IS the readout: each series is named once, beside its swatch
        and its current value, so identity never rests on colour alone and the
        numbers are in ink, not in the series colour.
      -->
      <div class="readout">
        <span class="readout-item">
          <i class="swatch swatch-down" aria-hidden="true"></i>
          <span class="readout-label">{{ $t('overview.connections.down') }}</span>
          <b class="readout-value">{{ latest ? rate(latest.down) : '–' }}</b>
        </span>
        <span class="readout-item">
          <i class="swatch swatch-up" aria-hidden="true"></i>
          <span class="readout-label">{{ $t('overview.connections.up') }}</span>
          <b class="readout-value">{{ latest ? rate(latest.up) : '–' }}</b>
        </span>
        <span class="readout-item">
          <i class="swatch swatch-count" aria-hidden="true"></i>
          <span class="readout-label">{{ $t('overview.connections.active') }}</span>
          <b class="readout-value">{{ latest ? latest.connections : '–' }}</b>
        </span>
      </div>

      <div
        class="plots"
        role="img"
        :aria-label="summary"
        @pointermove="onMove"
        @pointerleave="onLeave"
      >
        <!-- Speed: download (filled) and upload on one axis. -->
        <div class="plot plot-speed">
          <span class="axis-top">{{ rate(speedTop) }}</span>
          <svg :viewBox="`0 0 ${WIDTH} ${SPEED_HEIGHT}`" preserveAspectRatio="none" aria-hidden="true">
            <line class="grid" x1="0" :x2="WIDTH" :y1="SPEED_HEIGHT / 2" :y2="SPEED_HEIGHT / 2" />
            <line class="grid" x1="0" :x2="WIDTH" y1="0.5" y2="0.5" />
            <path v-if="downPath.area" :d="downPath.area" class="area area-down" />
            <path v-if="downPath.line" :d="downPath.line" class="line line-down" />
            <path v-if="upPath.line" :d="upPath.line" class="line line-up" />
            <line v-if="hovered" class="crosshair" :x1="hoverX" :x2="hoverX" y1="0" :y2="SPEED_HEIGHT" />
          </svg>
          <template v-if="marker">
            <i class="dot dot-down" :style="{ left: marker.left, top: marker.down }"></i>
            <i class="dot dot-up" :style="{ left: marker.left, top: marker.up }"></i>
          </template>
        </div>

        <!-- Connection count: its own panel, its own axis, the same time axis. -->
        <div class="plot plot-count">
          <span class="axis-top">{{ countTop }}</span>
          <svg :viewBox="`0 0 ${WIDTH} ${COUNT_HEIGHT}`" preserveAspectRatio="none" aria-hidden="true">
            <line class="grid" x1="0" :x2="WIDTH" y1="0.5" y2="0.5" />
            <path v-if="countPath.area" :d="countPath.area" class="area area-count" />
            <path v-if="countPath.line" :d="countPath.line" class="line line-count" />
            <line v-if="hovered" class="crosshair" :x1="hoverX" :x2="hoverX" y1="0" :y2="COUNT_HEIGHT" />
          </svg>
          <i v-if="marker" class="dot dot-count" :style="{ left: marker.left, top: marker.count }"></i>
        </div>

        <!--
          Rendered inside the plot area and positioned from the same percentage
          as the crosshair, so it tracks the sample, not the raw pointer: it
          steps with the data and does not jitter with every pixel of movement.
        -->
        <div v-if="hovered && marker" class="tip" :class="{ 'is-flipped': tipFlipped }" :style="{ left: marker.left }" role="status">
          <span class="tip-when">{{ when }}</span>
          <span class="tip-row">
            <i class="swatch swatch-down" aria-hidden="true"></i>
            <span class="tip-label">{{ $t('overview.connections.down') }}</span>
            <b>{{ rate(hovered.down) }}</b>
          </span>
          <span class="tip-row">
            <i class="swatch swatch-up" aria-hidden="true"></i>
            <span class="tip-label">{{ $t('overview.connections.up') }}</span>
            <b>{{ rate(hovered.up) }}</b>
          </span>
          <span class="tip-row">
            <i class="swatch swatch-count" aria-hidden="true"></i>
            <span class="tip-label">{{ $t('overview.connections.active') }}</span>
            <b>{{ hovered.connections }}</b>
          </span>
        </div>

        <div class="time-axis">
          <span>{{ $t('overview.connections.windowStart', { n: WINDOW_SAMPLES }) }}</span>
          <span>{{ $t('overview.connections.now') }}</span>
        </div>
      </div>

      <p v-if="error" class="mt-2 text-xs text-red-600 dark:text-red-400">{{ error }}</p>

      <dl class="totals">
        <div>
          <dt>{{ $t('overview.connections.totalDown') }}</dt>
          <dd>{{ totals ? formatBytes(totals.download_total) : '–' }}</dd>
        </div>
        <div>
          <dt>{{ $t('overview.connections.totalUp') }}</dt>
          <dd>{{ totals ? formatBytes(totals.upload_total) : '–' }}</dd>
        </div>
        <div>
          <dt>{{ $t('overview.connections.memory') }}</dt>
          <dd>{{ totals ? formatBytes(totals.memory) : '–' }}</dd>
        </div>
      </dl>
    </template>
  </div>
</template>

<style scoped>
/*
 * Series colours. Checked with the dataviz palette validator against this
 * card's surfaces (white / #151c25): lightness band, chroma floor, colour-
 * vision separation and >= 3:1 contrast all pass in both modes. Dark mode uses
 * its own steps of the same hues, not the light values dimmed.
 * Download is the app's brand blue — it is the series the card is about.
 */
.readout,
.plots {
  --series-down: #1575ff;
  --series-up: #eb6834;
  --series-count: #14996a;
}
@media (prefers-color-scheme: dark) {
  .readout,
  .plots {
    --series-down: #3987e5;
    --series-up: #d95926;
    --series-count: #199e70;
  }
}

.readout {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 4px 16px;
  margin-bottom: 8px;
  font-size: 12px;
}
.readout-item {
  display: inline-flex;
  align-items: baseline;
  gap: 5px;
}
.swatch {
  width: 10px;
  height: 3px;
  flex: none;
  align-self: center;
  border-radius: 2px;
}
.swatch-down { background: var(--series-down); }
.swatch-up { background: var(--series-up); }
.swatch-count { background: var(--series-count); }
.readout-label { color: var(--color-text-secondary); }
.readout-value {
  font-size: 14px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-primary);
}

.plots {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 6px;
  cursor: crosshair;
  touch-action: pan-y;
}
.plot { position: relative; }
.plot-speed { height: 110px; }
.plot-count { height: 44px; }
.plot svg {
  display: block;
  width: 100%;
  height: 100%;
  overflow: visible;
}
.axis-top {
  position: absolute;
  top: 2px;
  left: 0;
  z-index: 1;
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-tertiary);
  pointer-events: none;
}
.grid {
  stroke: var(--color-border);
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}
.line {
  fill: none;
  stroke-width: 2;
  stroke-linejoin: round;
  stroke-linecap: round;
  vector-effect: non-scaling-stroke;
}
.line-down { stroke: var(--series-down); }
.line-up { stroke: var(--series-up); }
.line-count { stroke: var(--series-count); }
.area { stroke: none; }
.area-down { fill: color-mix(in srgb, var(--series-down) 14%, transparent); }
.area-count { fill: color-mix(in srgb, var(--series-count) 14%, transparent); }
.crosshair {
  stroke: var(--color-text-tertiary);
  stroke-width: 1;
  vector-effect: non-scaling-stroke;
}
/* HTML, not SVG circles: a circle in a stretched viewBox is an ellipse. */
.dot {
  position: absolute;
  width: 8px;
  height: 8px;
  border: 2px solid var(--dot-ring, #ffffff);
  border-radius: 50%;
  transform: translate(-50%, -50%);
  pointer-events: none;
}
.dot-down { background: var(--series-down); }
.dot-up { background: var(--series-up); }
.dot-count { background: var(--series-count); }
@media (prefers-color-scheme: dark) {
  .dot { --dot-ring: #151c25; }
}
/*
 * The hover card. Opaque: it sits over the lines, and the app's translucent
 * glass would let them show through the numbers. `pointer-events: none` so it
 * can never come between the pointer and the plot it is describing.
 */
.tip {
  position: absolute;
  top: 6px;
  z-index: 2;
  display: flex;
  min-width: 132px;
  flex-direction: column;
  gap: 3px;
  border: 1px solid var(--color-border-dark);
  border-radius: 8px;
  background: #ffffff;
  padding: 6px 9px;
  box-shadow: 0 6px 18px rgba(15, 23, 42, 0.16);
  font-size: 12px;
  white-space: nowrap;
  color: var(--color-text-primary);
  transform: translateX(10px);
  pointer-events: none;
}
.tip.is-flipped {
  transform: translateX(calc(-100% - 10px));
}
.tip-when {
  font-size: 10px;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-secondary);
}
.tip-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.tip-label {
  color: var(--color-text-secondary);
}
.tip-row b {
  margin-left: auto;
  padding-left: 10px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
}
@media (prefers-color-scheme: dark) {
  .tip {
    background: #131a23;
  }
}
.time-axis {
  display: flex;
  justify-content: space-between;
  font-size: 10px;
  color: var(--color-text-tertiary);
}

.totals {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  margin: 10px 0 0;
  border-top: 1px solid var(--color-border);
  padding-top: 10px;
}
.totals dt {
  font-size: 10px;
  letter-spacing: 0.05em;
  text-transform: uppercase;
  color: var(--color-text-secondary);
}
.totals dd {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-primary);
}
</style>
