<script setup lang="ts">
import { ChevronRightIcon } from '@heroicons/vue/24/outline'
import { RANK_N, TOP_N } from '../utils/flowOverlay'
defineProps<{ legendOpen: boolean; expanded: boolean; liveEnabled: boolean; heatScale: { threshold: string; swatch: string }[]; floorLabel: string }>()
const emit = defineEmits<{ toggle: [] }>()
</script>
<template>
      <!-- Names the three columns, and says what the middle one's order means. -->
      <div
        class="flex items-center justify-between gap-2 mt-2 pt-2 text-[11px] text-gray-500 dark:text-gray-400 border-t border-gray-100 dark:border-slate-700"
      >
        <span>{{ $t('routeFlow.columns.inbounds') }}</span>
        <span class="text-center">{{ $t('routeFlow.columns.rules') }}</span>
        <span>{{ $t('routeFlow.columns.outbounds') }}</span>
      </div>

      <!--
        The key, on demand.

        This is reference material: a new install needs it two or three times,
        and after that it is a permanent five-item footer explaining symbols the
        operator has long since learned. So it behaves like a manual — shelved
        by default once shelved, one muted line away when a symbol IS unfamiliar
        — and the choice persists, because re-collapsing it on every visit is
        the same nuisance in slower motion.

        The column strip above is deliberately NOT in here: it carries the claim
        that the ladder is ordered and stops at the first match, which is the
        diagram's whole thesis rather than a symbol key.
      -->
      <div class="flex items-center justify-between gap-2 mt-2">
        <button
          type="button"
          class="inline-flex items-center gap-1 text-[11px] text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 transition-colors"
          :aria-expanded="legendOpen"
          @click="emit('toggle')"
        >
          <ChevronRightIcon class="h-3 w-3 transition-transform" :class="legendOpen ? 'rotate-90' : ''" />
          {{ $t('routeFlow.legend.title') }}
        </button>
        <span v-if="expanded" class="text-[11px] text-gray-400">{{ $t('routeFlow.fullWindow.escHint') }}</span>
      </div>

      <!-- What the non-obvious strokes mean. -->
      <div v-if="legendOpen" class="flex flex-wrap gap-x-4 gap-y-1 mt-2 text-[11px] text-gray-500 dark:text-gray-400">
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
        <template v-if="liveEnabled">
          <!--
            The colour scale, with its thresholds. A scale nobody can read the
            steps of is decoration; each swatch says the rate it starts at.
          -->
          <span class="inline-flex items-center gap-1.5">
            <span class="inline-flex items-center gap-1">
              <span
                v-for="step in heatScale"
                :key="step.threshold"
                class="inline-flex items-center gap-0.5"
              >
                <span class="h-2 w-2 rounded-sm" :class="step.swatch"></span>
                <span class="tabular-nums">{{ step.threshold }}</span>
              </span>
            </span>
            {{ $t('routeFlow.live.legendHeat') }}
          </span>
          <span class="inline-flex items-center gap-1.5">
            <!-- A lit ribbon with a comet on it — what the top N look like. -->
            <svg width="22" height="8" aria-hidden="true">
              <line x1="1" y1="4" x2="21" y2="4" stroke-width="2" stroke-linecap="round" class="stroke-primary-400 dark:stroke-primary-600 opacity-80" />
              <line x1="7" y1="4" x2="15" y2="4" stroke-width="2" stroke-linecap="round" class="stroke-primary-600 dark:stroke-primary-300 opacity-40" />
              <line x1="13" y1="4" x2="16" y2="4" stroke-width="3" stroke-linecap="round" class="stroke-primary-600 dark:stroke-primary-300" />
            </svg>
            {{ $t('routeFlow.live.legendMoving', { n: TOP_N }) }}
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span
              class="inline-flex h-3.5 w-3.5 items-center justify-center rounded-full bg-primary-600 dark:bg-primary-400 text-[8px] font-bold text-white dark:text-slate-900"
              >1</span
            >
            {{ $t('routeFlow.live.legendRank', { n: RANK_N }) }}
          </span>
          <span class="inline-flex items-center gap-1.5">
            <svg width="22" height="8" aria-hidden="true">
              <line x1="0" y1="4" x2="22" y2="4" stroke-width="2.5" class="stroke-primary-400 dark:stroke-primary-600" />
            </svg>
            {{ $t('routeFlow.live.legendLit', { rate: floorLabel }) }}
          </span>
        </template>
        <span class="ml-auto text-gray-400">{{ $t('routeFlow.zoom.hint') }}</span>
      </div>
</template>
