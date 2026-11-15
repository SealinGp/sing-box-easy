<script setup lang="ts">
import TooltipTarget from "./TooltipTarget.vue"
import { RouterLink } from 'vue-router'
import { ArrowTopRightOnSquareIcon, CheckCircleIcon, CloudArrowDownIcon, ClockIcon, ExclamationCircleIcon, RectangleStackIcon } from '@heroicons/vue/24/outline'
import Button from './Button.vue'
import Tooltip from './Tooltip.vue'
import SubscriptionQualityCell from './SubscriptionQualityCell.vue'
import SubscriptionQuotaDetails from './SubscriptionQuotaDetails.vue'
import { useSubscriptionsOverview } from '../composables/useSubscriptionsOverview'
const { subscriptions, loading, refreshing, updating, failed, loaded, rows, SUBSCRIPTIONS_ROUTE, loadProbe, updateOne, updateAll, healthDotClass, healthLabel, expiryLabel, expiryIsUrgent, expiryTitle, statusToneClass, formatCount } = useSubscriptionsOverview()
</script>

<template>
  <div class="overview-card-frame bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <div class="flex items-center justify-between gap-3 mb-3">
      <h3 class="flex min-w-0 items-center gap-2 text-lg font-semibold text-gray-700 dark:text-gray-300">
        <span>{{ $t('overview.subscriptions.title') }}</span>
        <span
          v-if="loaded"
          class="inline-flex min-w-5 shrink-0 items-center justify-center rounded-pill bg-gray-100 px-1.5 py-0.5 text-xs font-medium tabular-nums text-gray-600 dark:bg-gray-700 dark:text-gray-300"
        >
          {{ formatCount(subscriptions.length) }}
        </span>
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

    <div class="overview-card-body quiet-scrollbar" tabindex="0" role="region" :aria-label="$t('overview.subscriptions.title')">
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
            <Tooltip class="shrink-0" :width="240">
              <template #default="{ describedBy }">
                <span class="-m-1.5 block cursor-default p-1.5 focus:outline-none" tabindex="0" role="img"
                  :aria-label="`${healthLabel(row.health)}${row.healthDetail ? '. ' + row.healthDetail : ''}`" :aria-describedby="describedBy">
                  <span class="block h-2 w-2 rounded-pill" :class="healthDotClass(row.health)" />
                </span>
              </template>
              <template #content>
                <span class="block font-medium">{{ healthLabel(row.health) }}</span>
                <span v-if="row.healthDetail" class="mt-0.5 block text-text-secondary">{{ row.healthDetail }}</span>
              </template>
            </Tooltip>
            <!--
              This card exists to answer "how much is left and when does it run
              out"; the action both answers lead to is topping up, which lives
              on the provider's site. Linking the name is the shortest path
              from the number to the fix. rel="noopener noreferrer" because the
              href is provider-controlled.
            -->
            <TooltipTarget as="a"
              v-if="row.officialUrl"
              :href="row.officialUrl"
              target="_blank"
              rel="noopener noreferrer"
              class="text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 truncate transition-colors"
              :text="$t('overview.subscriptions.openSite', { name: row.name })"
            >
              {{ row.name }}
            </TooltipTarget>
            <TooltipTarget as="span"
              v-else
              class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate"
              :text="row.name"
            >
              {{ row.name }}
            </TooltipTarget>

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
            <TooltipTarget as="button"
              type="button"
              class="cursor-pointer shrink-0 rounded p-0.5 text-gray-400 transition hover:text-primary-600 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 disabled:cursor-not-allowed disabled:opacity-40 dark:hover:text-primary-400 sm:opacity-0 sm:group-hover:opacity-100 sm:group-focus-within:opacity-100"
              :class="{ 'sm:opacity-100': row.status?.state === 'updating' }"
              :disabled="updating"
              :text="$t('overview.subscriptions.updateOne', { name: row.name })"
              :aria-label="$t('overview.subscriptions.updateOne', { name: row.name })"
              @click="updateOne(row.id)"
            >
              <CloudArrowDownIcon
                class="h-3.5 w-3.5"
                :class="{ 'animate-pulse': row.status?.state === 'updating' }"
              />
            </TooltipTarget>

            <TooltipTarget as="span"
              v-if="row.plan.expiresLabel"
              class="ml-auto inline-flex items-center gap-1 text-xs flex-shrink-0"
              :class="
                expiryIsUrgent(row.plan)
                  ? 'text-red-600 dark:text-red-400 font-medium'
                  : 'text-gray-500 dark:text-gray-400'
              "
              :text="expiryTitle(row.plan)"
            >
              <ClockIcon class="h-3.5 w-3.5" />
              {{ expiryLabel(row.plan) }}
            </TooltipTarget>
          </div>

          <!-- The shared module owns the bar, fallbacks, extras, and tooltips.
               Expiry stays in the name line above on this compact card. -->
          <SubscriptionQuotaDetails
            class="mt-2"
            :plan="row.plan"
            :subscription-name="row.name"
            :show-expiry="false"
          />

          <div v-if="row.probe" class="mt-2 flex flex-wrap items-center gap-x-4 gap-y-2">
            <!--
              Node quality: how much of this feed actually works right now. It
              sits with quota and expiry because it answers the other half of
              "should I renew this" — a subscription with plenty of traffic left
              and 0% reachable nodes is worth nothing.

              Rendered only when a measurement exists: a deployment without
              clash_api never probes, and an empty placeholder on every row would
              be noise about a feature that deployment does not have.
            -->
            <!-- Rendering, interaction, and the trend dialog share the same
                 module as the full Subscriptions table. -->
            <SubscriptionQualityCell
              v-if="row.probe"
              :point="row.probe"
              :subscription="row.subscription"
              size="small"
              @refresh="loadProbe"
            />

          </div>

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

  </div>
</template>
