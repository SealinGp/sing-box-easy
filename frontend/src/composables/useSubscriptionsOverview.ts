import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { subProbeService, subscriptionService } from '../services'
import { useNotify } from '../composables/useNotify'
import { summarizePlan, type PlanSummary } from '../utils/subscriptionInfo'
import { safeExternalUrl } from '../utils/safeExternalUrl'
import { summarizeUpdate, updateSubscriptionsSequentially } from '../utils/subscriptionUpdate'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { subscriptionHealth, type SubscriptionHealth } from '../utils/subscriptionHealth'
import { formatRelativeTime } from '../utils/relativeTime'
import type { Subscription } from '../types/api'
import type { ProbePoint } from '../types/subprobe'

export function useSubscriptionsOverview() {
  const { t, locale } = useI18n()
  const notify = useNotify()

  const subscriptions = ref<Subscription[]>([])
  /** First paint only. A manual refresh deliberately does NOT set this. */
  const loading = ref(true)
  /** A re-read is in flight — keeps the rows on screen rather than blanking. */
  const refreshing = ref(false)
  /**
   * A provider fetch is in flight — bulk or single.
   *
   * One flag for both on purpose: every refresh ends in a config.json write, and
   * two of them overlapping is how one update's outbounds get written over the
   * other's. It also keeps the UI honest — no second run can start while one is
   * still walking the list.
   */
  const updating = ref(false)
  /** Set when the fetch failed, so the card can say so instead of showing "0". */
  const failed = ref(false)
  /** Whether any fetch has ever succeeded. Guards the failure fallback below. */
  const loaded = ref(false)

  const SUBSCRIPTIONS_ROUTE = '/dashboard/outbounds/subscriptions'

  /**
   * @param manual `true` when the read is part of an operator-triggered action.
   *
   * The two modes differ in what they do to the card while the request is in
   * flight. The initial load has nothing to show, so it takes over the card with
   * a spinner. A manual re-read already has rows on screen — blanking them would
   * make the card flash and lose the numbers the operator is reading, so it
   * leaves them up.
   */
  async function load(manual = false) {
    if (manual) refreshing.value = true
    else loading.value = true

    try {
      const response = await subscriptionService.getSubscriptions()
      subscriptions.value = response.data.subscriptions || []
      loaded.value = true
      // Clear a stale failure so a recovered fetch stops showing the error.
      failed.value = false
    } catch (err) {
      // A failed refresh must not discard rows that are already on screen: stale
      // quota figures beat an error message that replaces them. Only a load that
      // has never succeeded falls back to the error state. `loaded` is tracked
      // rather than testing `subscriptions.length`, which cannot tell "fetch
      // failed" apart from "fetched fine, you have none".
      if (!loaded.value) failed.value = true
      notify.apiError(err, t('overview.subscriptions.loadFailed'))
    } finally {
      loading.value = false
      refreshing.value = false
    }
  }

  /**
   * Newest quality sample per subscription.
   *
   * Read alongside the list, not instead of it: quality is one line on a card
   * whose subject is quota and expiry, so a prober that is off or unreachable
   * must cost this card nothing — the line simply does not render.
   */
  const probeLatest = ref<Record<string, ProbePoint>>({})

  async function loadProbe() {
    try {
      const { data } = await subProbeService.getStatus()
      probeLatest.value = data.latest ?? {}
    } catch {
      // Deliberately silent. The Subscriptions page reports probe failures; a
      // toast here would fire on every dashboard visit of a deployment that has
      // no clash_api configured, which is a supported way to run the panel.
      probeLatest.value = {}
    }
  }

  onMounted(() => {
    void load()
    void loadProbe()
  })

  /**
   * Per-row outcome of the last "update all", keyed by subscription ID.
   *
   * This replaces the summary toast that used to end the run. A toast reports in
   * the corner what happened somewhere else, so with several subscriptions it
   * could only say "2 failed: A, C" and leave the reader to map names back onto
   * rows. Reporting on the row itself removes that step: the row that failed says
   * so, and the row that gained three nodes says that.
   */
  type RowState = 'updating' | 'ok' | 'error'
  interface RowStatus {
    state: RowState
    message: string
  }
  const rowStatus = ref<Record<string, RowStatus>>({})

  /**
   * Auto-clear timers, one per row. Kept outside `rowStatus` so replacing a
   * status never strands a timer that would later wipe its successor.
   */
  const statusTimers = new Map<string, ReturnType<typeof setTimeout>>()

  /**
   * How long the transient line stays. Success matches CopyIcon's in-place
   * confirmation (3s) — long enough to read, short enough that the row goes back
   * to showing its refreshed figures, which are the card's actual content.
   *
   * A failure holds longer on purpose: it names a provider and a cause, it is the
   * one outcome that asks the operator to do something, and unlike a success it
   * cannot be re-derived from the numbers left on the row.
   */
  const OK_TTL_MS = 3000
  const ERROR_TTL_MS = 8000

  function setRowStatus(id: string, status: RowStatus, ttl?: number) {
    const pending = statusTimers.get(id)
    if (pending) clearTimeout(pending)
    statusTimers.delete(id)

    rowStatus.value = { ...rowStatus.value, [id]: status }

    if (ttl === undefined) return
    statusTimers.set(
      id,
      setTimeout(() => {
        statusTimers.delete(id)
        const { [id]: _cleared, ...rest } = rowStatus.value
        rowStatus.value = rest
      }, ttl),
    )
  }

  // A row can be removed (or the whole dashboard left) while a status is still
  // showing; a timer firing after that would write to a dead component.
  onBeforeUnmount(() => {
    statusTimers.forEach((timer) => clearTimeout(timer))
    statusTimers.clear()
  })

  /**
   * Fetch ONE subscription from its provider and report on its row.
   *
   * Shared by the header's "update all" and each row's own button so a single
   * refresh and a bulk one cannot report differently — same states, same
   * messages, same timings.
   */
  async function refreshOne(subscription: Subscription) {
    setRowStatus(subscription.id, {
      state: 'updating',
      message: t('overview.subscriptions.rowUpdating'),
    })
    try {
      const response = await subscriptionService.updateSubscriptionContent(subscription.id)
      setRowStatus(
        subscription.id,
        { state: 'ok', message: summarizeUpdate(response.data, t) },
        OK_TTL_MS,
      )
    } catch (err) {
      setRowStatus(
        subscription.id,
        { state: 'error', message: apiErrorMessage(err, t('overview.subscriptions.rowFailed')) },
        ERROR_TTL_MS,
      )
    }
  }

  /**
   * Pull one subscription from its provider — the row's own button.
   *
   * A provider is often slow or briefly unreachable while the others are fine,
   * and re-running all of them to retry one is both slow and rude to the
   * providers that already answered.
   */
  async function updateOne(id: string) {
    if (updating.value) return
    const subscription = subscriptions.value.find((item) => item.id === id)
    if (!subscription) return

    updating.value = true
    try {
      await refreshOne(subscription)
      await load(true)
    } finally {
      updating.value = false
    }
  }

  /**
   * Pull every subscription from its provider, then re-read.
   *
   * This card exists to answer "how much traffic is left and when does it run
   * out" — and those numbers only move when the provider is contacted. A button
   * that merely re-read the panel's own database would redraw the same figures,
   * which is why this is the update, not a refresh.
   *
   * Sequential on purpose. This runs on a home router; firing a fetch at every
   * provider at once is how the panel ends up timing out. The shared helper also
   * leaves the same service-recovery window as the Subscriptions page: an update
   * restarts sing-box, whose local DNS listener may not be ready the instant the
   * request resolves. The sequence also makes the per-row states legible: exactly
   * one row says "updating" at a time, so the card shows progress instead of
   * freezing until the last provider answers.
   *
   * One provider failing must not abandon the rest — refreshOne catches per
   * subscription and reports on that row.
   */
  async function updateAll() {
    if (updating.value || !subscriptions.value.length) return
    updating.value = true

    try {
      await updateSubscriptionsSequentially(subscriptions.value, refreshOne)

      // Re-read regardless: the ones that did succeed have new figures. Row
      // statuses survive it — they are keyed by ID, not by list position.
      await load(true)
      // A refresh can add or remove nodes, which moves the quality denominator.
      await loadProbe()
    } finally {
      updating.value = false
    }
  }

  interface SubscriptionRow {
    subscription: Subscription
    id: string
    name: string
    /** The provider's site, already vetted as a linkable http(s) URL. */
    officialUrl: string | null
    health: SubscriptionHealth
    /** What the health dot means, and why it is that colour. */
    healthDetail: string
    plan: PlanSummary
    /** Transient outcome of the last update, or null when there is none. */
    status: RowStatus | null
    /** Newest quality sample, or null when this subscription has none. */
    probe: ProbePoint | null
  }

  const rows = computed<SubscriptionRow[]>(() =>
    subscriptions.value.map((subscription) => {
      const plan = summarizePlan(subscription)
      return {
        subscription,
        id: subscription.id,
        name: subscription.name,
        officialUrl: safeExternalUrl(subscription.official_url),
        health: subscriptionHealth(subscription),
        healthDetail: healthDetail(subscription),
        plan,
        status: rowStatus.value[subscription.id] ?? null,
        probe: probeLatest.value[subscription.id] ?? null,
      }
    }),
  )

  const healthDotClass = (health: SubscriptionHealth) => {
    switch (health) {
      case 'ok':
        return 'bg-green-500'
      case 'outdated':
        return 'bg-amber-500'
      default:
        return 'bg-gray-400'
    }
  }

  const healthLabel = (health: SubscriptionHealth) => {
    switch (health) {
      case 'ok':
        return t('subscriptions.status.updated')
      case 'outdated':
        return t('subscriptions.status.outdated')
      default:
        return t('subscriptions.status.notUpdated')
    }
  }

  /**
   * Why the dot is the colour it is, in the operator's own terms.
   *
   * The state name alone ("outdated") does not answer the question that makes
   * someone hover an amber dot, which is "what happened to this subscription?".
   * The facts that produced the colour do: when it was last fetched, and the
   * cadence it is being judged against (subscriptionHealth mirrors the backend's
   * own refresh rule).
   */
  const healthDetail = (subscription: Subscription): string => {
    const health = subscriptionHealth(subscription)
    if (health === 'never') return t('overview.subscriptions.healthNeverDetail')

    const lastUpdate = new Date(subscription.last_update ?? '').getTime()
    // Unparseable timestamps are already treated as "not evidence of staleness"
    // by subscriptionHealth; say nothing rather than render "Invalid Date".
    const when = Number.isNaN(lastUpdate)
      ? ''
      : formatRelativeTime(lastUpdate, locale.value, Date.now())
    if (!when) return ''

    if (health === 'outdated') {
      return t('overview.subscriptions.healthOutdatedDetail', {
        when,
        interval: subscription.update_interval,
      })
    }
    return t('overview.subscriptions.healthOkDetail', { when })
  }

  /**
   * "in 12 days" / "expires today" / "expired". Falls back to the raw provider
   * string when the date could not be interpreted.
   */
  const expiryLabel = (plan: PlanSummary) => {
    const days = plan.daysUntilExpiry
    if (days === null) return plan.expiresLabel ?? ''
    if (days < 0) return t('overview.subscriptions.expired')
    if (days === 0) return t('overview.subscriptions.expiresToday')
    return t('overview.subscriptions.expiresInDays', { days }, days)
  }

  const expiryIsUrgent = (plan: PlanSummary) =>
    plan.daysUntilExpiry !== null && plan.daysUntilExpiry <= 7

  const expiryTitle = (plan: PlanSummary) =>
    plan.expiresLabel
      ? `${t('overview.subscriptions.expires')}: ${plan.expiresLabel}`
      : ''

  /**
   * Colour for a row's status line. Neutral while in flight — an update that has
   * not finished has no outcome yet, and colouring it green or red before the
   * provider answers would be a claim the card cannot back.
   */
  const statusToneClass = (state: RowState) => {
    switch (state) {
      case 'ok':
        return 'text-green-600 dark:text-green-400'
      case 'error':
        return 'text-red-600 dark:text-red-400'
      default:
        return 'text-gray-500 dark:text-gray-400'
    }
  }

  // Formats the count in the viewer's locale (e.g. Arabic-Indic digits).
  const formatCount = (value: number) => value.toLocaleString(locale.value)
  return { subscriptions, loading, refreshing, updating, failed, loaded, rows, SUBSCRIPTIONS_ROUTE, loadProbe, updateOne, updateAll, healthDotClass, healthLabel, expiryLabel, expiryIsUrgent, expiryTitle, statusToneClass, formatCount }
}
