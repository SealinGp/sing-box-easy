/**
 * The shape the rule ladder renders, shared by DNS and route diagnostics.
 *
 * Both probes already answer the same question — "which rung handled this, and
 * what happened at every rung above it" — but they answered it in two different
 * payloads and, until now, two different amounts of detail: the route panel
 * rendered a verdict per rule, while the DNS diagram threw its per-rule verdicts
 * away and highlighted only the winner. This is the common denominator, so one
 * component can render both.
 *
 * WHY "SKIPPED" IS DERIVED AND NOT TRUSTED FROM THE WIRE
 * ─────────────────────────────────────────────────────
 * `dnsprobe.Attribute` evaluates EVERY rule, including the ones below the match
 * (match.go:94-115 — it keeps walking after `decided`, it just stops acting on
 * the result). Those verdicts are real, but sing-box never consulted those rules
 * for this query, so presenting one as "no match" states a fact about a
 * comparison that did not happen. Anything past the deciding rung is therefore
 * re-labelled `skipped` here regardless of what the backend computed for it.
 *
 * `routeprobe` already marks them `skipped` itself. Deriving it in one place
 * means the two agree without the DNS backend having to change.
 */

/** Verdict for one rung. Shared by both probe families. */
export type LadderState = 'matched' | 'not_matched' | 'unevaluated' | 'skipped'

export interface LadderRung {
  /** Position in the rule list, as the config numbers it. */
  index: number
  state: LadderState
  /** The rule's conditions, rendered by whichever backend produced it. */
  summary: string
  /** The action and its result, pre-rendered — "route(proxy)", "reject". */
  outcome: string
  /**
   * The rung that actually decided the query.
   *
   * Not the same as `state === 'matched'`: a route rule with a `sniff` or
   * `resolve` action matches, changes what the rules below it see, and hands
   * over. Those rungs are lit green and still are not the answer.
   */
  deciding: boolean
  /** Matched without deciding — see `deciding`. */
  continues?: boolean
  /** Conditions that could not be decided offline, e.g. ["rule_set"]. */
  unevaluated?: string[]
}

/**
 * A ladder, plus where the walk stops.
 *
 * `matchedIndex` is -1 when the query falls through to `final`, in which case
 * the walk runs to the end and the fallthrough block lights instead.
 */
export interface Ladder {
  rungs: LadderRung[]
  matchedIndex: number
}

/**
 * Re-labels everything below the deciding rung as never-reached.
 *
 * See the header note: a verdict the runtime never asked for is not a verdict.
 */
export function markUnreached(rungs: LadderRung[], matchedIndex: number): LadderRung[] {
  if (matchedIndex < 0) return rungs
  return rungs.map((rung) =>
    rung.index > matchedIndex ? { ...rung, state: 'skipped' as LadderState } : rung,
  )
}
