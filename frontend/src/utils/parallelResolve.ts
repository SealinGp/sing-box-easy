/**
 * Generating the "race several resolvers, take the first good answer" pattern.
 *
 * WHY THIS IS GENERATED RATHER THAN HAND-BUILT
 * ────────────────────────────────────────────
 * The pattern is N+N rules that only work as a set, and sing-box's failure
 * mode for getting it wrong is unusually harsh. From the 1.14 docs:
 *
 *   `respond` is "Only allowed after a preceding top-level `evaluate` rule. If
 *   the action is reached without an evaluated response at runtime, the
 *   request FAILS with an error instead of falling through to later rules."
 *
 * So a mis-ordered or half-entered group does not resolve more slowly or fall
 * back to the next rule — it breaks resolution for everything it matches. Add
 * to that: every evaluate must repeat the same conditions, each `tag` must be
 * unique and each `match_response` must name one exactly, and a three-server
 * group is six rules typed by hand. That is a generator's job.
 *
 * HOW THE PATTERN WORKS
 * ─────────────────────
 * The N `evaluate` rules each send the query to one server and store the
 * answer under a tag; evaluate does not terminate matching, so all N dispatch.
 * The N `respond` rules then each carry `match_response: <tag>`, `race: true`
 * and `response_rcode: NOERROR`. Per the docs, race rules do not hold the
 * listed order: each is judged as soon as its response arrives, and "the first
 * race rule that matches terminates rule evaluation immediately; the remaining
 * queries are canceled".
 *
 * `response_rcode: NOERROR` is what makes it a race for the first USEFUL
 * answer rather than the first answer: a resolver returning NXDOMAIN loses
 * instead of winning with a negative result.
 */

/** Condition fields the group can match on, mirroring the rule form. */
export interface ParallelConditions {
  rule_set?: string[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  geosite?: string[]
}

export interface ParallelResolveSpec {
  conditions: ParallelConditions
  /** Server tags to race, in listed order. At least two. */
  servers: string[]
  /** Prefix for the generated response tags. */
  group: string
  /** Per-query timeout applied to every evaluate. Optional. */
  timeout?: string
}

/** A generated rule. Loose by design: it goes straight to the API. */
export type GeneratedRule = Record<string, any>

/**
 * The rcode a racing respond accepts.
 *
 * Fixed rather than exposed. The whole point of the pattern is "first server
 * that actually answers wins", and any other value turns the race into
 * something else — accepting NXDOMAIN, for instance, lets the fastest resolver
 * win by failing.
 */
const ACCEPTED_RCODE = 'NOERROR'

/**
 * Builds the rules, evaluates first.
 *
 * The ordering is a correctness requirement, not a preference — see the file
 * header. Callers must insert the returned array contiguously and in order.
 */
export function buildParallelResolveGroup(spec: ParallelResolveSpec): GeneratedRule[] {
  const conditions = compactConditions(spec.conditions)
  const timeout = spec.timeout?.trim()
  const tags = responseTagsFor(spec.group, spec.servers)

  const evaluates = spec.servers.map((server, index) => {
    const rule: GeneratedRule = {
      // Conditions are repeated on EVERY evaluate: each is matched
      // independently, so putting them on the first alone would dispatch one
      // query and skip the rest.
      ...conditions,
      action: 'evaluate',
      server,
      tag: tags[index],
    }
    if (timeout) rule.timeout = timeout
    return rule
  })

  const responds = tags.map((tag) => ({
    // No conditions here. A respond selects by match_response, not by the
    // query, and repeating the domain list would suggest it is what decides
    // the winner when the arrival order is.
    match_response: tag,
    response_rcode: ACCEPTED_RCODE,
    action: 'respond',
    race: true,
  }))

  return [...evaluates, ...responds]
}

/**
 * Response tags: `<group>_<server>`, made safe and unique.
 *
 * A server tag is free-form and commonly contains spaces or emoji. The
 * response tag is a key that `match_response` has to name exactly, so
 * whitespace is collapsed to underscores for legibility; non-ASCII is kept,
 * because sing-box tags are free-form and the operator chose that name.
 *
 * Uniqueness is enforced after sanitising, since two distinct server tags can
 * collapse to the same string — and two evaluates sharing a tag would leave
 * one respond rule pointed at the wrong response.
 */
function responseTagsFor(group: string, servers: string[]): string[] {
  const prefix = sanitiseTagPart(group)
  const used = new Set<string>()

  return servers.map((server) => {
    const base = `${prefix}_${sanitiseTagPart(server)}`
    let tag = base
    let suffix = 2
    while (used.has(tag)) {
      tag = `${base}_${suffix++}`
    }
    used.add(tag)
    return tag
  })
}

function sanitiseTagPart(value: string): string {
  return value.trim().replace(/[\s|/\\.]+/g, '_').replace(/_+/g, '_').replace(/^_|_$/g, '')
}

/** Drops empty lists so the rule carries only conditions that constrain it. */
function compactConditions(conditions: ParallelConditions): Record<string, string[]> {
  const compact: Record<string, string[]> = {}
  for (const [key, value] of Object.entries(conditions)) {
    if (Array.isArray(value) && value.length > 0) compact[key] = [...value]
  }
  return compact
}

/**
 * Validates a spec against the rules already in the config.
 *
 * Returns an i18n key, or null when the spec is sound — the same shape
 * validateDNSRuleAction uses, so the caller toasts it identically.
 */
export function validateParallelResolveGroup(
  spec: ParallelResolveSpec,
  existingRules: readonly GeneratedRule[],
): string | null {
  if (!spec.group.trim()) return 'dns.rules.parallel.errors.needGroup'

  const servers = spec.servers.filter((server) => server.trim())
  // One server is not a race — it is a `route` rule, and generating a pair for
  // it spends two rules doing one rule's job.
  if (servers.length < 2) return 'dns.rules.parallel.errors.needTwoServers'

  // No conditions is allowed on purpose: the production config's final
  // fallback races three resolvers unconditionally, and a generator that
  // cannot produce the pattern it is modelled on is not much of a generator.

  const taken = new Set<string>()
  for (const rule of existingRules) {
    if (typeof rule.tag === 'string' && rule.tag) taken.add(rule.tag)
  }
  for (const tag of responseTagsFor(spec.group, servers)) {
    // A duplicate tag does not error at load: it silently re-points whichever
    // respond rule names it at this group's response.
    if (taken.has(tag)) return 'dns.rules.parallel.errors.tagExists'
  }

  return null
}

/**
 * Indices of `respond` rules that no preceding `evaluate` can satisfy.
 *
 * WHAT THIS IS ACTUALLY FOR — measured, not assumed.
 * sing-box validates this statically: deleting an evaluate or reordering a
 * respond above its evaluate is rejected by `sing-box check` with
 * "dns rule[33]: undefined evaluate tag: <tag>", so the panel's
 * validate-then-commit workflow already prevents an orphan reaching a running
 * config. This is NOT the last line of defence and must not be described as
 * one.
 *
 * It earns its place by being EARLY. Reorder mode is a batch edit: the list
 * sorts live while dragging and nothing is written until Save, so without this
 * the operator arranges rules, presses Save, and only then meets an opaque
 * upstream FATAL string naming a rule index that has since moved. Computed off
 * the live list, the marker appears on the offending row the instant a drag
 * separates a group — while the drag is still undoable.
 *
 * It also covers rules the panel did not write, which is the case a static
 * check cannot help with: a config being composed in the raw editor before it
 * is validated.
 */
export function findOrphanRespondRules(rules: readonly GeneratedRule[]): number[] {
  const orphans: number[] = []
  const seenTags = new Set<string>()
  // `match_response: true` references the latest evaluate WITHOUT a tag, so a
  // tagged one does not satisfy it.
  let sawUntaggedEvaluate = false

  rules.forEach((rule, index) => {
    if (rule.action === 'evaluate') {
      if (typeof rule.tag === 'string' && rule.tag) seenTags.add(rule.tag)
      else sawUntaggedEvaluate = true
      return
    }
    if (rule.action !== 'respond') return

    const reference = rule.match_response
    if (reference === true) {
      if (!sawUntaggedEvaluate) orphans.push(index)
      return
    }
    if (typeof reference === 'string' && reference) {
      if (!seenTags.has(reference)) orphans.push(index)
      return
    }
    // A respond with no match_response at all relies on the latest untagged
    // evaluate, the same as `true`.
    if (!sawUntaggedEvaluate) orphans.push(index)
  })

  return orphans
}

/**
 * A starting group name derived from what the group matches.
 *
 * Suggested rather than required: the tags end up in the config and a name
 * like `deepseek_com` reads far better than `parallel_1` when someone opens
 * config.json six months later.
 */
export function suggestGroupName(conditions: ParallelConditions): string {
  const ordered = [
    conditions.rule_set,
    conditions.geosite,
    conditions.domain_suffix,
    conditions.domain,
    conditions.domain_keyword,
  ]
  for (const values of ordered) {
    const first = values?.find((value) => value.trim())
    if (first) return sanitiseTagPart(first.replace(/^[.*]+/, ''))
  }
  return 'parallel'
}
