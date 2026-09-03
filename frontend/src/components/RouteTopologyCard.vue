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
  ExclamationTriangleIcon,
  SignalIcon,
  XMarkIcon,
} from '@heroicons/vue/24/outline'
import RouteFlowDiagram from './RouteFlowDiagram.vue'
import { configService } from '../services'
import { useServiceStore } from '../stores/service'
import { useTrafficFlow } from '../composables/useTrafficFlow'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { buildRouteTopology } from '../utils/routeTopology'
import { TOP_N, formatRate } from '../utils/flowOverlay'
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

const liveEnabled = ref(false)

// A stop while Live is on turns it off rather than leaving the composable
// retrying against a controller that is not there.
watch(running, (isRunning) => {
  if (!isRunning) liveEnabled.value = false
})

onMounted(() => serviceStore.startPolling())
onBeforeUnmount(() => serviceStore.stopPolling())

const toggleLive = () => {
  if (!running.value) return
  liveEnabled.value = !liveEnabled.value
}

/**
 * Filter inputs are debounced into the stream's filter: each change is a new
 * server-side stream, and re-opening one per keystroke while someone types an
 * IP is a burst of `/rules` reads for nothing.
 */
const FILTER_DEBOUNCE_MS = 400
const sourceIpInput = ref('')
const hostInput = ref('')
const filter = ref<TrafficFilter>({ sourceIp: '', host: '' })
let filterTimer: ReturnType<typeof setTimeout> | null = null

watch([sourceIpInput, hostInput], ([sourceIp, host]) => {
  if (filterTimer !== null) clearTimeout(filterTimer)
  filterTimer = setTimeout(() => {
    filter.value = { sourceIp, host }
  }, FILTER_DEBOUNCE_MS)
})

const hasFilter = computed(() => sourceIpInput.value.trim() !== '' || hostInput.value.trim() !== '')

const clearFilter = () => {
  sourceIpInput.value = ''
  hostInput.value = ''
}

const {
  overlay,
  error: liveError,
  connecting,
} = useTrafficFlow(liveEnabled, filter, () => t('routeFlow.live.failed'))

const rate = (bytesPerSec: number) => t('routeFlow.live.rate', { rate: formatRate(bytesPerSec) })

/** Connections whose rule string has no row — lit by exit only. */
const unmatchedConnections = computed(() =>
  (overlay.value?.unmatched ?? []).reduce((sum, flow) => sum + flow.connections, 0),
)

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
    -->
    <Teleport to="body" :disabled="!expanded">
      <div
        :class="
          expanded
            ? 'fixed inset-0 z-50 flex flex-col bg-white dark:bg-slate-900 p-4 sm:p-6 overflow-hidden'
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
            :class="liveEnabled ? 'bg-white animate-pulse' : running ? 'bg-emerald-500' : 'bg-gray-400'"
          ></span>
          <SignalIcon class="h-3.5 w-3.5" />
          {{ $t('routeFlow.live.toggle') }}
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1 text-xs font-medium text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors disabled:opacity-50"
          :disabled="refreshing"
          :title="$t('routeFlow.refresh')"
          :aria-label="$t('routeFlow.refresh')"
          @click="load(true)"
        >
          <ArrowPathIcon class="h-4 w-4" :class="refreshing ? 'animate-spin' : ''" />
        </button>
        <button
          type="button"
          class="inline-flex items-center gap-1 text-xs font-medium text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors"
          :title="expanded ? $t('routeFlow.fullWindow.exit') : $t('routeFlow.fullWindow.enter')"
          :aria-label="expanded ? $t('routeFlow.fullWindow.exit') : $t('routeFlow.fullWindow.enter')"
          :aria-pressed="expanded"
          @click="toggleExpanded"
        >
          <ArrowsPointingInIcon v-if="expanded" class="h-4 w-4" />
          <ArrowsPointingOutIcon v-else class="h-4 w-4" />
        </button>
        <RouterLink
          :to="ROUTE_PAGE"
          class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
        >
          {{ $t('routeFlow.openRoute') }}
          <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
        </RouterLink>
      </div>
    </div>

    <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
      {{ $t('routeFlow.desc') }}
    </p>

    <p v-if="error" class="text-xs text-red-600 dark:text-red-400 mb-3">{{ error }}</p>

    <!-- Live strip: what is flowing right now, and the two ways to narrow it. -->
    <div
      v-if="liveEnabled"
      class="flex flex-wrap items-center gap-x-4 gap-y-2 mb-3 px-3 py-2 rounded-md bg-primary-50/60 dark:bg-primary-950/30 border border-primary-100 dark:border-primary-900/60 text-xs"
    >
      <template v-if="overlay">
        <span class="tabular-nums text-gray-800 dark:text-gray-100">
          <span class="text-gray-500 dark:text-gray-400">{{ $t('routeFlow.live.down') }}</span>
          <strong class="ml-1">{{ rate(overlay.totals.down) }}</strong>
        </span>
        <span class="tabular-nums text-gray-800 dark:text-gray-100">
          <span class="text-gray-500 dark:text-gray-400">{{ $t('routeFlow.live.up') }}</span>
          <strong class="ml-1">{{ rate(overlay.totals.up) }}</strong>
        </span>
        <span class="tabular-nums text-gray-600 dark:text-gray-300">
          {{ $t('routeFlow.live.connections', { n: overlay.totals.connections }, overlay.totals.connections) }}
          <span v-if="overlay.totals.connections !== overlay.totals.all" class="text-gray-400">
            ({{ $t('routeFlow.live.shownOf', { shown: overlay.totals.connections, all: overlay.totals.all }) }})
          </span>
        </span>
        <span v-if="overlay.totals.connections > 0 && overlay.totals.down === 0" class="text-gray-400 italic">
          {{ $t('routeFlow.live.idle') }}
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

      <!-- Narrowing to one device or one site is how a slow-site complaint gets pinned. -->
      <label class="inline-flex items-center gap-1">
        <input
          v-model="sourceIpInput"
          type="text"
          inputmode="decimal"
          :placeholder="$t('routeFlow.live.filterSource')"
          class="w-32 px-2 py-1 rounded border border-gray-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-xs text-gray-800 dark:text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-1 focus:ring-primary-500"
        />
      </label>
      <label class="inline-flex items-center gap-1">
        <input
          v-model="hostInput"
          type="text"
          :placeholder="$t('routeFlow.live.filterHost')"
          class="w-40 px-2 py-1 rounded border border-gray-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-xs text-gray-800 dark:text-gray-100 placeholder:text-gray-400 focus:outline-none focus:ring-1 focus:ring-primary-500"
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
      <div
        :class="expanded ? 'flex-1 min-h-0 overflow-auto -mx-1 px-1' : 'overflow-auto max-h-[34rem] -mx-1 px-1'"
      >
        <RouteFlowDiagram :topology="topology" :live="liveEnabled ? overlay : null" :fit="expanded" />
      </div>

      <!-- Names the three columns, and says what the middle one's order means. -->
      <div
        class="flex items-center justify-between gap-2 mt-2 pt-2 text-[11px] text-gray-500 dark:text-gray-400 border-t border-gray-100 dark:border-slate-700"
      >
        <span>{{ $t('routeFlow.columns.inbounds') }}</span>
        <span class="text-center">{{ $t('routeFlow.columns.rules') }}</span>
        <span>{{ $t('routeFlow.columns.outbounds') }}</span>
      </div>

      <!-- What the non-obvious strokes mean. -->
      <div class="flex flex-wrap gap-x-4 gap-y-1 mt-2 text-[11px] text-gray-500 dark:text-gray-400">
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
            <svg width="22" height="8" aria-hidden="true">
              <line x1="0" y1="4" x2="22" y2="4" stroke-width="2" stroke-dasharray="5 4" class="stroke-primary-600 dark:stroke-primary-300" />
            </svg>
            {{ $t('routeFlow.live.legendMoving', { n: TOP_N }) }}
          </span>
          <span class="inline-flex items-center gap-1.5">
            <svg width="22" height="8" aria-hidden="true">
              <line x1="0" y1="4" x2="22" y2="4" stroke-width="2.5" class="stroke-primary-400 dark:stroke-primary-600" />
            </svg>
            {{ $t('routeFlow.live.legendLit') }}
          </span>
        </template>
        <span v-if="expanded" class="ml-auto text-gray-400">{{ $t('routeFlow.fullWindow.escHint') }}</span>
      </div>
    </template>
      </div>
    </Teleport>
  </div>
</template>
