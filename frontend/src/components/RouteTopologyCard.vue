<script setup lang="ts">
/**
 * Overview card: the config's traffic flow, drawn.
 *
 * The Route page lists `route.rules` as rows, which is the right shape for
 * editing one and the wrong shape for answering "what does this config
 * actually do". Two facts are invisible in a list and obvious in a picture:
 * that many rules share one outbound, and that the list is an ordered ladder
 * whose first match wins. Both are what someone is checking right after an edit.
 *
 * It reads `GET /config` directly rather than the three narrower endpoints —
 * the diagram needs `inbounds`, `outbounds`, `endpoints`, `route.rules` and
 * `route.final` to be from the SAME read, or a node can appear missing simply
 * because two requests straddled a config write.
 */
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import {
  ArrowPathIcon,
  ArrowTopRightOnSquareIcon,
  ExclamationTriangleIcon,
} from '@heroicons/vue/24/outline'
import RouteFlowDiagram from './RouteFlowDiagram.vue'
import { configService } from '../services'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { buildRouteTopology } from '../utils/routeTopology'
import type { SingBoxConfig } from '../types/api'

const { t } = useI18n()

const ROUTE_PAGE = '/dashboard/route'

const config = ref<SingBoxConfig | null>(null)
/** First paint only — a manual refresh keeps the diagram on screen. */
const loading = ref(true)
const refreshing = ref(false)
const error = ref('')

const topology = computed(() => buildRouteTopology(config.value))

const hasSomethingToDraw = computed(
  () => topology.value.rules.length > 0 || topology.value.inbounds.length > 0,
)

/**
 * Two diagnostics worth pulling out of the picture and into words.
 *
 * A missing outbound is only reported by sing-box at START, and an unreachable
 * rule is never reported at all — `sing-box check` passes on both. Someone
 * scanning the card should not have to spot a red node to learn about them.
 */
const unreachableCount = computed(
  () => topology.value.rules.filter((rule) => !rule.reachable).length,
)

const missingCount = computed(() => {
  const missing = new Set(
    topology.value.exits.filter((exit) => exit.kind === 'missing').map((exit) => exit.id),
  )
  return topology.value.rules.filter((rule) => rule.exitId && missing.has(rule.exitId)).length
})

const load = async (manual = false) => {
  if (manual) refreshing.value = true
  try {
    const response = await configService.getConfig()
    config.value = response.data ?? null
    error.value = ''
  } catch (err) {
    console.error('Failed to load config for the route topology card:', err)
    // Keep the last good drawing rather than blanking it — a failed re-read
    // says nothing about whether the config it drew is still correct.
    error.value = apiErrorMessage(err, t('routeFlow.loadFailed'))
  } finally {
    loading.value = false
    refreshing.value = false
  }
}

onMounted(() => load())
</script>

<template>
  <div class="bg-white dark:bg-slate-800 p-4 rounded-surface shadow-surface">
    <div class="flex items-start justify-between gap-3 mb-1">
      <h3 class="text-lg font-semibold text-gray-700 dark:text-gray-300">
        {{ $t('routeFlow.title') }}
      </h3>
      <div class="flex items-center gap-3 flex-shrink-0">
        <button
          type="button"
          class="inline-flex items-center gap-1 text-xs font-medium text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 transition-colors disabled:opacity-50"
          :disabled="refreshing"
          :title="$t('routeFlow.refresh')"
          :aria-label="$t('routeFlow.refresh')"
          @click="load(true)"
        >
          <ArrowPathIcon class="h-4 w-4" :class="refreshing ? 'animate-spin' : ''" />
        </button>
        <RouterLink
          :to="ROUTE_PAGE"
          class="inline-flex items-center gap-1 text-xs font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300 transition-colors"
        >
          {{ $t('routeFlow.openRoute') }}
          <ArrowTopRightOnSquareIcon class="h-3.5 w-3.5" />
        </RouterLink>
      </div>
    </div>

    <p class="text-xs text-gray-500 dark:text-gray-400 mb-3">
      {{ $t('routeFlow.desc') }}
    </p>

    <p v-if="error" class="text-xs text-red-600 dark:text-red-400 mb-3">{{ error }}</p>

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
      <!--
        A tall ladder scrolls rather than shrinking: this repo's own config has
        28 rules, and scaling that to fit a card makes every label unreadable.
      -->
      <div class="overflow-auto max-h-[34rem] -mx-1 px-1">
        <RouteFlowDiagram :topology="topology" />
      </div>

      <!-- Names the three columns, and says what the middle one's order means. -->
      <div
        class="flex items-center justify-between gap-2 mt-2 pt-2 text-[11px] text-gray-500 dark:text-gray-400 border-t border-gray-100 dark:border-slate-700"
      >
        <span>{{ $t('routeFlow.columns.inbounds') }}</span>
        <span class="text-center">{{ $t('routeFlow.columns.rules') }}</span>
        <span>{{ $t('routeFlow.columns.outbounds') }}</span>
      </div>

      <!-- What the non-obvious strokes mean. -->
      <div class="flex flex-wrap gap-x-4 gap-y-1 mt-2 text-[11px] text-gray-500 dark:text-gray-400">
        <span class="inline-flex items-center gap-1.5">
          <svg width="18" height="8" aria-hidden="true">
            <line x1="0" y1="4" x2="18" y2="4" stroke-width="1.5" stroke-dasharray="4 3" class="stroke-gray-400 dark:stroke-slate-500" />
          </svg>
          {{ $t('routeFlow.legend.passthrough') }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <span class="h-2 w-2 rounded-full bg-amber-400"></span>
          {{ $t('routeFlow.legend.scoped') }}
        </span>
        <span class="inline-flex items-center gap-1.5">
          <span
            class="px-1 rounded-full bg-primary-100 dark:bg-primary-900/60 text-primary-700 dark:text-primary-300 font-semibold tabular-nums"
            >3</span
          >
          {{ $t('routeFlow.legend.converge', { n: 3 }) }}
        </span>
      </div>
    </template>
  </div>
</template>
