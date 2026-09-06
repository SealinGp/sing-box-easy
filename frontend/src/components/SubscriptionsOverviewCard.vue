<script setup lang="ts">
/**
 * Overview card summarising the user's subscriptions and their plan quotas.
 *
 * The Subscriptions page is where subscriptions are managed; this is the
 * at-a-glance answer to "how much traffic do I have left and when does it
 * run out", which is what an operator actually opens the dashboard to check.
 */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import {
  ArrowTopRightOnSquareIcon,
  CheckCircleIcon,
  CloudArrowDownIcon,
  ClockIcon,
  ExclamationCircleIcon,
  RectangleStackIcon,
} from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import { subProbeService, subscriptionService } from '../services'
import { useNotify } from '../composables/useNotify'
import { summarizePlan, type PlanSummary } from '../utils/subscriptionInfo'
import { safeExternalUrl } from '../utils/safeExternalUrl'
import { summarizeUpdate } from '../utils/subscriptionUpdate'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { subscriptionHealth, type SubscriptionHealth } from '../utils/subscriptionHealth'
import { formatRelativeTime } from '../utils/relativeTime'
import type { Subscription } from '../types/api'
import type { ProbePoint } from '../types/subprobe'
import { availabilityRatio, formatAvailability, formatLatency } from '../utils/probeChart'

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
 * provider at once is how the panel ends up timing out. The Subscriptions page
 * does the same. The sequence is also what makes the per-row states legible:
 * exactly one row says "updating" at a time, so the card shows progress instead
 * of freezing until the last provider answers.
 *
 * One provider failing must not abandon the rest — refreshOne catches per
 * subscription and reports on that row.
 */
async function updateAll() {
  if (updating.value || !subscriptions.value.length) return
  updating.value = true

  try {
    for (const subscription of subscriptions.value) {
      await refreshOne(subscription)
    }

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
  subscriptions.value.map((subscription) => ({
    id: subscription.id,
    name: subscription.name,
    officialUrl: safeExternalUrl(subscription.official_url),
    health: subscriptionHealth(subscription),
    healthDetail: healthDetail(subscription),
    plan: summarizePlan(subscription),
    status: rowStatus.value[subscription.id] ?? null,
    probe: probeLatest.value[subscription.id] ?? null,
  })),
)

/** Availability colour, matching the Subscriptions page's quality column. */
const probeToneClass = (point: ProbePoint) => {
  const ratio = availabilityRatio(point)
  if (ratio >= 0.9) return 'text-green-600 dark:text-green-400'
  if (ratio >= 0.5) return 'text-amber-600 dark:text-amber-400'
  return 'text-red-600 dark:text-red-400'
}

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

/** Quota bar turns amber past 75% and red past 90% — before it runs out. */
const usageBarClass = (ratio: number) => {
  if (ratio >= 0.9) return 'bg-red-500'
  if (ratio >= 0.75) return 'bg-amber-500'
  return 'bg-primary-600'
}

const usagePercent = (ratio: number) => Math.round(ratio * 100)

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
</script>

<template>
  <div class="bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <div class="flex items-center justify-between gap-3 mb-3">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">
        {{ $t('overview.subscriptions.title') }}
      </h3>
      <div class="flex items-center gap-1 flex-shrink-0">
        <!--
          This pulls from the providers — it is not a view refresh. The card's
          whole content is quota and expiry, which only change when the provider
          is contacted, so re-reading the local database would redraw identical
          numbers.

          `CloudArrowDownIcon` is the app's mark for "fetch from the provider",
          shared with the Subscriptions page's Update All and its per-row update
          button; the circular arrow means "reload the view" and would say the
          wrong thing here.

          Icon-only, so it carries both `title` (pointer) and `aria-label`
          (assistive tech).
        -->
        <Button
          v-if="rows.length"
          variant="ghost"
          size="sm"
          action
          :disabled="loading || refreshing || updating"
          :title="$t('overview.subscriptions.updateTooltip')"
          :aria-label="$t('overview.subscriptions.update')"
          @click="updateAll"
        >
          <CloudArrowDownIcon
            class="h-3.5 w-3.5"
            :class="{ 'animate-pulse': updating }"
          />
        </Button>
        <RouterLink
          :to="SUBSCRIPTIONS_ROUTE"
          class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
        >
          {{ $t('overview.subscriptions.manage') }}
          <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
        </RouterLink>
      </div>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-6">
      <div class="animate-spin rounded-pill h-8 w-8 border-b-2 border-primary-600"></div>
    </div>

    <!-- A failed fetch must not be rendered as "you have 0 subscriptions". -->
    <p v-else-if="failed" class="py-4 text-sm text-gray-500 dark:text-gray-400">
      {{ $t('overview.subscriptions.loadFailed') }}
    </p>

    <div v-else-if="rows.length === 0" class="py-4">
      <div class="flex items-center gap-3 mb-3">
        <div class="flex items-center justify-center w-10 h-10 rounded-pill bg-gray-100 dark:bg-gray-700">
          <RectangleStackIcon class="h-5 w-5 text-gray-400" />
        </div>
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ $t('overview.subscriptions.empty') }}
        </p>
      </div>
      <RouterLink
        :to="SUBSCRIPTIONS_ROUTE"
        class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400"
      >
        {{ $t('overview.subscriptions.addFirst') }}
      </RouterLink>
    </div>

    <div v-else class="space-y-4">
      <!-- Headline counts -->
      <div class="flex items-baseline gap-2">
        <span class="text-2xl font-bold text-gray-900 dark:text-gray-100">
          {{ formatCount(rows.length) }}
        </span>
        <span class="text-sm text-gray-500 dark:text-gray-400">
          {{ $t('overview.subscriptions.count', rows.length) }}
        </span>
      </div>

      <!-- Per-subscription plan detail -->
      <ul class="space-y-3">
        <!--
          `group` sits on the whole row, not on the name line: everything in
          here — the quota bar, the remaining figure, the plan chips — is this
          subscription, so pointing at any of it should offer its action. A
          reveal armed only by the name made the target a few characters wide
          and, on the rows whose name is short, easy to miss entirely.
        -->
        <li
          v-for="row in rows"
          :key="row.id"
          class="group pt-3 border-t border-gray-100 dark:border-gray-700 first:pt-0 first:border-t-0"
        >
          <div class="flex items-center gap-2 min-w-0">
            <!--
              An amber dot raises a question ("what happened?"), so hovering it
              has to answer one. A native `title` did not: it needs a ~1s hold
              on an 8px target, renders outside the page, and can only carry the
              state's NAME — which is the half the colour already told you.

              So: a real bubble, shown instantly on hover AND on keyboard focus
              (tabindex, or the explanation would be mouse-only), with the state
              plus the facts behind it. The padding wrapper turns the 8px dot
              into a ~20px hit area without changing how it looks.
            -->
            <span
              class="group/health relative -m-1.5 flex-shrink-0 cursor-default p-1.5 focus:outline-none"
              tabindex="0"
              role="img"
              :aria-label="`${healthLabel(row.health)}${row.healthDetail ? '. ' + row.healthDetail : ''}`"
            >
              <span class="block h-2 w-2 rounded-pill" :class="healthDotClass(row.health)"></span>
              <span
                role="tooltip"
                class="pointer-events-none absolute bottom-full left-0 z-20 mb-1 w-max max-w-[15rem] rounded-md bg-gray-900/95 px-2 py-1 text-left text-xs text-white opacity-0 shadow-lg transition-opacity duration-100 group-hover/health:opacity-100 group-focus/health:opacity-100 dark:bg-gray-700"
              >
                <span class="block font-medium">{{ healthLabel(row.health) }}</span>
                <span v-if="row.healthDetail" class="mt-0.5 block text-gray-300">
                  {{ row.healthDetail }}
                </span>
              </span>
            </span>
            <!--
              This card exists to answer "how much is left and when does it run
              out"; the action both answers lead to is topping up, which lives
              on the provider's site. Linking the name is the shortest path
              from the number to the fix. rel="noopener noreferrer" because the
              href is provider-controlled.
            -->
            <a
              v-if="row.officialUrl"
              :href="row.officialUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 truncate transition-colors"
              :title="$t('overview.subscriptions.openSite', { name: row.name })"
            >
              {{ row.name }}
            </a>
            <span
              v-else
              class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate"
              :title="row.name"
            >
              {{ row.name }}
            </span>

            <!--
              Fetch just this subscription. Hidden until anywhere in the row is
              hovered, so four subscriptions do not read as four buttons — the
              card is a summary first, and the action belongs to whichever row
              you are pointing at.

              Three states it must survive:
                - keyboard: `group-focus-within` reveals it when it is tabbed
                  to, or it would be a control nobody without a mouse can find;
                - touch: no hover exists, so below `sm` it is simply always on;
                - in flight: stays visible while THIS row updates, otherwise the
                  spinner would vanish the moment the pointer leaves.
            -->
            <button
              type="button"
              class="cursor-pointer shrink-0 rounded p-0.5 text-gray-400 transition hover:text-primary-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-40 dark:hover:text-primary-400 sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100"
              :class="{ 'sm:opacity-100': row.status?.state === 'updating' }"
              :disabled="updating"
              :title="$t('overview.subscriptions.updateOne', { name: row.name })"
              :aria-label="$t('overview.subscriptions.updateOne', { name: row.name })"
              @click="updateOne(row.id)"
            >
              <CloudArrowDownIcon
                class="h-3.5 w-3.5"
                :class="{ 'animate-pulse': row.status?.state === 'updating' }"
              />
            </button>

            <span
              v-if="row.plan.expiresLabel"
              class="ml-auto inline-flex items-center gap-1 text-xs flex-shrink-0"
              :class="
                expiryIsUrgent(row.plan)
                  ? 'text-red-600 dark:text-red-400 font-medium'
                  : 'text-gray-500 dark:text-gray-400'
              "
              :title="expiryTitle(row.plan)"
            >
              <ClockIcon class="h-3.5 w-3.5" />
              {{ expiryLabel(row.plan) }}
            </span>
          </div>

          <!-- Quota bar, only when both Used and Total were reported -->
          <div v-if="row.plan.usedRatio !== null" class="mt-2">
            <div class="flex items-center justify-between text-xs mb-1">
              <span class="text-gray-500 dark:text-gray-400">
                {{ row.plan.usedLabel }} / {{ row.plan.totalLabel }}
              </span>
              <span class="font-mono text-gray-500 dark:text-gray-400">
                {{ usagePercent(row.plan.usedRatio) }}%
              </span>
            </div>
            <div
              class="h-1.5 w-full rounded-pill bg-gray-200 dark:bg-gray-700 overflow-hidden"
              role="progressbar"
              :aria-valuenow="usagePercent(row.plan.usedRatio)"
              aria-valuemin="0"
              aria-valuemax="100"
              :aria-label="$t('overview.subscriptions.usageLabel', { name: row.name })"
            >
              <div
                class="h-full rounded-pill transition-all duration-300"
                :class="usageBarClass(row.plan.usedRatio)"
                :style="{ width: `${usagePercent(row.plan.usedRatio)}%` }"
              ></div>
            </div>
            <p v-if="row.plan.remainingLabel" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ $t('overview.subscriptions.remaining') }}:
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ row.plan.remainingLabel }}</span>
            </p>
          </div>

          <!--
            Quota reported without a usable total (e.g. unlimited plans, which
            send total=0): show the figures we do have, without a bar.
          -->
          <p
            v-else-if="row.plan.usedLabel || row.plan.remainingLabel"
            class="mt-2 text-xs text-gray-500 dark:text-gray-400"
          >
            <span v-if="row.plan.usedLabel">
              {{ $t('overview.subscriptions.used') }}:
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ row.plan.usedLabel }}</span>
            </span>
            <span v-if="row.plan.usedLabel && row.plan.remainingLabel"> · </span>
            <span v-if="row.plan.remainingLabel">
              {{ $t('overview.subscriptions.remaining') }}:
              <span class="font-medium text-gray-700 dark:text-gray-300">{{ row.plan.remainingLabel }}</span>
            </span>
          </p>

          <!--
            Node quality: how much of this feed actually works right now. It
            sits with quota and expiry because it answers the other half of
            "should I renew this" — a subscription with plenty of traffic left
            and 0% reachable nodes is worth nothing.

            Rendered only when a measurement exists: a deployment without
            clash_api never probes, and an empty placeholder on every row would
            be noise about a feature that deployment does not have.
          -->
          <p v-if="row.probe" class="mt-2 text-xs text-gray-500 dark:text-gray-400">
            {{ $t('subProbe.column') }}:
            <span class="font-medium" :class="probeToneClass(row.probe)">
              {{ formatAvailability(row.probe) }}
            </span>
            <span class="text-gray-400 dark:text-gray-500">
              ({{ row.probe.reachable }}/{{ row.probe.total }})
            </span>
            <span v-if="row.probe.reachable > 0">
              · {{ formatLatency(row.probe.avg_ms) }}
            </span>
          </p>

          <!--
            Provider-defined entries, shown verbatim: their keys are arbitrary
            localized text, so the UI must not try to interpret them.
          -->
          <div v-if="row.plan.extras.length" class="mt-2 flex flex-wrap gap-1.5">
            <span
              v-for="(entry, index) in row.plan.extras"
              :key="`${entry.key}-${index}`"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-pill bg-gray-100 dark:bg-gray-700 text-xs"
              :title="entry.value ? `${entry.key}: ${entry.value}` : entry.key"
            >
              <span class="text-gray-500 dark:text-gray-400">{{ entry.key }}</span>
              <span v-if="entry.value" class="font-medium text-gray-700 dark:text-gray-200">
                {{ entry.value }}
              </span>
            </span>
          </div>

          <p
            v-if="!row.plan.hasAny"
            class="mt-2 text-xs text-gray-400 dark:text-gray-500"
          >
            {{ $t('overview.subscriptions.noPlanInfo') }}
          </p>

          <!--
            The outcome of "update all", on the row it belongs to. Transient by
            design: it answers a question the operator asked a second ago, and
            an outcome that stayed would still be on screen tomorrow claiming to
            describe the numbers above it.

            role="status" + aria-live: removing the toast also removed what
            screen readers announced, and this has to replace both halves.
          -->
          <Transition
            enter-active-class="transition duration-150 ease-out"
            enter-from-class="opacity-0 -translate-y-1"
            leave-active-class="transition duration-200 ease-in"
            leave-to-class="opacity-0"
          >
            <p
              v-if="row.status"
              class="mt-2 flex items-start gap-1.5 text-xs"
              :class="statusToneClass(row.status.state)"
              role="status"
              aria-live="polite"
            >
              <span
                v-if="row.status.state === 'updating'"
                class="mt-0.5 h-3 w-3 flex-shrink-0 animate-spin rounded-pill border-b-2 border-primary-600"
              ></span>
              <CheckCircleIcon
                v-else-if="row.status.state === 'ok'"
                class="mt-0.5 h-3.5 w-3.5 flex-shrink-0"
              />
              <ExclamationCircleIcon v-else class="mt-0.5 h-3.5 w-3.5 flex-shrink-0" />
              <!-- A provider's error text can be long; wrap it rather than
                   truncating away the part that names the cause. -->
              <span class="min-w-0 break-words">{{ row.status.message }}</span>
            </p>
          </Transition>
        </li>
      </ul>
    </div>
  </div>
</template>
