import { useWorkspaceActive } from './workspaceContext'
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { configService } from '../services'
import { useServiceStore } from '../stores/service'
import { useTrafficFlow } from './useTrafficFlow'
import { useDiagramZoom } from './useDiagramZoom'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { buildRouteTopology } from '../utils/routeTopology'
import { HEAT_STEPS, RATE_FLOOR, formatRate, heatFor } from '../utils/flowOverlay'
import { HEAT_SWATCH, HEAT_TEXT, heatClass } from '../utils/flowHeat'
import type { SingBoxConfig } from '../types/api'
import type { TrafficFilter } from '../types/trafficFlow'

export function useRouteTopologyCard() {
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

/**
 * The total's colour follows the same scale as the diagram, so the strip's
 * one accented number and the ribbons below it never disagree about whether
 * "a lot" is happening.
 */
const totalHeat = computed(() =>
  overlay.value ? heatFor(overlay.value.totals.down, overlay.value.totals.up) : 0,
)
const totalDownClass = computed(() =>
  totalHeat.value === 0 ? 'text-primary-700 dark:text-primary-300' : heatClass(HEAT_TEXT, totalHeat.value),
)

/** The legend's colour scale: tier 1 starts at the floor, then each step. */
const heatScale = computed(() =>
  [RATE_FLOOR, ...HEAT_STEPS].map((threshold, i) => ({
    threshold: formatRate(threshold),
    swatch: heatClass(HEAT_SWATCH, (i + 1) as 1 | 2 | 3 | 4),
  })),
)

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
const workspaceActive = useWorkspaceActive()
watch(workspaceActive, active => { if (!active) expanded.value = false })

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

return {
  ROUTE_PAGE,
  config,
  loading,
  refreshing,
  error,
  topology,
  hasSomethingToDraw,
  unreachableCount,
  missingCount,
  load,
  running,
  liveEnabled,
  toggleLive,
  sourceIpInput,
  hostInput,
  hasFilter,
  clearFilter,
  sourceOptions,
  sourceLabel,
  rate,
  floorLabel,
  totalDownClass,
  heatScale,
  unmatchedConnections,
  zoom,
  busyOnly,
  hiddenRuleIndices,
  hiddenCount,
  expandBand,
  legendOpen,
  toggleLegend,
  expanded,
  toggleExpanded,
  overlay,
  liveError,
  connecting
}
}
