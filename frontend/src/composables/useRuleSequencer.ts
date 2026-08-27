/**
 * Walks a rule ladder one rung at a time, the way a POST code walks a
 * motherboard: each rung lights with its verdict, top to bottom, and the walk
 * stops dead at the first match.
 *
 * THIS IS A REPLAY, AND IT IS LABELLED AS ONE
 * ───────────────────────────────────────────
 * The verdicts are not arriving over time. `dnsprobe.Attribute` and
 * `routeprobe.Run` evaluate the whole ladder in one synchronous pass measured in
 * microseconds — by the time the panel has a result, every rung's answer is
 * already known. Pacing them out is a reading aid, not a measurement.
 *
 * Streaming them per-rung from the server was considered and rejected: it would
 * require the server to sleep between rules, which would make a diagnostic tool
 * misreport its own timing. The phases around the ladder DO have real latency
 * (a live query over the Clash API, a 250ms log settle, a fan-out across every
 * configured resolver) and those are streamed for real — see useProbeStream.
 *
 * Two consequences follow, and both are load-bearing:
 *
 *   1. THE ANIMATION NEVER GATES CONTENT. Every rung's summary and action render
 *      immediately. Only the lamp and the verdict badge are sequenced. An
 *      operator who needs the answer now can read it now.
 *   2. `prefers-reduced-motion` SKIPS TO THE END STATE, and so does `skip()`.
 *      The end state is not "everything lit" — it is "lit down to the match",
 *      because the rungs below it were never reached and their unlit look IS
 *      that fact.
 */
import { computed, onUnmounted, ref, watch, type Ref } from 'vue'

/** Per-rung dwell. Long enough to read as sequential, short enough not to wait. */
const DWELL_MS = 110

/**
 * The whole walk is capped rather than the step: a 27-rule config at 110ms is
 * three seconds of watching, which stops being an aid and becomes an obstacle.
 */
const MAX_TOTAL_MS = 1600

function prefersReducedMotion(): boolean {
  if (typeof window === 'undefined' || !window.matchMedia) return false
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

export interface RuleSequencerOptions {
  /** Skip the walk and render the end state immediately. */
  instant?: Ref<boolean>
  /**
   * Changes once per run, to restart a walk whose shape did not change.
   *
   * Without it, re-probing is silent whenever the new result happens to have
   * the same rung count and the same deciding rung — which is not a corner
   * case: it is what "I fixed the rule, let me check again" looks like when the
   * fix did NOT work. The walk would sit finished from the previous run and the
   * operator would read a stale ladder as a fresh answer.
   */
  runToken?: Ref<number | string | undefined>
}

/**
 * @param count       how many rungs the ladder has
 * @param matchedIndex the deciding rung, or -1 when the query falls through
 */
export function useRuleSequencer(
  count: Ref<number>,
  matchedIndex: Ref<number>,
  options: RuleSequencerOptions = {},
) {
  /**
   * How many rungs have lit. Rungs at `index < revealed` show their verdict;
   * everything else stays unlit — which is also, permanently, how a rung below
   * the match renders, so "pending" and "never reached" are the same look and
   * the walk teaches what it means.
   */
  const revealed = ref(0)

  /**
   * Total steps in the walk.
   *
   * A fallthrough (no rule matched) needs one step beyond the last rung, for
   * the `final` block to light — otherwise the ladder ends on a rung that did
   * not decide anything and the answer appears from nowhere.
   */
  const steps = computed(() =>
    matchedIndex.value >= 0 ? matchedIndex.value + 1 : count.value + 1,
  )

  const running = computed(() => revealed.value < steps.value)

  /** The `final` block lights only after every rung has failed to match. */
  const fallthroughRevealed = computed(
    () => matchedIndex.value < 0 && revealed.value >= steps.value,
  )

  let timer: ReturnType<typeof setInterval> | null = null

  const stop = () => {
    if (timer !== null) {
      clearInterval(timer)
      timer = null
    }
  }

  /** Jump to the end state — lit down to the match, and no further. */
  const skip = () => {
    stop()
    revealed.value = steps.value
  }

  const start = () => {
    stop()
    revealed.value = 0

    if (steps.value <= 0) return
    if (prefersReducedMotion() || options.instant?.value) {
      skip()
      return
    }

    const dwell = Math.max(16, Math.min(DWELL_MS, MAX_TOTAL_MS / steps.value))
    timer = setInterval(() => {
      revealed.value += 1
      if (revealed.value >= steps.value) stop()
    }, dwell)
  }

  /**
   * Restart on a new result.
   *
   * `runToken` is the one that actually guarantees it — count and matchedIndex
   * are both unchanged when a re-probe lands on the same answer, which is
   * precisely the case where a stale finished ladder is most misleading. The
   * other two are kept so a caller that supplies no token still restarts on
   * anything structurally different.
   */
  watch(
    [count, matchedIndex, () => options.runToken?.value],
    start,
    { immediate: true },
  )

  onUnmounted(stop)

  return {
    revealed,
    steps,
    running,
    fallthroughRevealed,
    /** Whether rung `index` has lit yet. */
    isRevealed: (index: number) => index < revealed.value,
    skip,
    replay: start,
  }
}
