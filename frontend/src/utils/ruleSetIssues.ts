/**
 * Grouping unreadable rule sets by their cause.
 *
 * Extracted from the component so it is testable, following the pattern the
 * flow diagram already uses: the model is a plain function and the .vue file
 * is markup only.
 */
import type { RuleSetStatus } from '../types/ruleSet'

export interface RuleSetIssue {
  /** Stable key for rendering. */
  key: string
  reason: string
  /** The underlying message, which usually names the offending path. */
  detail?: string
  tags: string[]
}

/**
 * One entry per distinct cause, carrying the tags it affects.
 *
 * Grouping matters because the causes are shared: a config with fourteen
 * remote rule sets and no cache file produced fourteen identical lines, which
 * reads as fourteen problems and buries the one sentence explaining all of
 * them.
 *
 * The key is reason AND detail, not reason alone. Two sets can fail for the
 * same reason from different places — a missing local file at one path and a
 * missing cache at another — and merging those would attribute both to
 * whichever detail happened to arrive first.
 *
 * Sets with no `reason` are healthy and are skipped. Order follows first
 * appearance, so the list matches the order the rules were evaluated in.
 */
export function groupRuleSetIssues(sets: readonly RuleSetStatus[]): RuleSetIssue[] {
  const grouped = new Map<string, RuleSetIssue>()

  for (const set of sets) {
    if (!set.reason) continue

    const key = `${set.reason}\u0000${set.detail ?? ''}`
    const existing = grouped.get(key)
    if (existing) {
      // The same set is reached from many rules; list each tag once.
      if (!existing.tags.includes(set.tag)) existing.tags.push(set.tag)
      continue
    }
    grouped.set(key, { key, reason: set.reason, detail: set.detail, tags: [set.tag] })
  }

  return [...grouped.values()]
}
