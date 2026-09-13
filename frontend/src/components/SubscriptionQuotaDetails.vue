<script setup lang="ts">
/**
 * A subscription plan's traffic, expiry, and provider-defined details.
 *
 * Both the overview and the management table need the same interpretation of
 * a PlanSummary. This module owns that presentation—including its teleported
 * tooltips—so callers only choose whether expiry is already shown elsewhere.
 */
import { computed, onBeforeUnmount, ref, useId } from 'vue'
import { useI18n } from 'vue-i18n'
import { ClockIcon } from '@heroicons/vue/24/outline'
import { formatPlanExtras, type PlanSummary } from '../utils/subscriptionInfo'

const { plan, subscriptionName, showExpiry = true } = defineProps<{
  plan: PlanSummary
  subscriptionName: string
  showExpiry?: boolean
}>()

const { t } = useI18n()
const componentId = useId()

const usagePercent = computed(() =>
  plan.usedRatio === null ? null : Math.round(plan.usedRatio * 100),
)

/** Quota turns amber past 75% and red past 90%, before it runs out. */
const usageBarClass = computed(() => {
  const ratio = plan.usedRatio
  if (ratio === null) return 'bg-primary-600'
  if (ratio >= 0.9) return 'bg-red-500'
  if (ratio >= 0.75) return 'bg-amber-500'
  return 'bg-primary-600'
})

const extrasText = computed(() => formatPlanExtras(plan.extras))

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

type TooltipKind = 'quota' | 'extras'
interface TooltipPosition {
  left: number
  top: number
  width: number
}

const activeTooltip = ref<TooltipKind | null>(null)
const tooltipPosition = ref<TooltipPosition | null>(null)
const quotaTooltipId = `${componentId}-quota`
const extrasTooltipId = `${componentId}-extras`

/**
 * Escape table/card scrollports and keep the tooltip inside the viewport.
 * Window listeners exist only while a tooltip is visible.
 */
function showTooltip(event: Event, kind: TooltipKind) {
  const bounds = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const margin = 8
  const availableWidth = Math.max(window.innerWidth - margin * 2, 0)
  const preferredWidth = kind === 'quota' ? Math.max(bounds.width, 192) : 384
  const width = Math.min(preferredWidth, availableWidth)
  const left = Math.max(
    margin,
    Math.min(bounds.left, window.innerWidth - width - margin),
  )

  activeTooltip.value = kind
  tooltipPosition.value = { left, top: bounds.top, width }
  window.addEventListener('scroll', hideTooltip, true)
  window.addEventListener('resize', hideTooltip)
}

function hideTooltip() {
  activeTooltip.value = null
  tooltipPosition.value = null
  window.removeEventListener('scroll', hideTooltip, true)
  window.removeEventListener('resize', hideTooltip)
}

function leaveTooltip(event: Event) {
  const target = event.currentTarget as HTMLElement
  if (!target.matches(':hover') && !target.contains(document.activeElement)) hideTooltip()
}

onBeforeUnmount(hideTooltip)
</script>

<template>
  <div class="min-w-0 space-y-2">
    <!-- A usable Used/Total pair gets the same compact visual in every view. -->
    <div
      v-if="plan.usedRatio !== null"
      @mouseenter="showTooltip($event, 'quota')"
      @mouseleave="leaveTooltip"
      @focusin="showTooltip($event, 'quota')"
      @focusout="leaveTooltip"
      @keydown.esc="hideTooltip"
    >
      <div
        class="relative w-full min-w-36 cursor-default overflow-hidden rounded-md bg-gray-100 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:bg-gray-700"
        role="progressbar"
        tabindex="0"
        :aria-valuenow="usagePercent ?? undefined"
        aria-valuemin="0"
        aria-valuemax="100"
        :aria-label="$t('overview.subscriptions.usageLabel', { name: subscriptionName })"
        :aria-describedby="activeTooltip === 'quota' ? quotaTooltipId : undefined"
      >
        <div
          class="quota-fill absolute inset-y-0 left-0 overflow-hidden opacity-25 transition-[width] duration-300 dark:opacity-35"
          :class="usageBarClass"
          :style="{ width: `${usagePercent}%` }"
          aria-hidden="true"
        ></div>
        <div class="relative flex min-h-7 items-center justify-between gap-3 px-2 py-1 text-xs font-medium tabular-nums text-gray-900 dark:text-gray-100">
          <span class="truncate">{{ plan.usedLabel }} / {{ plan.totalLabel }}</span>
          <span class="ml-auto shrink-0">{{ usagePercent }}%</span>
        </div>
      </div>
    </div>

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

    <p
      v-if="showExpiry && plan.expiresLabel"
      class="inline-flex items-center gap-1 text-xs"
      :class="
        expiryIsUrgent
          ? 'font-medium text-red-600 dark:text-red-400'
          : 'text-gray-500 dark:text-gray-400'
      "
      :title="expiryTitle"
    >
      <ClockIcon class="h-3.5 w-3.5 shrink-0" />
      {{ expiryLabel }}
    </p>

    <!-- Provider-defined fields stay one line; the tooltip contains all text. -->
    <div
      v-if="plan.extras.length"
      @mouseenter="showTooltip($event, 'extras')"
      @mouseleave="leaveTooltip"
      @focusin="showTooltip($event, 'extras')"
      @focusout="leaveTooltip"
      @keydown.esc="hideTooltip"
    >
      <p
        class="truncate rounded-pill bg-gray-100 px-2 py-0.5 text-xs text-gray-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 dark:bg-gray-700 dark:text-gray-300"
        tabindex="0"
        :aria-label="extrasText"
        :aria-describedby="activeTooltip === 'extras' ? extrasTooltipId : undefined"
      >
        {{ extrasText }}
      </p>
    </div>

    <p v-if="!plan.hasAny" class="text-xs text-gray-400 dark:text-gray-500">
      {{ $t('overview.subscriptions.noPlanInfo') }}
    </p>

    <Teleport to="body">
      <div
        v-if="activeTooltip === 'quota' && tooltipPosition"
        :id="quotaTooltipId"
        role="tooltip"
        class="pointer-events-none fixed z-50 -translate-y-full rounded-md bg-gray-900/95 px-2 py-1 text-xs text-white shadow-lg dark:bg-gray-700"
        :style="{ left: `${tooltipPosition.left}px`, top: `${tooltipPosition.top}px`, width: `${tooltipPosition.width}px` }"
      >
        <p v-if="plan.usedLabel">
          {{ $t('overview.subscriptions.used') }}:
          <span class="font-medium">{{ plan.usedLabel }} / {{ plan.totalLabel }}</span>
        </p>
        <p v-if="plan.remainingLabel">
          {{ $t('overview.subscriptions.remaining') }}:
          <span class="font-medium">{{ plan.remainingLabel }}</span>
        </p>
      </div>
    </Teleport>

    <Teleport to="body">
      <div
        v-if="activeTooltip === 'extras' && tooltipPosition && plan.extras.length"
        :id="extrasTooltipId"
        role="tooltip"
        class="pointer-events-none fixed z-50 -translate-y-full space-y-1 rounded-md bg-gray-900/95 px-2 py-1.5 text-xs text-white shadow-lg dark:bg-gray-700"
        :style="{ left: `${tooltipPosition.left}px`, top: `${tooltipPosition.top}px`, width: `${tooltipPosition.width}px` }"
      >
        <p
          v-for="(entry, index) in plan.extras"
          :key="`${entry.key}-${index}`"
          class="break-words"
        >
          {{ formatPlanExtras([entry]) }}
        </p>
      </div>
    </Teleport>
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
