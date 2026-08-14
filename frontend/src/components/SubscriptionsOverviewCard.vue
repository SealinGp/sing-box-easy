<script setup lang="ts">
/**
 * Overview card summarising the user's subscriptions and their plan quotas.
 *
 * The Subscriptions page is where subscriptions are managed; this is the
 * at-a-glance answer to "how much traffic do I have left and when does it
 * run out", which is what an operator actually opens the dashboard to check.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import {
  ArrowTopRightOnSquareIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  RectangleStackIcon,
} from '@heroicons/vue/24/outline'
import { subscriptionService } from '../services'
import { useNotify } from '../composables/useNotify'
import { summarizePlan, type PlanSummary } from '../utils/subscriptionInfo'
import { subscriptionHealth, type SubscriptionHealth } from '../utils/subscriptionHealth'
import type { Subscription } from '../types/api'

const { t, locale } = useI18n()
const notify = useNotify()

const subscriptions = ref<Subscription[]>([])
const loading = ref(true)
/** Set when the fetch failed, so the card can say so instead of showing "0". */
const failed = ref(false)

const SUBSCRIPTIONS_ROUTE = '/dashboard/outbounds/subscriptions'

onMounted(async () => {
  try {
    const response = await subscriptionService.getSubscriptions()
    subscriptions.value = response.data.subscriptions || []
  } catch (err) {
    failed.value = true
    notify.apiError(err, t('overview.subscriptions.loadFailed'))
  } finally {
    loading.value = false
  }
})

interface SubscriptionRow {
  id: string
  name: string
  health: SubscriptionHealth
  plan: PlanSummary
}

const rows = computed<SubscriptionRow[]>(() =>
  subscriptions.value.map((subscription) => ({
    id: subscription.id,
    name: subscription.name,
    health: subscriptionHealth(subscription),
    plan: summarizePlan(subscription),
  })),
)

/** How many need attention — surfaced as a single count, not per-row noise. */
const needsAttention = computed(
  () => rows.value.filter((row) => row.health !== 'ok').length,
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

// Formats the count in the viewer's locale (e.g. Arabic-Indic digits).
const formatCount = (value: number) => value.toLocaleString(locale.value)
</script>

<template>
  <div class="bg-white dark:bg-slate-800 p-6 rounded-surface shadow dark:shadow-float dark:shadow-slate-700/50">
    <div class="flex items-start justify-between gap-3 mb-4">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">
        {{ $t('overview.subscriptions.title') }}
      </h3>
      <RouterLink
        :to="SUBSCRIPTIONS_ROUTE"
        class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
      >
        {{ $t('overview.subscriptions.manage') }}
        <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
      </RouterLink>
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
        <span
          v-if="needsAttention > 0"
          class="ml-auto inline-flex items-center gap-1 text-xs font-medium text-amber-600 dark:text-amber-400"
        >
          <ExclamationTriangleIcon class="h-3.5 w-3.5" />
          {{ $t('overview.subscriptions.needAttention', { count: needsAttention }, needsAttention) }}
        </span>
      </div>

      <!-- Per-subscription plan detail -->
      <ul class="space-y-3">
        <li
          v-for="row in rows"
          :key="row.id"
          class="pt-3 border-t border-gray-100 dark:border-gray-700 first:pt-0 first:border-t-0"
        >
          <div class="flex items-center gap-2 min-w-0">
            <span
              class="h-2 w-2 rounded-pill flex-shrink-0"
              :class="healthDotClass(row.health)"
              :title="healthLabel(row.health)"
            ></span>
            <span
              class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate"
              :title="row.name"
            >
              {{ row.name }}
            </span>
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
        </li>
      </ul>
    </div>
  </div>
</template>
