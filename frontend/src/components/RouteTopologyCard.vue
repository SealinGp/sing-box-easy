<script setup lang="ts">
import { RouterLink } from 'vue-router'
import { computed, nextTick, ref } from 'vue'
import ProxyInspector from './ProxyInspector.vue'
import { ArrowPathIcon, ArrowTopRightOnSquareIcon, ArrowsPointingInIcon, ArrowsPointingOutIcon, ExclamationTriangleIcon, SignalIcon, FunnelIcon, MinusIcon, PlusIcon, XMarkIcon } from '@heroicons/vue/24/outline'
import RouteFlowDiagram from './RouteFlowDiagram.vue'
import RouteTopologyLegend from './RouteTopologyLegend.vue'
import RouteTopologyExits from './RouteTopologyExits.vue'
import { Select } from '../volt'
import { FILTER_THRESHOLD } from '../utils/selectFilter'
import { useRouteTopologyCard } from '../composables/useRouteTopologyCard'
const {
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
} = useRouteTopologyCard()
const inspector = ref<InstanceType<typeof ProxyInspector>>()
const inspectedExit = ref('')
const inspectedOutbound = computed(() => config.value?.outbounds?.find(item => item.tag === inspectedExit.value))
async function inspectExit(name: string, anchor: HTMLElement) {
  inspectedExit.value = name
  await nextTick()
  inspector.value?.showAt(anchor)
}
</script>

<template>
  <div class="overview-card-frame bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <!-- Where the body went while it is full-window. -->
    <div v-if="expanded" class="flex items-center justify-between gap-3">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">{{ $t('routeFlow.title') }}</h3>
      <button
        type="button"
        class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
        @click="toggleExpanded"
      >
        <ArrowsPointingInIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.fullWindow.restore') }}
      </button>
    </div>

    <!--
      A constrained flex column while docked; a fixed, viewport-filling column
      while expanded. Same children in
      both cases — see `expanded` in the script for why it moves rather than
      re-renders.

      z-40, one step BELOW the app's overlay layer: this container teleports to
      <body>, and so do the select panels inside it. At an equal z-index the
      later sibling wins, which put the diagram over an open dropdown. Above
      the page (the top bar is z-30), under anything that overlays it.
    -->
    <Teleport to="body" :disabled="!expanded">
      <div
        :class="
          expanded
            ? 'topology-content fixed inset-0 z-40 flex flex-col bg-bg-secondary p-4 sm:p-6 overflow-hidden'
            : 'topology-content flex flex-col min-h-0'
        "
        :role="expanded ? 'dialog' : undefined"
        :aria-modal="expanded ? 'true' : undefined"
        :aria-label="expanded ? $t('routeFlow.title') : undefined"
      >
    <div class="flex items-start justify-between gap-3 mb-1">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">
        {{ $t('routeFlow.title') }}
      </h3>
      <div class="flex items-center gap-3 flex-shrink-0">
        <!--
          Live: light the expected drawing with real connections. Disabled
          while sing-box is stopped — the data comes from sing-box's own API,
          so there is nothing to show and the click would only fail.
        -->
        <button
          type="button"
          role="switch"
          :aria-checked="liveEnabled"
          class="inline-flex items-center gap-1.5 text-xs font-medium px-2 py-1 rounded-md border transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          :class="
            liveEnabled
              ? 'bg-primary-600 border-primary-600 text-white hover:bg-primary-700'
              : 'border-gray-300 dark:border-slate-600 text-gray-600 dark:text-gray-300 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-400'
          "
          :disabled="!running"
          :title="!running ? $t('routeFlow.live.needsRunning') : liveEnabled ? $t('routeFlow.live.on') : $t('routeFlow.live.off')"
          @click="toggleLive"
        >
          <span
            class="h-1.5 w-1.5 rounded-full"
            :class="liveEnabled ? 'bg-emerald-500' : 'bg-gray-400'"
          ></span>
          <SignalIcon class="h-3.5 w-3.5" />
          {{ $t('routeFlow.live.toggle') }}
        </button>
        <!--
          View chrome, grouped tight. Three clusters at gap-3 — the Live mode
          toggle, these two icon buttons, then the link away — is what stops the
          header reading as one undifferentiated row of six controls. Zoom used
          to sit here and now floats on the diagram, which is both where it acts
          and one fewer thing competing at this level.
        -->
        <div class="inline-flex items-center gap-1">
          <button
            type="button"
            class="inline-flex items-center p-1 rounded text-gray-500 hover:text-gray-700 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-slate-700 transition-colors disabled:opacity-50"
            :disabled="refreshing"
            :title="$t('routeFlow.refresh')"
            :aria-label="$t('routeFlow.refresh')"
            @click="load(true)"
          >
            <ArrowPathIcon class="h-4 w-4" :class="refreshing ? 'animate-spin' : ''" />
          </button>
          <button
            type="button"
            class="inline-flex items-center p-1 rounded text-gray-500 hover:text-gray-700 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-200 dark:hover:bg-slate-700 transition-colors"
            :title="expanded ? $t('routeFlow.fullWindow.exit') : $t('routeFlow.fullWindow.enter')"
            :aria-label="expanded ? $t('routeFlow.fullWindow.exit') : $t('routeFlow.fullWindow.enter')"
            :aria-pressed="expanded"
            @click="toggleExpanded"
          >
            <ArrowsPointingInIcon v-if="expanded" class="h-4 w-4" />
            <ArrowsPointingOutIcon v-else class="h-4 w-4" />
          </button>
        </div>
        <RouterLink
          :to="ROUTE_PAGE"
          class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
        >
          {{ $t('routeFlow.openRoute') }}
          <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
        </RouterLink>
      </div>
    </div>

    <p v-if="error" class="text-xs text-red-600 dark:text-red-400 mb-3">{{ error }}</p>

    <!--
      Live strip: what is flowing right now, and the three ways to narrow it.

      NO TINT. A full-width tinted bar sitting above content reads as a HEADER —
      a label for the diagram — when this row is the opposite of a label: values
      that change every second, plus the controls that shape them. Liveness is
      already said twice above (the Live toggle's own state, and the diagram
      lighting up), and spending a whole surface saying it a third time is what
      made the row look like chrome.

      Two kinds of thing, separated by WEIGHT rather than by a box. The readouts
      are data — a micro-label in the app's usual uppercase idiom, then a number
      with enough presence to be read at a glance while it ticks. The filters
      are controls and carry their own borders, which only read as borders
      against the card: behind a tint, a bordered input and its background
      flatten into each other.
    -->
    <div v-if="liveEnabled" class="flex flex-wrap items-center gap-x-5 gap-y-2 mb-3 text-xs">
      <template v-if="overlay">
        <!--
          Down carries the accent because it is the number the whole overlay is
          built on: ribbon width, pulse speed and the top-N ranking are all
          functions of it. Up is the same size but neutral — equal billing would
          claim the two drive the picture equally, and they do not.
        -->
        <span class="inline-flex items-baseline gap-1.5">
          <span class="font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ $t('routeFlow.live.down') }}
          </span>
          <span class="text-sm font-semibold tabular-nums transition-colors duration-200" :class="totalDownClass">
            {{ rate(overlay.totals.down) }}
          </span>
        </span>
        <span class="inline-flex items-baseline gap-1.5">
          <span class="font-semibold uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ $t('routeFlow.live.up') }}
          </span>
          <span class="text-sm font-semibold tabular-nums text-gray-700 dark:text-gray-200">
            {{ rate(overlay.totals.up) }}
          </span>
        </span>

        <!--
          Counts and caveats stay secondary: they qualify the two rates rather
          than competing with them, and at equal weight the eye is given five
          numbers to rank instead of two.
        -->
        <span class="tabular-nums text-gray-500 dark:text-gray-400">
          {{ $t('routeFlow.live.connections', { n: overlay.totals.connections }, overlay.totals.connections) }}
          <span v-if="overlay.totals.connections !== overlay.totals.all" class="text-gray-400 dark:text-gray-500">
            ({{ $t('routeFlow.live.shownOf', { shown: overlay.totals.connections, all: overlay.totals.all }) }})
          </span>
        </span>
        <span
          v-if="overlay.totals.connections > 0 && overlay.totals.down === 0"
          class="text-gray-400 dark:text-gray-500 italic"
        >
          {{ $t('routeFlow.live.idle') }}
        </span>
        <!--
          How many rows the floor is holding back. A diagram that went quiet has
          to say why, or a working stream reads as a broken one.
        -->
        <span v-if="overlay.belowFloor > 0" class="text-gray-400 dark:text-gray-500">
          {{ $t('routeFlow.live.belowFloor', { n: overlay.belowFloor }, overlay.belowFloor) }}
        </span>
      </template>
      <span v-else-if="connecting" class="text-gray-500 dark:text-gray-400 inline-flex items-center gap-1">
        <ArrowPathIcon class="h-3.5 w-3.5 animate-spin" />
        {{ $t('routeFlow.live.connecting') }}
      </span>
      <span v-if="liveError" class="text-red-600 dark:text-red-400 inline-flex items-center gap-1">
        <ExclamationTriangleIcon class="h-3.5 w-3.5" />
        {{ liveError }} · {{ $t('routeFlow.live.retrying') }}
      </span>

      <span class="flex-1"></span>

      <!--
        Busy only: fold the rules carrying nothing into bands, so a 28-rung
        ladder shortens to what is moving. Off by default — it is the one
        control here that changes the diagram's SHAPE, and the shape being
        identical with the overlay on and off is what makes expected and actual
        comparable.
      -->
      <button
        type="button"
        role="switch"
        :aria-checked="busyOnly"
        class="inline-flex items-center gap-1 px-2 py-1 rounded-md border text-xs font-medium transition-colors"
        :class="
          busyOnly
            ? 'bg-primary-600 border-primary-600 text-white hover:bg-primary-700'
            : 'border-gray-300 dark:border-slate-600 text-gray-600 dark:text-gray-300 hover:border-primary-400 hover:text-primary-600 dark:hover:text-primary-400'
        "
        :title="$t('routeFlow.live.busyOnlyHint')"
        @click="busyOnly = !busyOnly"
      >
        <FunnelIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.live.busyOnly') }}
        <span v-if="busyOnly && hiddenCount > 0" class="tabular-nums opacity-80">−{{ hiddenCount }}</span>
      </button>

      <!--
        Narrowing to one device or one site is how a slow-site complaint gets
        pinned. The device is PICKED, not typed: the frame already carries every
        client address in the snapshot, so a list beats recalling an IP — and a
        typo in a typed one is indistinguishable from a device with no traffic.
        A value picker, so `volt/Select`; the `value: ''` entry needs its label
        repeated as the placeholder so an empty value still has a label.

        The panel teleports to <body> like every other overlay in the app — see
        the full-window container's z-index for why that keeps working there,
        and why keeping the panel inside the card does not: `backdrop-filter`
        samples only what is painted BEHIND an element, so a panel inside the
        card had the diagram show straight through its glass.
      -->
      <label class="inline-flex items-center gap-1">
        <span class="sr-only">{{ $t('routeFlow.live.filterSource') }}</span>
        <Select
          v-model="sourceIpInput"
          :options="sourceOptions"
          optionValue="value"
          :filter="sourceOptions.length >= FILTER_THRESHOLD"
          :filterPlaceholder="$t('common.search')"
          :emptyFilterMessage="$t('common.noMatch')"
          :emptyMessage="$t('routeFlow.live.noSources')"
          :placeholder="$t('routeFlow.live.allSources')"
          :title="$t('routeFlow.live.filterSource')"
          class="w-56 text-xs"
          scrollHeight="14rem"
        >
          <template #value="{ value }">
            <span class="truncate">{{ sourceLabel(value ?? '') }}</span>
          </template>
          <!--
            The address never truncates; the busy-ness does. An address clipped
            to "192.168.31.1…" is unusable — it is the whole identity of the
            row — while a clipped rate still reads as "this one is busy".
          -->
          <template #option="{ option }">
            <span class="shrink-0">{{ sourceLabel(option.value) }}</span>
            <span v-if="option.connections" class="ml-auto min-w-0 truncate pl-2 tabular-nums text-[10px] text-gray-500 dark:text-gray-400">
              {{ $t('routeFlow.live.sourceConnections', { n: option.connections }) }}
              <template v-if="option.down">· {{ rate(option.down) }}</template>
            </span>
          </template>
        </Select>
      </label>
      <!-- Sized to match the source picker beside it, which is a control-height field. -->
      <label class="inline-flex items-center gap-1">
        <span class="sr-only">{{ $t('routeFlow.live.filterHost') }}</span>
        <input
          v-model="hostInput"
          type="text"
          :placeholder="$t('routeFlow.live.filterHost')"
          class="w-40 px-3 py-2 rounded-control border border-gray-300 dark:border-slate-600 bg-white dark:bg-slate-900 text-xs text-gray-800 dark:text-gray-100 placeholder:text-gray-400 focus:outline-none focus:border-primary-500"
        />
      </label>
      <button
        v-if="hasFilter"
        type="button"
        class="inline-flex items-center gap-0.5 text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-100"
        :title="$t('routeFlow.live.clearFilter')"
        @click="clearFilter"
      >
        <XMarkIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.live.clearFilter') }}
      </button>
    </div>

    <!-- Traffic the running rule list cannot place on a row. -->
    <p
      v-if="overlay && overlay.unmatched.length > 0"
      class="text-xs text-amber-700 dark:text-amber-300 mb-3 inline-flex items-center gap-1"
    >
      <ExclamationTriangleIcon class="h-3.5 w-3.5" />
      {{ $t('routeFlow.live.unmatched', { n: unmatchedConnections }, unmatchedConnections) }}
    </p>

    <!-- Config-level problems `sing-box check` does not report. -->
    <div v-if="!loading && (unreachableCount || missingCount)" class="flex flex-wrap gap-2 mb-3">
      <span
        v-if="missingCount"
        class="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md bg-red-50 text-red-700 dark:bg-red-950/40 dark:text-red-300"
      >
        <ExclamationTriangleIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.warnMissing', { n: missingCount }, missingCount) }}
      </span>
      <span
        v-if="unreachableCount"
        class="inline-flex items-center gap-1 text-xs px-2 py-1 rounded-md bg-amber-50 text-amber-800 dark:bg-amber-950/40 dark:text-amber-300"
      >
        <ExclamationTriangleIcon class="h-3.5 w-3.5" />
        {{ $t('routeFlow.warnUnreachable', { n: unreachableCount }, unreachableCount) }}
      </span>
    </div>

    <div v-if="loading" class="h-40 flex items-center justify-center">
      <ArrowPathIcon class="h-5 w-5 animate-spin text-gray-400" />
    </div>

    <p v-else-if="!hasSomethingToDraw" class="text-sm text-gray-500 dark:text-gray-400 py-6 text-center">
      {{ config ? $t('routeFlow.empty') : $t('routeFlow.noConfig') }}
    </p>

    <template v-else>
      <RouteTopologyExits :exits="topology.exits" @inspect="inspectExit" />
      <!--
        Docked, a tall ladder scrolls rather than shrinking: this repo's own
        config has 28 rules, and scaling that into a card makes every label
        unreadable. Full-window, the wrapper takes the remaining height and the
        diagram fits it — the whole ladder at once is the point of the mode.
      -->
      <div class="topology-canvas relative flex min-h-0 flex-col" :class="expanded ? 'flex-1' : ''">
        <div
          :ref="zoom.bindViewport"
          tabindex="0"
          :class="[
            'quiet-scrollbar min-h-0 overflow-auto -mx-1 px-1',
            expanded ? 'h-full' : '',
            'focus:outline-none focus-visible:ring-1 focus-visible:ring-primary-500 rounded',
            // The gesture is otherwise invisible: the cursor is the only thing
            // that says the held modifier turned the diagram into a canvas.
            zoom.panning.value ? 'cursor-grabbing' : zoom.panReady.value ? 'cursor-grab' : '',
          ]"
        >
          <RouteFlowDiagram interactive-exits
            :topology="topology"
            :live="liveEnabled ? overlay : null"
            :fit="expanded"
            :scale="zoom.scale.value"
            :hidden-rule-indices="hiddenRuleIndices"
            @expand-band="expandBand"
            @inspect-exit="inspectExit"
          />
        </div>

        <!--
          Zoom floats ON the diagram, the way every map and design tool puts it,
          rather than in the card header. Two reasons beyond decluttering: it is
          adjacent to the thing it scales, and it stays put while the content
          under it scrolls, so it is reachable at any pan position.

          Absolutely positioned against this wrapper, NOT inside the scrolling
          child — inside, it would scroll away with the diagram and be gone the
          moment it was needed. Nudged clear of the corner so it does not sit on
          top of both scrollbars at once.
        -->
        <div
          class="absolute bottom-3 right-3 inline-flex items-center rounded-md border border-border bg-bg-secondary shadow-surface overflow-hidden"
          :title="$t('routeFlow.zoom.hint')"
        >
          <button
            type="button"
            class="px-1.5 py-1 text-gray-500 hover:text-gray-800 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-100 dark:hover:bg-slate-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            :disabled="!zoom.canZoomOut.value"
            :title="$t('routeFlow.zoom.out')"
            :aria-label="$t('routeFlow.zoom.out')"
            @click="zoom.zoomOut()"
          >
            <MinusIcon class="h-3.5 w-3.5" />
          </button>
          <button
            type="button"
            class="px-1.5 py-1 text-xs tabular-nums text-gray-600 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-slate-700 transition-colors min-w-[3rem]"
            :title="$t('routeFlow.zoom.reset')"
            @click="zoom.reset()"
          >
            {{ zoom.percent.value === null ? $t('routeFlow.zoom.fit') : `${zoom.percent.value}%` }}
          </button>
          <button
            type="button"
            class="px-1.5 py-1 text-gray-500 hover:text-gray-800 hover:bg-gray-100 dark:text-gray-400 dark:hover:text-gray-100 dark:hover:bg-slate-700 transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
            :disabled="!zoom.canZoomIn.value"
            :title="$t('routeFlow.zoom.in')"
            :aria-label="$t('routeFlow.zoom.in')"
            @click="zoom.zoomIn()"
          >
            <PlusIcon class="h-3.5 w-3.5" />
          </button>
        </div>
      </div>

      <RouteTopologyLegend :legend-open="legendOpen" :expanded="expanded" :live-enabled="liveEnabled"
        :heat-scale="heatScale" :floor-label="floorLabel" @toggle="toggleLegend" />
    </template>
      </div>
    </Teleport>
    <ProxyInspector ref="inspector" :name="inspectedExit" :outbound="inspectedOutbound" external />
  </div>
</template>

<style scoped>
.topology-content > :not(.topology-canvas) { flex-shrink: 0; }
/* This wrapper must shrink with the capped card, leaving one canvas scrollport. */
.overview-card-frame > .topology-content { flex-shrink: 1; }
</style>
