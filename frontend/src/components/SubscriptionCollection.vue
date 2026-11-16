<script setup lang="ts">
import TooltipTarget from "./TooltipTarget.vue"
import Tooltip from './Tooltip.vue'
import CardActions from './CardActions.vue'
import TruncatedText from './TruncatedText.vue'
import SubscriptionActions from './SubscriptionActions.vue'
import { computed, type UnwrapRef } from 'vue'
import { useI18n } from 'vue-i18n'
import type { useSubscriptionsPage } from '../composables/useSubscriptionsPage'
import Button from "./Button.vue";
import StatusIndicator from "./StatusIndicator.vue";
import Table from "./Table.vue";
import CopyIcon from "./CopyIcon.vue";
import SubscriptionQualityCell from "./SubscriptionQualityCell.vue";
import SubscriptionQuotaDetails from "./SubscriptionQuotaDetails.vue";
import { ArrowTopRightOnSquareIcon, ClockIcon } from '@heroicons/vue/24/outline'
const { page } = defineProps<{ page: UnwrapRef<ReturnType<typeof useSubscriptionsPage>> }>()
const { t } = useI18n()
const { summarizePlan, officialLink, formatDate, getSubscriptionStatus } = page
const subscriptions = computed(() => page.subscriptions)
const isUpdating = computed(() => page.isUpdating)
const probeLatest = computed(() => page.probeLatest)
const { loadProbeStatus, openAddModal, openEditModal, deleteSubscription, updateSubscription } = page
const columns = computed(() => [
  { key: 'name', label: t('subscriptions.table.subscription'), class: 'w-56' },
  { key: 'status', label: t('subscriptions.table.status'), class: 'w-36' },
  { key: 'quality', label: t('subProbe.column'), class: 'w-64' },
  { key: 'quota', label: t('subscriptions.table.info'), class: 'w-56' },
  { key: 'expiry', label: t('overview.subscriptions.expires'), class: 'w-36' },
  { key: 'updated', label: t('subscriptions.table.lastUpdate'), class: 'w-44' },
  { key: 'actions', minWidth: 144, label: t('subscriptions.table.actions'), class: 'w-36' },
])
</script>
<template><div>
<Table class="subscription-list table-surface" table-class="table-fixed" :columns="columns" column-storage-key="subscriptions" :loading="page.isLoading && !subscriptions.length" :empty="!subscriptions.length">
<template #empty><p>{{ t('subscriptions.empty.title') }}</p><Button class="mt-3" @click="openAddModal">{{ t('subscriptions.add') }}</Button></template>
<tr v-for="subscription in subscriptions" :key="subscription.id">
<td><TruncatedText :text="subscription.name" /></td>
<td><StatusIndicator :variant="getSubscriptionStatus(subscription).type" :busy="isUpdating.includes(subscription.id)"><TruncatedText :text="getSubscriptionStatus(subscription).text" /></StatusIndicator></td>
<td data-cell-tooltip="false"><SubscriptionQualityCell size="list" :point="probeLatest[subscription.id]" :subscription="subscription" @refresh="loadProbeStatus" /></td>
<td data-cell-tooltip="false"><SubscriptionQuotaDetails compact :show-expiry="false" :plan="summarizePlan(subscription)" :subscription-name="subscription.name" /></td>
<td><TruncatedText :text="summarizePlan(subscription).expiresLabel ?? '—'" /></td>
<td><TruncatedText :text="formatDate(subscription.last_update)" /></td>
<td class="col-actions">
            <SubscriptionActions :subscription="subscription" :busy="isUpdating.includes(subscription.id)" @update="updateSubscription(subscription)" @edit="openEditModal(subscription)" @delete="deleteSubscription(subscription)" />
          </td></tr>
<template #cards="{ selected, toggle }"><div class="subscription-cards">        <article class="subscription-card"
          v-for="subscription in subscriptions"
          :key="subscription.id"
          :data-collection-selected="selected.has(subscription.id)"
        >
<input type="checkbox" class="collection-checkbox" :aria-label="t('common.collection.select')" :checked="selected.has(subscription.id)" @change="toggle(subscription.id)" />
<section>
            <div class="flex flex-col gap-1 max-w-[18rem]">
<Tooltip v-if="officialLink(subscription)" class="min-w-0">
<template #content>{{ $t('subscriptions.tooltip.openSite', { name: subscription.name }) }}</template>
<template #default="{ describedBy }"><a
                :href="officialLink(subscription)!"
                target="_blank"
                rel="noopener noreferrer"
                class="group inline-flex items-center gap-1 min-w-0 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
                :aria-describedby="describedBy"
              >
                <span class="truncate">{{ subscription.name }}</span>
                <ArrowTopRightOnSquareIcon class="h-3 w-3 shrink-0 opacity-60 group-hover:opacity-100" />
              </a></template></Tooltip>
              <TooltipTarget as="span"
                v-else
                class="text-sm font-medium text-gray-900 dark:text-gray-100 truncate"
                :text="subscription.name"
              >
                {{ subscription.name }}
              </TooltipTarget>
<div class="flex items-center gap-1 min-w-0">
                <span class="text-xs text-gray-400 dark:text-gray-500 shrink-0">
                  {{ $t("subscriptions.table.id") }}
                </span>
                <TooltipTarget as="span"
                  class="font-mono text-xs text-gray-500 dark:text-gray-400 truncate"
                  :text="subscription.id"
                >
                  {{ subscription.id }}
                </TooltipTarget>
                <CopyIcon
                  :text="subscription.id"
                  size="xs"
                  :label="$t('subscriptions.tooltip.copyId')"
                />
              </div>

              <div class="flex items-center gap-1 min-w-0">
                <TooltipTarget as="span"
                  class="text-xs text-gray-500 dark:text-gray-400 truncate"
                  :text="subscription.url"
                >
                  {{ subscription.url }}
                </TooltipTarget>
                <CopyIcon
                  :text="subscription.url"
                  :label="$t('subscriptions.tooltip.copy')"
                />
              </div>
            </div>
          </section>
<section>
            <div class="flex flex-col items-start gap-1.5">
              <StatusIndicator :variant="getSubscriptionStatus(subscription).type" :busy="isUpdating.includes(subscription.id)">
                {{ getSubscriptionStatus(subscription).text }}
              </StatusIndicator>
              <div class="flex items-center gap-1.5 flex-wrap">
                <span class="text-xs text-text-secondary">{{ $t('subscriptions.form.autoUpdate') }}:</span>
                <StatusIndicator :variant="subscription.auto_update ? 'success' : 'secondary'">
                  {{ subscription.auto_update ? $t('subscriptions.enabled') : $t('subscriptions.disabled') }}
                </StatusIndicator>
                <TooltipTarget as="span" v-if="subscription.auto_update && subscription.update_interval" class="inline-flex items-center gap-1 text-xs text-text-secondary" :text="$t('subscriptions.form.updateInterval')">
                  <ClockIcon class="h-3 w-3" />{{ subscription.update_interval }}
                </TooltipTarget>
              </div>
              <TooltipTarget as="span"
                class="text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap"
                :text="$t('subscriptions.table.lastUpdate')"
              >
                {{ $t("subscriptions.table.lastUpdate") }}: {{ formatDate(subscription.last_update) }}
              </TooltipTarget>
            </div>
          </section>
<section>
            <SubscriptionQualityCell
              :point="probeLatest[subscription.id]"
              :subscription="subscription"
              @refresh="loadProbeStatus"
            />
          </section>
<section>
            <SubscriptionQuotaDetails
              :plan="summarizePlan(subscription)"
              :subscription-name="subscription.name"
            />
          </section>
          <CardActions>
            <SubscriptionActions expanded :subscription="subscription" :busy="isUpdating.includes(subscription.id)" @update="updateSubscription(subscription)" @edit="openEditModal(subscription)" @delete="deleteSubscription(subscription)" />
          </CardActions>
        </article>
</div></template></Table></div></template>
<style scoped>
.subscription-cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr)); gap: 16px; align-items: start; }
.subscription-card { position: relative; min-width: 0; padding: 24px; border: 1px solid var(--color-border); border-radius: 16px; background: var(--color-bg-secondary); display: flex; flex-direction: column; gap: 18px; }

.subscription-card :deep(.max-w-\[18rem\]) { max-width: 100%; }
.subscription-list :deep(td) { height: 52px; vertical-align: middle; overflow: hidden; }
.subscription-value, .subscription-name { display: block; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 100%; }
.subscription-name { color: var(--color-primary-600); cursor: pointer; text-align: left; }
.collection-enter-active { transition: opacity 220ms var(--motion-enter), transform 220ms var(--motion-enter); }
.collection-leave-active { transition: opacity 120ms var(--motion-exit), transform 120ms var(--motion-exit); pointer-events: none; }
.collection-enter-from, .collection-leave-to { opacity: 0; transform: translateY(6px) scale(.995); }
@media (prefers-reduced-motion: reduce) { .collection-enter-active, .collection-leave-active { transition: none; transform: none; } }
</style>
