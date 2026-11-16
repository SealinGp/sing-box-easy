<script setup lang="ts">
import TooltipTarget from "./TooltipTarget.vue"
/**
 * A subscription plan's traffic, expiry, and provider-defined details.
 *
 * Both the overview and the management table need the same interpretation of
 * a PlanSummary. This module owns that presentation—including its teleported
 * tooltips—so callers only choose whether expiry is already shown elsewhere.
 */
import { computed } from 'vue'
import Tooltip from './Tooltip.vue'
import { useI18n } from 'vue-i18n'
import { ClockIcon } from '@heroicons/vue/24/outline'
import { formatPlanExtras, type PlanSummary } from '../utils/subscriptionInfo'

const { plan, subscriptionName, showExpiry = true, compact = false } = defineProps<{
  plan: PlanSummary
  subscriptionName: string
  showExpiry?: boolean
  compact?: boolean
}>()

const { t } = useI18n()

const usagePercent = computed(() =>
  plan.usedRatio === null ? null : Math.round(plan.usedRatio * 100),
)

/** Quota turns amber past 75% and red past 90%, before it runs out. */
const usageBarClass = computed(() => {
  const ratio = plan.usedRatio
  if (ratio === null) return 'bg-teal-500 dark:bg-teal-400'
  if (ratio >= 0.9) return 'bg-red-500'
  if (ratio >= 0.75) return 'bg-amber-500'
  return 'bg-teal-500 dark:bg-teal-400'
})

const extrasText = computed(() => formatPlanExtras(plan.extras))
const compactLabel = computed(() =>
  plan.remainingLabel ?? plan.usedLabel ?? plan.totalLabel ??
  (extrasText.value || t('overview.subscriptions.noPlanInfo')),
)

const expiryLabel = computed(() => {
  const days = plan.daysUntilExpiry
  if (days === null) return plan.expiresLabel ?? ''
  if (days < 0) return t('overview.subscriptions.expired')
  if (days === 0) return t('overview.subscriptions.expiresToday')
  return t('overview.subscriptions.expiresInDays', { days }, days)
})

const expiryIsUrgent = computed(
  () => plan.daysUntilExpiry !== null && plan.daysUntilExpiry <= 7,
)

const expiryTitle = computed(() =>
  plan.expiresLabel
    ? `${t('overview.subscriptions.expires')}: ${plan.expiresLabel}`
    : '',
)

</script>

<template>
  <div class="min-w-0 space-y-2">
    <!-- A usable Used/Total pair gets the same compact visual in every view. -->
    <Tooltip v-if="plan.usedRatio !== null" :width="192" match-anchor-width>
      <template #default="{ describedBy }">
      <div
        class="relative w-full min-w-0 cursor-default overflow-hidden rounded-md border border-border bg-bg-tertiary focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500"
        role="progressbar"
        tabindex="0"
        :aria-valuenow="usagePercent ?? undefined"
        aria-valuemin="0"
        aria-valuemax="100"
        :aria-label="$t('overview.subscriptions.usageLabel', { name: subscriptionName })"
        :aria-describedby="describedBy"
      >
        <div
          class="quota-fill absolute inset-y-0 left-0 overflow-hidden opacity-40 transition-[width] duration-300 dark:opacity-45"
          :class="usageBarClass"
          :style="{ width: `${usagePercent}%` }"
          aria-hidden="true"
        ></div>
        <div class="relative flex min-h-7 items-center justify-between gap-3 px-2 py-1 text-xs font-medium tabular-nums text-gray-900 dark:text-gray-100">
          <span class="truncate">{{ plan.usedLabel }} / {{ plan.totalLabel }}</span>
          <span class="ml-auto shrink-0">{{ usagePercent }}%</span>
        </div>
      </div>
      </template>
      <template #content>
        <p v-if="plan.usedLabel">
          {{ $t('overview.subscriptions.used') }}:
          <span class="font-medium">{{ plan.usedLabel }} / {{ plan.totalLabel }}</span>
        </p>
        <p v-if="plan.remainingLabel">
          {{ $t('overview.subscriptions.remaining') }}:
          <span class="font-medium">{{ plan.remainingLabel }}</span>
        </p>
        <p v-if="compact && extrasText" class="break-words">{{ extrasText }}</p>
      </template>
    </Tooltip>

    <Tooltip v-else-if="compact" :width="320">
      <template #default="{ describedBy }">
        <span tabindex="0" class="block truncate text-xs focus-visible:outline-2 focus-visible:outline-primary-500" :aria-describedby="describedBy">{{ compactLabel }}</span>
      </template>
      <template #content>
        <p v-if="plan.usedLabel">{{ $t('overview.subscriptions.used') }}: {{ plan.usedLabel }}</p>
        <p v-if="plan.totalLabel">{{ $t('overview.subscriptions.total') }}: {{ plan.totalLabel }}</p>
        <p v-if="plan.remainingLabel">{{ $t('overview.subscriptions.remaining') }}: {{ plan.remainingLabel }}</p>
        <p v-if="extrasText" class="break-words">{{ extrasText }}</p>
        <p v-if="!plan.usedLabel && !plan.totalLabel && !plan.remainingLabel && !extrasText">{{ $t('overview.subscriptions.noPlanInfo') }}</p>
      </template>
    </Tooltip>

    <!-- Unlimited and provider-specific values can have no meaningful ratio. -->
    <p
      v-else-if="plan.usedLabel || plan.remainingLabel"
      class="flex flex-wrap gap-x-2 text-xs text-gray-500 dark:text-gray-400"
    >
      <span v-if="plan.usedLabel">
        {{ $t('overview.subscriptions.used') }}:
        <span class="font-medium text-gray-700 dark:text-gray-300">{{ plan.usedLabel }}</span>
      </span>
      <span v-if="plan.remainingLabel">
        {{ $t('overview.subscriptions.remaining') }}:
        <span class="font-medium text-gray-700 dark:text-gray-300">{{ plan.remainingLabel }}</span>
      </span>
    </p>

    <TooltipTarget as="p"
      v-if="!compact && showExpiry && plan.expiresLabel"
      class="inline-flex items-center gap-1 text-xs"
      :class="
        expiryIsUrgent
          ? 'font-medium text-red-600 dark:text-red-400'
          : 'text-gray-500 dark:text-gray-400'
      "
      :text="expiryTitle"
    >
      <ClockIcon class="h-3.5 w-3.5 shrink-0" />
      {{ expiryLabel }}
    </TooltipTarget>

    <!-- Provider-defined fields stay one line; the tooltip contains all text. -->
    <Tooltip v-if="plan.extras.length && !compact" :width="384">
      <template #default="{ describedBy }">
      <p
        class="truncate rounded-pill bg-gray-100 px-2 py-0.5 text-xs text-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:bg-gray-700 dark:text-gray-300"
        tabindex="0"
        :aria-label="extrasText"
        :aria-describedby="describedBy"
      >
        {{ extrasText }}
      </p>
      </template>
      <template #content>
        <p
          v-for="(entry, index) in plan.extras"
          :key="`${entry.key}-${index}`"
          class="break-words"
        >
          {{ formatPlanExtras([entry]) }}
        </p>
      </template>
    </Tooltip>

    <p v-if="!compact && !plan.hasAny" class="text-xs text-gray-400 dark:text-gray-500">
      {{ $t('overview.subscriptions.noPlanInfo') }}
    </p>


  </div>
</template>

<style scoped>
.quota-fill::after {
  content: '';
  position: absolute;
  inset: 0 -32px 0 0;
  background: repeating-linear-gradient(135deg, transparent 0 11.3137px, rgb(255 255 255 / 45%) 11.3137px 22.6274px);
  animation: quota-flow 1.8s linear infinite;
}

@keyframes quota-flow {
  from { transform: translateX(-32px); }
  to { transform: translateX(0); }
}

@media (prefers-reduced-motion: reduce) {
  .quota-fill { transition: none; }
  .quota-fill::after { animation: none; }
}
</style>
