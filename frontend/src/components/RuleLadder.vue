<script setup lang="ts">
/**
 * The rule ladder, with a lamp per rung.
 *
 * sing-box evaluates both `dns.rules` and `route.rules` top-down and stops at
 * the first match, so the config is a decision ladder rather than a graph.
 * Rendering it as one needs no graph library, which matters for the OpenWrt
 * builds where the whole binary has to fit in the router's overlay.
 *
 * FOUR STATES, NOT TWO
 * ────────────────────
 * The DNS diagram used to highlight the winning rung and nothing else, so a rule
 * the query was tested against and failed looked exactly like a rule that was
 * never reached. Those are opposite facts. The lamps separate them:
 *
 *   matched      the rule fired
 *   not_matched  tested, did not fire
 *   unevaluated  could not be decided offline — it MIGHT have fired first,
 *                which is why the verdict below it is only a guess
 *   skipped      never reached; the walk had already stopped
 *
 * The `skipped` lamp is also the unlit lamp of a rung the walk has not got to
 * yet — deliberately, so watching the sequence teaches what the dark rungs at
 * the bottom mean.
 */
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { PlayIcon } from '@heroicons/vue/24/outline'
import { useRuleSequencer } from '../composables/useRuleSequencer'
import type { LadderRung, LadderState } from '../types/ruleLadder'

const props = withDefaults(
  defineProps<{
    rungs: readonly LadderRung[]
    /** The deciding rung, or -1 when the query falls through to `final`. */
    matchedIndex: number
    /** Rungs hidden by a caller-side compact filter, so the ladder can say so. */
    hiddenCount?: number
    /**
     * Changes once per probe run. Restarts the walk even when the new result
     * has the same shape as the last — see useRuleSequencer.
     */
    runToken?: number | string
  }>(),
  { hiddenCount: 0, runToken: undefined },
)

const { t } = useI18n()

const count = computed(() => props.rungs.length)
const matched = computed(() => props.matchedIndex)
const runToken = computed(() => props.runToken)

const sequencer = useRuleSequencer(count, matched, { runToken })

/**
 * The lamp, and the rung's frame.
 *
 * An unlit rung is dimmed rather than hidden: the rule is still part of the
 * config and its conditions are still worth reading. Only the VERDICT waits.
 */
const LAMP: Record<LadderState, string> = {
  matched: 'bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.20)]',
  not_matched: 'bg-gray-400 dark:bg-gray-600',
  unevaluated: 'bg-amber-400 shadow-[0_0_0_3px_rgba(251,191,36,0.20)]',
  skipped: 'bg-gray-200 dark:bg-gray-700',
}

const FRAME: Record<LadderState, string> = {
  matched: 'border-emerald-500 bg-emerald-50/60 dark:bg-emerald-950/30',
  not_matched: 'border-gray-200 dark:border-gray-700',
  unevaluated: 'border-amber-400 bg-amber-50/60 dark:bg-amber-950/20',
  skipped: 'border-gray-200 dark:border-gray-700',
}

/** An unlit rung has no verdict yet, so it borrows the never-reached look. */
const lampClass = (rung: LadderRung) =>
  sequencer.isRevealed(rung.index) ? LAMP[rung.state] : LAMP.skipped

const frameClass = (rung: LadderRung) =>
  sequencer.isRevealed(rung.index) ? FRAME[rung.state] : `${FRAME.skipped} opacity-45`

const stateLabel = (state: LadderState) => t(`rule.ladder.state.${state}`)
</script>

<template>
  <div class="space-y-1.5">
    <!-- The walk is a replay of a settled result, so it says so and offers the
         way out. Both are one line; neither is worth a dialog. -->
    <div v-if="rungs.length" class="flex items-center justify-between gap-2">
      <p class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wide">
        {{ $t('rule.ladder.title') }}
      </p>
      <button
        v-if="sequencer.running.value"
        type="button"
        class="text-xs text-primary-600 dark:text-primary-400 hover:underline"
        @click="sequencer.skip()"
      >
        {{ $t('rule.ladder.skip') }}
      </button>
      <button
        v-else
        type="button"
        class="inline-flex items-center gap-1 text-xs text-gray-400 dark:text-gray-500 hover:text-primary-600 dark:hover:text-primary-400"
        @click="sequencer.replay()"
      >
        <PlayIcon class="h-3 w-3" />
        {{ $t('rule.ladder.replay') }}
      </button>
    </div>

    <div
      v-for="rung in rungs"
      :key="rung.index"
      class="rounded-control border px-3 py-2 transition-all duration-200"
      :class="[frameClass(rung), rung.deciding && sequencer.isRevealed(rung.index) ? 'ring-2 ring-emerald-500/40' : '']"
    >
      <div class="flex items-start gap-2">
        <!-- The lamp. Fixed width so the ladder reads as a column of indicators
             rather than as text that happens to be coloured. -->
        <span
          class="mt-1.5 h-2.5 w-2.5 shrink-0 rounded-pill transition-all duration-200"
          :class="lampClass(rung)"
          aria-hidden="true"
        ></span>
        <span class="mt-0.5 w-6 shrink-0 text-xs font-mono text-gray-400">{{ rung.index }}</span>

        <div class="min-w-0 flex-1">
          <!-- Never gated on the animation: the operator who needs the answer
               now can read it now. -->
          <p class="text-xs font-mono text-gray-700 dark:text-gray-300 break-words">
            {{ rung.summary }}
          </p>
          <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
            &rarr; {{ rung.outcome }}
            <!-- A matched rule that does not decide is the single most confusing
                 thing in the ladder, so it is stated rather than inferred. -->
            <span v-if="rung.continues && sequencer.isRevealed(rung.index)" class="italic">
              · {{ $t('rule.ladder.continues') }}
            </span>
          </p>
          <p
            v-if="rung.unevaluated?.length && sequencer.isRevealed(rung.index)"
            class="mt-0.5 text-xs text-amber-700 dark:text-amber-300"
          >
            {{ $t('rule.ladder.couldNotEvaluate', { fields: rung.unevaluated.join(', ') }) }}
          </p>
          <slot name="extra" :rung="rung" />
        </div>

        <span
          v-if="sequencer.isRevealed(rung.index)"
          class="shrink-0 rounded-pill bg-white/70 dark:bg-black/25 px-2 py-0.5 text-xs"
        >
          {{ stateLabel(rung.state) }}
        </span>
      </div>
    </div>

    <!-- Never let a compact view read as "these are all the rules". -->
    <p v-if="hiddenCount > 0" class="text-xs text-gray-400 dark:text-gray-500">
      {{ $t('rule.ladder.hidden', { count: hiddenCount }) }}
    </p>

    <!-- Fallthrough. Lights only once every rung has failed, so the answer
         never appears before the reason for it. -->
    <slot name="fallthrough" :revealed="sequencer.fallthroughRevealed.value" />
  </div>
</template>
