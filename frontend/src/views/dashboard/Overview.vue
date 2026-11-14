<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { onBeforeRouteLeave } from 'vue-router'
import OverviewLayoutEditor from '../../components/OverviewLayoutEditor.vue'
import Button from '../../components/Button.vue'
import Skeleton from '../../components/Skeleton.vue'
import { useOverviewLayout } from '../../composables/useOverviewLayout'
import ServiceStatusCard from '../../components/ServiceStatusCard.vue'
import ConnectionsOverviewCard from '../../components/ConnectionsOverviewCard.vue'
import SubscriptionsOverviewCard from '../../components/SubscriptionsOverviewCard.vue'
import DnsProbeCard from '../../components/DnsProbeCard.vue'
import RouteProbeCard from '../../components/RouteProbeCard.vue'
import ApiEndpointsCard from '../../components/ApiEndpointsCard.vue'
import RouteTopologyCard from '../../components/RouteTopologyCard.vue'
import LatencyOverviewCard from '../../components/LatencyOverviewCard.vue'
import NetworkInfoOverviewCard from '../../components/NetworkInfoOverviewCard.vue'
import { overviewSlots } from '../../utils/overviewSort'

const cards = [
  { id: 'route-topology', component: RouteTopologyCard, title: 'routeFlow.title' },
  { id: 'service-status', component: ServiceStatusCard, title: 'overview.serviceStatus' },
  { id: 'connections', component: ConnectionsOverviewCard, title: 'overview.connections.title' },
  { id: 'latency', component: LatencyOverviewCard, title: 'overview.networkChecks.latency' },
  { id: 'network-info', component: NetworkInfoOverviewCard, title: 'overview.networkChecks.networkInfo' },
  { id: 'subscriptions', component: SubscriptionsOverviewCard, title: 'overview.subscriptions.title' },
  { id: 'dns-probe', component: DnsProbeCard, title: 'dnsProbe.title' },
  { id: 'route-probe', component: RouteProbeCard, title: 'routeProbe.title' },
  { id: 'api-endpoints', component: ApiEndpointsCard, title: 'overview.apis.title' },
]
/** The one card that takes a whole row: the flow diagram needs the width. */
const WIDE_CARD = 'route-topology'
const defaults = cards.map(card => card.id)
const { t } = useI18n()
const reorder = useOverviewLayout(defaults)
const { order, hidden, enabled, saving, loading, error, message, dirty, authEnabled, reset, load } = reorder
const editorCards = computed(() => order.value.map(id => cards.find(card => card.id === id)!))
const renderedCards = computed(() => order.value.filter(id => !hidden.value.includes(id)).map(id => cards.find(card => card.id === id)!))

// The editor and the live overview share row/column placement. Native CSS
// grid sizes each row to its tallest card, keeping subsequent rows aligned.
const twoColumns = window.matchMedia('(min-width: 768px)')
const threeColumns = window.matchMedia('(min-width: 1024px)')
const columns = ref(1)
const syncColumns = () => {
  columns.value = threeColumns.matches ? 3 : twoColumns.matches ? 2 : 1
}
const placements = computed(() => new Map(
  overviewSlots(renderedCards.value, columns.value).map(slot => [slot.id, slot]),
))
const tileStyle = (id: string) => {
  const slot = placements.value.get(id)
  return slot ? { gridColumn: `${slot.column} / span ${slot.span}`, gridRow: slot.row } : undefined
}
onMounted(() => {
  syncColumns()
  twoColumns.addEventListener('change', syncColumns)
  threeColumns.addEventListener('change', syncColumns)
})
onBeforeUnmount(() => {
  twoColumns.removeEventListener('change', syncColumns)
  threeColumns.removeEventListener('change', syncColumns)
})

// Leaving the overview discards its unsaved arrangement; saved layouts reload
// from the current account when this route is visited again.
onBeforeRouteLeave(() => { if (enabled.value) reorder.cancel() })
onMounted(load)
</script>

<template>
  <div class="page-shell">
    <div v-if="!enabled" class="flex justify-end mb-4">
      <Button variant="secondary" size="sm" :disabled="loading || saving" :loading="saving" @click="reorder.start">{{ t('overview.layout.customize') }}</Button>
    </div>
    <div v-if="enabled" class="overview-toolbar is-arranging mb-4 flex flex-wrap items-center gap-2" aria-live="polite">
      <div class="mr-auto min-w-0">
        <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('overview.layout.arranging') }}</p>
        <p class="hidden text-xs text-gray-500 dark:text-gray-400 sm:block">{{ t('overview.layout.hint') }}</p>
      </div>
      <Button variant="ghost" size="sm" action :disabled="saving" @click="reset">{{ t('overview.layout.reset') }}</Button>
      <Button variant="secondary" size="sm" action :disabled="saving" @click="reorder.cancel">{{ t('common.cancel') }}</Button>
      <Button size="sm" action :loading="saving" @click="reorder.save">
        {{ dirty ? t('common.save') : t('overview.layout.done') }}
      </Button>
    </div>
    <p v-if="enabled && !authEnabled" class="mb-4 text-sm text-gray-500">{{ t('overview.layout.local') }}</p>
    <p v-if="error" role="alert" class="mb-4 text-sm text-red-600">{{ error }}</p>
    <p v-if="message" role="status" class="mb-4 text-sm text-text-secondary">{{ message }}</p>
    <p v-if="!enabled && !loading && !renderedCards.length" class="mb-4 rounded-surface border border-dashed border-border p-6 text-sm text-text-secondary">{{ t('overview.layout.empty') }}</p>
    <OverviewLayoutEditor v-if="enabled" :inert="saving" :cards="editorCards" :hidden="hidden" :columns="columns"
      @hide="reorder.hide" @add="reorder.add" @move="(from, to) => reorder.nudge(from, to - from)" />
    <div v-if="loading" class="overview-grid" role="status" :aria-label="t('common.loading')" :style="{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }">
      <Skeleton v-for="index in 3" :key="index" class="h-48 w-full" />
    </div>
    <TransitionGroup v-show="!enabled" name="overview-sort" tag="div" class="overview-grid" :style="{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }" :aria-busy="loading || saving">
      <div v-for="card in loading ? [] : renderedCards" :key="card.id" class="overview-tile" :style="tileStyle(card.id)" :class="{ 'is-wide': card.id === WIDE_CARD }">
        <div :data-tile="card.id" class="overview-tile-surface">
          <component :is="card.component" class="overview-card quiet-scrollbar" />
        </div>
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
/* Vue measures keyed grid positions and slides each tile to its new slot.
   The inner surface owns lift, leaving the outer transform free for FLIP. */
.overview-sort-move {
  transition: transform 280ms cubic-bezier(0.22, 1, 0.36, 1);
}
/* Cards align at the top of each row and retain their natural height. Frame
   cards (header + .overview-card-body) scroll only their body, so the title
   and actions stay in view. One shared cap keeps a long subscription list
   from towering over its row. The full-width flow diagram is alone in its
   row and keeps a larger cap. */
.overview-grid {
  display: grid;
  gap: 1rem;
  align-items: start;
  isolation: isolate;
  --overview-card-max: 26rem;
}
.overview-tile { min-width: 0; }
.overview-tile.is-wide { --overview-card-max: 32rem; }
.overview-card {
  min-height: 0;
  max-height: var(--overview-card-max);
  overflow: auto;
}
.overview-card.overview-card-frame { overflow: hidden; }
.overview-tile-surface {
  min-width: 0;
  position: relative;
  border-radius: var(--radius-surface);
  transition: transform 180ms ease-out, opacity 180ms ease-out, box-shadow 180ms ease-out;
}
.overview-toolbar.is-arranging {
  position: sticky;
  top: 0.5rem;
  z-index: 10;
  padding: 0.75rem;
  border: 1px solid var(--glass-border-muted);
  border-radius: var(--radius-surface);
  background: var(--glass-bg-strong);
  box-shadow: var(--shadow-surface);
}
@media (prefers-reduced-motion: reduce) {
  .overview-sort-move { transition: none; }
}
</style>
