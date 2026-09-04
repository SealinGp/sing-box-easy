<script setup lang="ts">
/**
 * Overview card: the config's traffic flow, drawn.
 *
 * The Route page lists `route.rules` as rows, which is the right shape for
 * editing one and the wrong shape for answering "what does this config
 * actually do". Two facts are invisible in a list and obvious in a picture:
 * that many rules share one outbound, and that the list is an ordered ladder
 * whose first match wins. Both are what someone is checking right after an edit.
 *
 * It reads `GET /config` directly rather than the three narrower endpoints —
 * the diagram needs `inbounds`, `outbounds`, `endpoints`, `route.rules` and
 * `route.final` to be from the SAME read, or a node can appear missing simply
 * because two requests straddled a config write.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import {
  ArrowPathIcon,
  ArrowTopRightOnSquareIcon,
  ArrowsPointingInIcon,
  ArrowsPointingOutIcon,
  ChevronRightIcon,
  ExclamationTriangleIcon,
  SignalIcon,
  FunnelIcon,
  MinusIcon,
  PlusIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import RouteFlowDiagram from './RouteFlowDiagram.vue'
import { Select } from '../volt'
import { FILTER_THRESHOLD } from '../utils/selectFilter'
import { configService } from '../services'
import { useServiceStore } from '../stores/service'
import { useTrafficFlow } from '../composables/useTrafficFlow'
import { useDiagramZoom } from '../composables/useDiagramZoom'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { buildRouteTopology } from '../utils/routeTopology'
import { RATE_FLOOR, TOP_N, formatRate } from '../utils/flowOverlay'
import type { SingBoxConfig } from '../types/api'
import type { TrafficFilter } from '../types/trafficFlow'

const { t } = useI18n()

const ROUTE_PAGE = '/dashboard/route'

const config = ref<SingBoxConfig | null>(null)
/** First paint only — a manual refresh keeps the diagram on screen. */
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')

const topology = computed(() => buildRouteTopology(config.value))

const hasSomethingToDraw = computed(
  () => topology.value.rules.length > 0 || topology.value.inbounds.length > 0,
)

/**
 * Two diagnostics worth pulling out of the picture and into words.
 *
 * A missing outbound is only reported by sing-box at START, and an unreachable
 * rule is never reported at all — `sing-box check` passes on both. Someone
 * scanning the card should not have to spot a red node to learn about them.
 */
const unreachableCount = computed(
  () => topology.value.rules.filter((rule) => !rule.reachable).length,
)

const missingCount = computed(() => {
  const missing = new Set(
    topology.value.exits.filter((exit) => exit.kind === 'missing').map((exit) => exit.id),
  )
  return topology.value.rules.filter((rule) => rule.exitId && missing.has(rule.exitId)).length
})

const load = async (manual = false) => {
  if (manual) refreshing.value = true
  try {
    const response = await configService.getConfig()
    config.value = response.data ?? null
    error.value = ''
  } catch (err) {
    console.error('Failed to load config for the route topology card:', err)
    // Keep the last good drawing rather than blanking it — a failed re-read
    // says nothing about whether the config it drew is still correct.
    error.value = apiErrorMessage(err, t('routeFlow.loadFailed'))
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

onMounted(() => load())

/* ── Live overlay ─────────────────────────────────────────────────────────── */

/**
 * The toggle is gated on sing-box actually running. The stream would only
 * fail otherwise — the Clash API is served by sing-box itself — and a button
 * that can be clicked but never works reads as a bug in the panel.
 */
const serviceStore = useServiceStore()
const running = computed(() => serviceStore.status?.status === 'running')

/**
 * Live follows the service: on while sing-box is running, off when it is not.
 *
 * The card exists to answer "is my config doing what I meant", and the live
 * overlay is the half that answers it — asking for a click first made the
 * default view the one with less information in it. The gate is the same one
 * the button has: the stream is served by sing-box itself, so it is only ever
 * enabled while sing-box is up.
 *
 * `immediate` matters because the store may already know the status by the
 * time this card mounts (the Overview polls it elsewhere), in which case the
 * watcher would never fire and Live would stay off on a running host.
 */
const liveEnabled = ref(false)

watch(
  running,
  (isRunning) => {
    // A stop turns it off rather than leaving the composable retrying against
    // a controller that is not there; a start turns it back on, so the view
    // recovers by itself after the restart the panel triggers on every save.
    liveEnabled.value = isRunning
  },
  { immediate: true },
)

onMounted(() => serviceStore.startPolling())
onBeforeUnmount(() => serviceStore.stopPolling())

const toggleLive = () => {
  if (!running.value) return
  liveEnabled.value = !liveEnabled.value
}

/**
 * The host box is debounced into the stream's filter: each change is a new
 * server-side stream, and re-opening one per keystroke while someone types is
 * a burst of `/rules` reads for nothing.
 *
 * The source is NOT debounced. It is a pick from a list, not a keystroke
 * sequence, so there is no burst to absorb — and 400ms of nothing happening
 * after a deliberate click reads as a dropped click.
 */
const FILTER_DEBOUNCE_MS = 400
const sourceIpInput = ref('')
const hostInput = ref('')
const filter = ref<TrafficFilter>({ sourceIp: '', host: '' })
let filterTimer: ReturnType<typeof setTimeout> | null = null

watch(hostInput, (host) => {
  if (filterTimer !== null) clearTimeout(filterTimer)
  filterTimer = setTimeout(() => {
    filter.value = { ...filter.value, host }
  }, FILTER_DEBOUNCE_MS)
})

watch(sourceIpInput, (sourceIp) => {
  filter.value = { ...filter.value, sourceIp }
})

const hasFilter = computed(() => sourceIpInput.value.trim() !== '' || hostInput.value.trim() !== '')

const clearFilter = () => {
  sourceIpInput.value = ''
  hostInput.value = ''
}

const {
  overlay,
  frame,
  error: liveError,
  connecting,
} = useTrafficFlow(liveEnabled, filter, () => t('routeFlow.live.failed'))

/**
 * The devices currently holding connections — the source filter's list.
 *
 * Built from the frame rather than typed, which is the whole point: the panel
 * already knows every client address in the snapshot, and typing one from
 * memory is how a filter ends up silently matching nothing. The server
 * collects them BEFORE applying the filter, so picking one does not empty the
 * list it was picked from.
 *
 * The selection is re-appended when it drops out of the frame. A device whose
 * last connection just closed would otherwise vanish from the picker while its
 * filter was still in force, leaving an empty diagram and no visible cause.
 */
const sourceOptions = computed(() => {
  const sources = frame.value?.sources ?? []
  const options = sources.map((source) => ({
    value: source.ip,
    connections: source.connections,
    down: source.down,
  }))
  if (sourceIpInput.value && !sources.some((source) => source.ip === sourceIpInput.value)) {
    options.unshift({ value: sourceIpInput.value, connections: 0, down: 0 })
  }
  return [{ value: '', connections: 0, down: 0 }, ...options]
})

const sourceLabel = (value: string) => (value === '' ? t('routeFlow.live.allSources') : value)

const rate = (bytesPerSec: number) => t('routeFlow.live.rate', { rate: formatRate(bytesPerSec) })

/** The floor, spelled out in the legend — "quiet" is not a number. */
const floorLabel = computed(() => rate(RATE_FLOOR))

/** Connections whose rule string has no row — lit by exit only. */
const unmatchedConnections = computed(() =>
  (overlay.value?.unmatched ?? []).reduce((sum, flow) => sum + flow.connections, 0),
)

/* ── Zoom ─────────────────────────────────────────────────────────────────── */

/**
 * The diagram's own sizing shrinks a fixed 1290px canvas to whatever room the
 * card has, which is the right first look and the wrong one for reading a
 * single rule. See `useDiagramZoom` for why this scales-and-scrolls rather
 * than transforming a viewBox.
 */
const zoom = useDiagramZoom()

/* ── Busy only ────────────────────────────────────────────────────────────── */

/**
 * Fold rules carrying nothing into collapsed bands while Live is on.
 *
 * The floor in `flowOverlay` quiets the ladder; it does not SHORTEN it, and 28
 * rungs of which 6 matter still has to be scrolled. This is the part that
 * answers "show me what is actually moving".
 *
 * Off by default, and live-only, because it is the one thing in this card that
 * breaks the promise the diagram otherwise keeps — that the picture is the same
 * shape with the overlay on and off. That comparison is what the card is for,
 * so changing the shape has to be something the operator asks for.
 */
const busyOnly = ref(false)

/** Bands the operator has clicked open. Cleared whenever the fold is re-armed. */
const revealed = ref<Set<number>>(new Set())

const hiddenRuleIndices = computed<ReadonlySet<number> | undefined>(() => {
  if (!busyOnly.value || !liveEnabled.value || !overlay.value) return undefined
  const hidden = new Set<number>()
  for (const rule of topology.value.rules) {
    if (overlay.value.ribbons.get(`rule:${rule.index}`)?.lit) continue
    if (revealed.value.has(rule.index)) continue
    hidden.add(rule.index)
  }
  return hidden
})

const hiddenCount = computed(() => hiddenRuleIndices.value?.size ?? 0)

const expandBand = (indices: number[]) => {
  revealed.value = new Set([...revealed.value, ...indices])
}

// Re-arming the fold, or losing the data it is computed from, throws the
// manual reveals away — otherwise yesterday's clicks quietly keep rules on
// screen that today's traffic says are idle.
watch([busyOnly, liveEnabled], () => {
  revealed.value = new Set()
})

/* ── Legend ───────────────────────────────────────────────────────────────── */

const LEGEND_KEY = 'sbe-routeflow-legend'

/**
 * Whether the symbol key is showing.
 *
 * Open on a fresh install and closed forever after one click, rather than the
 * other way round: the symbols are genuinely needed for the first few visits,
 * and a key nobody can find is the same as no key. Once shelved it stays
 * shelved across sessions — the whole complaint about a permanent footer is
 * that it outlives its usefulness, and re-opening on every visit reproduces it.
 *
 * Read at setup, before the first paint, so the footer does not appear and then
 * disappear. Storage can throw outright (Safari private browsing) and can hold
 * anything, so both are handled — either way the default is to teach.
 */
const readLegendOpen = (): boolean => {
  try {
    return localStorage.getItem(LEGEND_KEY) !== '0'
  } catch {
    return true
  }
}

const legendOpen = ref(readLegendOpen())

const toggleLegend = () => {
  legendOpen.value = !legendOpen.value
  try {
    localStorage.setItem(LEGEND_KEY, legendOpen.value ? '1' : '0')
  } catch (err) {
    // Non-fatal: the choice still holds for this session.
    console.warn('Could not persist the traffic-flow legend state:', err)
  }
}

/* ── Full-window mode ─────────────────────────────────────────────────────── */

/**
 * The card's body moves to a viewport-filling overlay, and back.
 *
 * It MOVES — one `<Teleport :disabled>` re-parents the same subtree — rather
 * than rendering a second copy, so the live stream, its overlay, the filter
 * text and the hover state all survive the switch. A dialog component would
 * mount fresh content and reconnect the stream for nothing.
 *
 * In-window rather than the Fullscreen API: an operator comparing this against
 * the Clash dashboard or a terminal wants the browser chrome and the other
 * windows exactly where they were.
 */
const expanded = ref(false)

const toggleExpanded = () => {
  expanded.value = !expanded.value
}

const onKeydown = (event: KeyboardEvent) => {
  if (event.key === 'Escape' && expanded.value) expanded.value = false
}

watch(expanded, (on) => {
  // The page behind must not scroll under the overlay; the diagram's own
  // wrapper is what scrolls if the ladder is taller than the window.
  document.body.style.overflow = on ? 'hidden' : ''
  if (on) window.addEventListener('keydown', onKeydown)
  else window.removeEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  document.body.style.overflow = ''
})
</script>

<template>
  <div class="bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <!-- Where the body went while it is full-window. -->
    <div v-if="expanded" class="flex items-center justify-between gap-3">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">{{ $t('routeFlow.title') }}</h3>
      <button
        type="button"
        class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
        @click="toggleExpanded"
      >
        <ArrowsPointingInIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.fullWindow.restore') }}
      </button>
    </div>

    <!--
      `display: contents` while docked, so the wrapper adds no box to the
      card; a fixed, viewport-filling column while expanded. Same children in
      both cases — see `expanded` in the script for why it moves rather than
      re-renders.

      z-40, one step BELOW the app's overlay layer: this container teleports to
      <body>, and so do the select panels inside it. At an equal z-index the
      later sibling wins, which put the diagram over an open dropdown. Above
      the page (the top bar is z-30), under anything that overlays it.
    -->
    <Teleport to="body" :disabled="!expanded">
      <div
        :class="
          expanded
            ? 'fixed inset-0 z-40 flex flex-col bg-white dark:bg-slate-900 p-4 sm:p-6 overflow-hidden'
            : 'contents'
        "
        :role="expanded ? 'dialog' : undefined"
        :aria-modal="expanded ? 'true' : undefined"
        :aria-label="expanded ? $t('routeFlow.title') : undefined"
      >
    <div class="flex items-start justify-between gap-3 mb-1">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">
        {{ $t('routeFlow.title') }}
      </h3>
      <div class="flex items-center gap-3 flex-shrink-0">
        <!--
          Live: light the expected drawing with real connections. Disabled
          while sing-box is stopped — the data comes from sing-box's own API,
          so there is nothing to show and the click would only fail.
        -->
        <button
          type="button"
          role="switch"
          :aria-checked="liveEnabled"
          class="inline-flex items-center gap-1.5 text-xs font-medium px-2 py-1 rounded-md border transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          :class="
            liveEnabled
              ? 'bg-primary-600 border-primary-600 text-white hover:bg-primary-700'
              : 'border-gray-300 dark:border-slate-600 text-gray-600 dark:text-gray-300 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-400'
          "
          :disabled="!running"
          :title="!running ? $t('routeFlow.live.needsRunning') : liveEnabled ? $t('routeFlow.live.on') : $t('routeFlow.live.off')"
          @click="toggleLive"
        >
          <span
            class="h-1.5 w-1.5 rounded-full"
            :class="liveEnabled ? 'bg-emerald-500' : 'bg-gray-400'"
          ></span>
          <SignalIcon class="h-3.5 w-3.5" />
          {{ $t('routeFlow.live.toggle') }}
        </button>
        <!--
          View chrome, grouped tight. Three clusters at gap-3 — the Live mode
          toggle, these two icon buttons, then the link away — is what stops the
          header reading as one undifferentiated row of six controls. Zoom used
          to sit here and now floats on the diagram, which is both where it acts
          and one fewer thing competing at this level.
        -->
        <div class="inline-flex items-center gap-1">
          <button
            type="button"
            class="inline-flex items-center p-1 rounded text-gray-500 hover:text-gray-700 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-slate-700 transition-colors disabled:opacity-50"
            :disabled="refreshing"
            :title="$t('routeFlow.refresh')"
            :aria-label="$t('routeFlow.refresh')"
            @click="load(true)"
          >
            <ArrowPathIcon class="h-4 w-4" :class="refreshing ? 'animate-spin' : ''" />
          </button>
          <button
            type="button"
            class="inline-flex items-center p-1 rounded text-gray-500 hover:text-gray-700 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-slate-700 transition-colors"
            :title="expanded ? $t('routeFlow.fullWindow.exit') : $t('routeFlow.fullWindow.enter')"
            :aria-label="expanded ? $t('routeFlow.fullWindow.exit') : $t('routeFlow.fullWindow.enter')"
            :aria-pressed="expanded"
            @click="toggleExpanded"
          >
            <ArrowsPointingInIcon v-if="expanded" class="h-4 w-4" />
            <ArrowsPointingOutIcon v-else class="h-4 w-4" />
          </button>
        </div>
        <RouterLink
          :to="ROUTE_PAGE"
          class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
        >
          {{ $t('routeFlow.openRoute') }}
          <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
        </RouterLink>
      </div>
    </div>

    <p v-if="error" class="text-xs text-red-600 dark:text-red-400 mb-3">{{ error }}</p>

    <!--
      Live strip: what is flowing right now, and the three ways to narrow it.

      NO TINT. A full-width tinted bar sitting above content reads as a HEADER —
      a label for the diagram — when this row is the opposite of a label: values
      that change every second, plus the controls that shape them. Liveness is
      already said twice above (the Live toggle's own state, and the diagram
      lighting up), and spending a whole surface saying it a third time is what
      made the row look like chrome.

      Two kinds of thing, separated by WEIGHT rather than by a box. The readouts
      are data — a micro-label in the app's usual uppercase idiom, then a number
      with enough presence to be read at a glance while it ticks. The filters
      are controls and carry their own borders, which only read as borders
      against the card: behind a tint, a bordered input and its background
      flatten into each other.
    -->
    <div v-if="liveEnabled" class="flex flex-wrap items-center gap-x-5 gap-y-2 mb-3 text-xs">
      <template v-if="overlay">
        <!--
          Down carries the accent because it is the number the whole overlay is
          built on: ribbon width, pulse speed and the top-N ranking are all
          functions of it. Up is the same size but neutral — equal billing would
          claim the two drive the picture equally, and they do not.
        -->
        <span class="inline-flex items-baseline gap-1.5">
          <span class="font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ $t('routeFlow.live.down') }}
          </span>
          <span class="text-sm font-semibold tabular-nums text-primary-700 dark:text-primary-300">
            {{ rate(overlay.totals.down) }}
          </span>
        </span>
        <span class="inline-flex items-baseline gap-1.5">
          <span class="font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ $t('routeFlow.live.up') }}
          </span>
          <span class="text-sm font-semibold tabular-nums text-gray-700 dark:text-gray-200">
            {{ rate(overlay.totals.up) }}
          </span>
        </span>

        <!--
          Counts and caveats stay secondary: they qualify the two rates rather
          than competing with them, and at equal weight the eye is given five
          numbers to rank instead of two.
        -->
        <span class="tabular-nums text-gray-500 dark:text-gray-400">
          {{ $t('routeFlow.live.connections', { n: overlay.totals.connections }, overlay.totals.connections) }}
          <span v-if="overlay.totals.connections !== overlay.totals.all" class="text-gray-400 dark:text-gray-500">
            ({{ $t('routeFlow.live.shownOf', { shown: overlay.totals.connections, all: overlay.totals.all }) }})
          </span>
        </span>
        <span
          v-if="overlay.totals.connections > 0 && overlay.totals.down === 0"
          class="text-gray-400 dark:text-gray-500 italic"
        >
          {{ $t('routeFlow.live.idle') }}
        </span>
        <!--
          How many rows the floor is holding back. A diagram that went quiet has
          to say why, or a working stream reads as a broken one.
        -->
        <span v-if="overlay.belowFloor > 0" class="text-gray-400 dark:text-gray-500">
          {{ $t('routeFlow.live.belowFloor', { n: overlay.belowFloor }, overlay.belowFloor) }}
        </span>
      </template>
      <span v-else-if="connecting" class="text-gray-500 dark:text-gray-400 inline-flex items-center gap-1">
        <ArrowPathIcon class="h-3.5 w-3.5 animate-spin" />
        {{ $t('routeFlow.live.connecting') }}
      </span>
      <span v-if="liveError" class="text-red-600 dark:text-red-400 inline-flex items-center gap-1">
        <ExclamationTriangleIcon class="h-3.5 w-3.5" />
        {{ liveError }} · {{ $t('routeFlow.live.retrying') }}
      </span>

      <span class="flex-1"></span>

      <!--
        Busy only: fold the rules carrying nothing into bands, so a 28-rung
        ladder shortens to what is moving. Off by default — it is the one
        control here that changes the diagram's SHAPE, and the shape being
        identical with the overlay on and off is what makes expected and actual
        comparable.
      -->
      <button
        type="button"
        role="switch"
        :aria-checked="busyOnly"
        class="inline-flex items-center gap-1 px-2 py-1 rounded-md border text-xs font-medium transition-colors"
        :class="
          busyOnly
            ? 'bg-primary-600 border-primary-600 text-white hover:bg-primary-700'
            : 'border-gray-300 dark:border-slate-600 text-gray-600 dark:text-gray-300 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-400'
        "
        :title="$t('routeFlow.live.busyOnlyHint')"
        @click="busyOnly = !busyOnly"
      >
        <FunnelIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.live.busyOnly') }}
        <span v-if="busyOnly && hiddenCount > 0" class="tabular-nums opacity-80">−{{ hiddenCount }}</span>
      </button>

      <!--
        Narrowing to one device or one site is how a slow-site complaint gets
        pinned. The device is PICKED, not typed: the frame already carries every
        client address in the snapshot, so a list beats recalling an IP — and a
        typo in a typed one is indistinguishable from a device with no traffic.
        A value picker, so `volt/Select`; the `value: ''` entry needs its label
        repeated as the placeholder (see DESIGN.md §6).

        The panel teleports to <body> like every other overlay in the app — see
        the full-window container's z-index for why that keeps working there,
        and why keeping the panel inside the card does not: `backdrop-filter`
        samples only what is painted BEHIND an element, so a panel inside the
        card had the diagram show straight through its glass.
      -->
      <label class="inline-flex items-center gap-1">
        <span class="sr-only">{{ $t('routeFlow.live.filterSource') }}</span>
        <Select
          v-model="sourceIpInput"
          :options="sourceOptions"
          optionValue="value"
          :filter="sourceOptions.length >= FILTER_THRESHOLD"
          :filterPlaceholder="$t('common.search')"
          :emptyFilterMessage="$t('common.noMatch')"
          :emptyMessage="$t('routeFlow.live.noSources')"
          :placeholder="$t('routeFlow.live.allSources')"
          :title="$t('routeFlow.live.filterSource')"
          class="w-56 text-xs"
          scrollHeight="14rem"
        >
          <template #value="{ value }">
            <span class="truncate">{{ sourceLabel(value ?? '') }}</span>
          </template>
          <!--
            The address never truncates; the busy-ness does. An address clipped
            to "192.168.31.1…" is unusable — it is the whole identity of the
            row — while a clipped rate still reads as "this one is busy".
          -->
          <template #option="{ option }">
            <span class="shrink-0">{{ sourceLabel(option.value) }}</span>
            <span v-if="option.connections" class="ml-auto min-w-0 truncate pl-2 tabular-nums text-[10px] text-gray-500 dark:text-gray-400">
              {{ $t('routeFlow.live.sourceConnections', { n: option.connections }) }}
              <template v-if="option.down">· {{ rate(option.down) }}</template>
            </span>
          </template>
        </Select>
      </label>
      <!-- Sized to match the source picker beside it, which is a control-height field. -->
      <label class="inline-flex items-center gap-1">
        <span class="sr-only">{{ $t('routeFlow.live.filterHost') }}</span>
        <input
          v-model="hostInput"
          type="text"
          :placeholder="$t('routeFlow.live.filterHost')"
          class="w-40 px-3 py-2 rounded-control border border-gray-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-xs text-gray-800 dark:text-gray-100 placeholder:text-gray-400 focus:outline-none focus:border-primary-500"
        />
      </label>
      <button
        v-if="hasFilter"
        type="button"
        class="inline-flex items-center gap-0.5 text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
        :title="$t('routeFlow.live.clearFilter')"
        @click="clearFilter"
      >
        <XMarkIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.live.clearFilter') }}
      </button>
    </div>

    <!-- Traffic the running rule list cannot place on a row. -->
    <p
      v-if="overlay && overlay.unmatched.length > 0"
      class="text-xs text-amber-700 dark:text-amber-300 mb-3 inline-flex items-center gap-1"
    >
      <ExclamationTriangleIcon class="h-3.5 w-3.5" />
      {{ $t('routeFlow.live.unmatched', { n: unmatchedConnections }, unmatchedConnections) }}
    </p>

    <!-- Config-level problems `sing-box check` does not report. -->
    <div v-if="!loading && (unreachableCount || missingCount)" class="flex flex-wrap gap-2 mb-3">
      <span
        v-if="missingCount"
        class="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300"
      >
        <ExclamationTriangleIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.warnMissing', { n: missingCount }, missingCount) }}
      </span>
      <span
        v-if="unreachableCount"
        class="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md bg-amber-50 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300"
      >
        <ExclamationTriangleIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.warnUnreachable', { n: unreachableCount }, unreachableCount) }}
      </span>
    </div>

    <div v-if="loading" class="h-40 flex items-center justify-center">
      <ArrowPathIcon class="h-5 w-5 animate-spin text-gray-400" />
    </div>

    <p v-else-if="!hasSomethingToDraw" class="text-sm text-gray-500 dark:text-gray-400 py-6 text-center">
      {{ config ? $t('routeFlow.empty') : $t('routeFlow.noConfig') }}
    </p>

    <template v-else>
      <!--
        Docked, a tall ladder scrolls rather than shrinking: this repo's own
        config has 28 rules, and scaling that into a card makes every label
        unreadable. Full-window, the wrapper takes the remaining height and the
        diagram fits it — the whole ladder at once is the point of the mode.
      -->
      <div :class="expanded ? 'relative flex-1 min-h-0' : 'relative'">
        <div
          :ref="zoom.bindViewport"
          tabindex="0"
          :class="[
            expanded ? 'h-full overflow-auto -mx-1 px-1' : 'overflow-auto max-h-[34rem] -mx-1 px-1',
            'focus:outline-none focus-visible:ring-1 focus-visible:ring-primary-500 rounded',
            // The gesture is otherwise invisible: the cursor is the only thing
            // that says the held modifier turned the diagram into a canvas.
            zoom.panning.value ? 'cursor-grabbing' : zoom.panReady.value ? 'cursor-grab' : '',
          ]"
        >
          <RouteFlowDiagram
            :topology="topology"
            :live="liveEnabled ? overlay : null"
            :fit="expanded"
            :scale="zoom.scale.value"
            :hidden-rule-indices="hiddenRuleIndices"
            @expand-band="expandBand"
          />
        </div>

        <!--
          Zoom floats ON the diagram, the way every map and design tool puts it,
          rather than in the card header. Two reasons beyond decluttering: it is
          adjacent to the thing it scales, and it stays put while the content
          under it scrolls, so it is reachable at any pan position.

          Absolutely positioned against this wrapper, NOT inside the scrolling
          child — inside, it would scroll away with the diagram and be gone the
          moment it was needed. Nudged clear of the corner so it does not sit on
          top of both scrollbars at once.
        -->
        <div
          class="absolute bottom-3 right-3 inline-flex items-center rounded-md border border-gray-200 dark:border-slate-600 bg-white/90 dark:bg-slate-800/90 backdrop-blur-sm shadow-surface overflow-hidden"
          :title="$t('routeFlow.zoom.hint')"
        >
          <button
            type="button"
            class="px-1.5 py-1 text-gray-500 hover:text-gray-800 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-100 dark:hover:bg-slate-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            :disabled="!zoom.canZoomOut.value"
            :title="$t('routeFlow.zoom.out')"
            :aria-label="$t('routeFlow.zoom.out')"
            @click="zoom.zoomOut()"
          >
            <MinusIcon class="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            class="px-1.5 py-1 text-xs tabular-nums text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-slate-700 transition-colors min-w-[3rem]"
            :title="$t('routeFlow.zoom.reset')"
            @click="zoom.reset()"
          >
            {{ zoom.percent.value === null ? $t('routeFlow.zoom.fit') : `${zoom.percent.value}%` }}
          </button>
          <button
            type="button"
            class="px-1.5 py-1 text-gray-500 hover:text-gray-800 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-100 dark:hover:bg-slate-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            :disabled="!zoom.canZoomIn.value"
            :title="$t('routeFlow.zoom.in')"
            :aria-label="$t('routeFlow.zoom.in')"
            @click="zoom.zoomIn()"
          >
            <PlusIcon class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      <!-- Names the three columns, and says what the middle one's order means. -->
      <div
        class="flex items-center justify-between gap-2 mt-2 pt-2 text-[11px] text-gray-500 dark:text-gray-400 border-t border-gray-100 dark:border-slate-700"
      >
        <span>{{ $t('routeFlow.columns.inbounds') }}</span>
        <span class="text-center">{{ $t('routeFlow.columns.rules') }}</span>
        <span>{{ $t('routeFlow.columns.outbounds') }}</span>
      </div>

      <!--
        The key, on demand.

        This is reference material: a new install needs it two or three times,
        and after that it is a permanent five-item footer explaining symbols the
        operator has long since learned. So it behaves like a manual — shelved
        by default once shelved, one muted line away when a symbol IS unfamiliar
        — and the choice persists, because re-collapsing it on every visit is
        the same nuisance in slower motion.

        The column strip above is deliberately NOT in here: it carries the claim
        that the ladder is ordered and stops at the first match, which is the
        diagram's whole thesis rather than a symbol key.
      -->
      <div class="flex items-center justify-between gap-2 mt-2">
        <button
          type="button"
          class="inline-flex items-center gap-1 text-[11px] text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 transition-colors"
          :aria-expanded="legendOpen"
          @click="toggleLegend"
        >
          <ChevronRightIcon class="h-3 w-3 transition-transform" :class="legendOpen ? 'rotate-90' : ''" />
          {{ $t('routeFlow.legend.title') }}
        </button>
        <span v-if="expanded" class="text-[11px] text-gray-400">{{ $t('routeFlow.fullWindow.escHint') }}</span>
      </div>

      <!-- What the non-obvious strokes mean. -->
      <div v-if="legendOpen" class="flex flex-wrap gap-x-4 gap-y-1 mt-2 text-[11px] text-gray-500 dark:text-gray-400">
        <span class="inline-flex items-center gap-1.5">
          <svg width="18" height="8" aria-hidden="true">
            <line x1="0" y1="4" x2="18" y2="4" stroke-width="1.5" stroke-dasharray="4 3" class="stroke-gray-400 dark:stroke-slate-500" />
          </svg>
          {{ $t('routeFlow.legend.passthrough') }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <span class="h-2 w-2 rounded-full bg-amber-400"></span>
          {{ $t('routeFlow.legend.scoped') }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <span
            class="px-1 rounded-full bg-primary-100 dark:bg-primary-900/60 text-primary-700 dark:text-primary-300 font-semibold tabular-nums"
            >3</span
          >
          {{ $t('routeFlow.legend.converge', { n: 3 }) }}
        </span>
        <template v-if="liveEnabled">
          <span class="inline-flex items-center gap-1.5">
            <!-- A lit ribbon with one pulse on it — what the top N look like. -->
            <svg width="22" height="8" aria-hidden="true">
              <line x1="1" y1="4" x2="21" y2="4" stroke-width="2" stroke-linecap="round" class="stroke-primary-400 dark:stroke-primary-600 opacity-80" />
              <line x1="8" y1="4" x2="14" y2="4" stroke-width="2.5" stroke-linecap="round" class="stroke-primary-600 dark:stroke-primary-300" />
            </svg>
            {{ $t('routeFlow.live.legendMoving', { n: TOP_N }) }}
          </span>
          <span class="inline-flex items-center gap-1.5">
            <svg width="22" height="8" aria-hidden="true">
              <line x1="0" y1="4" x2="22" y2="4" stroke-width="2.5" class="stroke-primary-400 dark:stroke-primary-600" />
            </svg>
            {{ $t('routeFlow.live.legendLit', { rate: floorLabel }) }}
          </span>
        </template>
        <span class="ml-auto text-gray-400">{{ $t('routeFlow.zoom.hint') }}</span>
      </div>
    </template>
      </div>
    </Teleport>
  </div>
</template>
