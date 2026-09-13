<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { onBeforeRouteLeave } from 'vue-router'
import { Bars3Icon } from '@heroicons/vue/24/outline'
import Button from '../../components/Button.vue'
import { userService } from '../../services'
import { useDeployment } from '../../composables/useDeployment'
import { useDragReorder } from '../../composables/useDragReorder'
import { useNotify } from '../../composables/useNotify'
import ServiceStatusCard from '../../components/ServiceStatusCard.vue'
import SubscriptionsOverviewCard from '../../components/SubscriptionsOverviewCard.vue'
import DnsProbeCard from '../../components/DnsProbeCard.vue'
import RouteProbeCard from '../../components/RouteProbeCard.vue'
import ApiEndpointsCard from '../../components/ApiEndpointsCard.vue'
import RouteTopologyCard from '../../components/RouteTopologyCard.vue'

const cards = [
  { id: 'route-topology', component: RouteTopologyCard, title: 'routeFlow.title' },
  { id: 'service-status', component: ServiceStatusCard, title: 'overview.serviceStatus' },
  { id: 'subscriptions', component: SubscriptionsOverviewCard, title: 'overview.subscriptions.title' },
  { id: 'dns-probe', component: DnsProbeCard, title: 'dnsProbe.title' },
  { id: 'route-probe', component: RouteProbeCard, title: 'routeProbe.title' },
  { id: 'api-endpoints', component: ApiEndpointsCard, title: 'overview.apis.title' },
]
const defaults = cards.map(card => card.id)
const order = ref([...defaults])
const renderedCards = computed(() => order.value.map(id => cards.find(card => card.id === id)!))
const { t } = useI18n()
const { authEnabled } = useDeployment()
const notify = useNotify()
const loading = ref(true)
const localKey = 'sbe-overview-order:anonymous'
let savedOrder = [...defaults]

function normalize(value: unknown): string[] {
  const ids = Array.isArray(value) ? value.filter(id => typeof id === 'string' && defaults.includes(id)) : []
  return [...new Set([...ids, ...defaults])]
}

async function load() {
  loading.value = true
  try {
    const value = authEnabled.value
      ? (await userService.getPreferences()).overview_order
      : JSON.parse(localStorage.getItem(localKey) ?? '[]')
    order.value = normalize(value)
    savedOrder = [...order.value]
  } catch {
    notify.error(t('overview.layout.loadError'))
  } finally {
    loading.value = false
  }
}

const reorder = useDragReorder(order, async () => {
  try {
    if (authEnabled.value) {
      const preferences = await userService.updatePreferences({ overview_order: [...order.value] })
      order.value = normalize(preferences.overview_order)
    } else {
      localStorage.setItem(localKey, JSON.stringify(order.value))
    }
    savedOrder = [...order.value]
    notify.success(t('overview.layout.saved'))
  } catch {
    order.value = [...savedOrder]
    notify.error(t('overview.layout.saveError'))
  }
})
reorder.syncKeys(defaults.length)
const { dirty, enabled, holdingIndex, saving } = reorder

function reset() {
  defaults.forEach((id, target) => {
    const from = order.value.indexOf(id)
    reorder.nudge(from, target - from)
  })
}

// Leaving the overview discards its unsaved arrangement; saved layouts reload
// from the current account when this route is visited again.
onBeforeRouteLeave(() => { if (enabled.value) reorder.cancel() })
onMounted(load)
</script>

<template>
  <div class="page-shell">
    <div v-if="enabled" class="overview-toolbar is-arranging mb-4 flex flex-wrap items-center gap-2" aria-live="polite">
      <div class="mr-auto min-w-0">
        <p class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('overview.layout.arranging') }}</p>
        <p class="hidden text-xs text-gray-500 dark:text-gray-400 sm:block">{{ t('overview.layout.hint') }}</p>
      </div>
      <Button variant="ghost" size="sm" action @click="reset">{{ t('overview.layout.reset') }}</Button>
      <Button variant="secondary" size="sm" action @click="reorder.cancel">{{ t('common.cancel') }}</Button>
      <Button size="sm" action @click="reorder.save">
        {{ dirty ? t('common.save') : t('overview.layout.done') }}
      </Button>
    </div>
    <p v-if="enabled && !authEnabled" class="mb-4 text-sm text-gray-500">{{ t('overview.layout.local') }}</p>
    <TransitionGroup name="overview-sort" tag="div" class="overview-grid grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 items-start" :aria-busy="loading || saving" :role="enabled ? 'list' : undefined">
      <div v-for="(card, index) in renderedCards" :key="card.id" class="overview-tile"
        v-bind="reorder.rowAttrs(index)"
        :role="enabled ? 'listitem' : undefined"
        :class="[card.id === 'route-topology' ? 'md:col-span-2 lg:col-span-3' : '', enabled ? 'is-arranging' : '']">
        <div
          class="overview-tile-surface"
          v-bind="enabled ? reorder.surfaceAttrs(index) : reorder.activationAttrs(index)"
          :class="{ 'is-holding': !enabled && holdingIndex === index }"
          :title="enabled ? t('overview.layout.handle', { name: t(card.title) }) : undefined"
        >
          <div v-if="enabled" class="overview-arrange-chrome">
            <span class="overview-arrange-label min-w-0">
              <Bars3Icon class="overview-arrange-grip h-4 w-4 flex-shrink-0" aria-hidden="true" />
              <span class="truncate">{{ t(card.title) }}</span>
            </span>
            <div class="flex items-center gap-1" data-reorder-control>
              <Button variant="ghost" size="sm" action :disabled="index === 0" :aria-label="t('overview.layout.earlier', { name: t(card.title) })" @click="reorder.nudge(index, -1)">↑</Button>
              <Button variant="ghost" size="sm" action :disabled="index === order.length - 1" :aria-label="t('overview.layout.later', { name: t(card.title) })" @click="reorder.nudge(index, 1)">↓</Button>
            </div>
          </div>
          <component :is="card.component" class="overview-card quiet-scrollbar" :inert="enabled" />
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
.overview-grid { isolation: isolate; }
.overview-tile { min-width: 0; }
.overview-card {
  max-height: 32rem;
  overflow: auto;
}
.overview-card.overview-card-frame { overflow: hidden; }
.overview-tile-surface {
  position: relative;
  border-radius: var(--radius-surface);
  transition: transform 180ms ease-out, opacity 180ms ease-out, box-shadow 180ms ease-out;
}
.overview-tile-surface.is-holding {
  transform: scale(0.992);
  outline: 2px solid color-mix(in srgb, var(--color-primary) 40%, transparent);
  box-shadow: var(--shadow-float), var(--glass-highlight);
}
.overview-tile.is-arranging > .overview-tile-surface {
  cursor: grab;
  outline: 1px solid color-mix(in srgb, var(--color-primary) 30%, transparent);
}
.overview-tile.is-arranging > .overview-tile-surface:active { cursor: grabbing; }
.overview-tile.is-arranging > .overview-tile-surface::after {
  position: absolute;
  z-index: 1;
  inset: 0;
  border-radius: inherit;
  background: color-mix(in srgb, var(--color-primary) 7%, transparent);
  content: '';
  pointer-events: none;
}
.overview-tile.is-arranging .overview-card {
  pointer-events: none;
  user-select: none;
  opacity: 0.62;
  filter: saturate(0.72);
}
.overview-tile.is-dragging { z-index: 1; }
.overview-tile.is-dragging > .overview-tile-surface {
  transform: scale(0.985);
  opacity: 0.65;
  box-shadow: var(--shadow-float-lg), var(--glass-highlight);
  outline-color: var(--color-primary);
}
.overview-arrange-chrome {
  position: absolute;
  z-index: 2;
  top: 0.5rem;
  right: 0.5rem;
  left: 0.5rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.375rem 0.5rem;
  border: 1px solid color-mix(in srgb, var(--color-primary) 24%, var(--glass-border-muted));
  border-radius: var(--radius-control);
  background: var(--glass-bg-strong);
  box-shadow: var(--shadow-float), var(--glass-highlight);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
  animation: overview-chrome-ready 220ms ease-out both;
}
.overview-arrange-label {
  display: inline-flex;
  flex: 1;
  align-items: center;
  gap: 0.375rem;
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text-primary);
}
.overview-arrange-grip {
  color: var(--color-primary);
  animation: overview-handle-ready 420ms ease-in-out 2;
}
.overview-toolbar.is-arranging {
  position: sticky;
  top: 0.5rem;
  z-index: 20;
  padding: 0.5rem;
  border: 1px solid var(--glass-border-muted);
  border-radius: var(--radius-surface);
  background: var(--glass-bg-strong);
  box-shadow: var(--shadow-float), var(--glass-highlight);
  backdrop-filter: var(--glass-blur);
  -webkit-backdrop-filter: var(--glass-blur);
}
@keyframes overview-chrome-ready {
  from { opacity: 0; transform: scale(0.98); }
  to { opacity: 1; transform: scale(1); }
}
@keyframes overview-handle-ready {
  0%, 100% { transform: rotate(0); }
  25% { transform: rotate(-8deg); }
  75% { transform: rotate(8deg); }
}
@media (prefers-reduced-motion: reduce) {
  .overview-sort-move, .overview-tile-surface { transition: none; }
  .overview-arrange-chrome, .overview-arrange-grip { animation: none; }
  .overview-tile.is-dragging > .overview-tile-surface { transform: none; }
}
</style>
