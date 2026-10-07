/**
 * The frontend half of the config pre-check (backend: reference_check.go).
 *
 * `sing-box check` passes a config whose `route.final`, route rule or detour
 * names an outbound that does not exist, and the core then fails to START on
 * it. The panel checks those references itself and, when one is broken,
 * answers with `code: ConfigError` and a body describing each problem. This
 * module reads that body and says where each problem is fixed.
 */

export type ProblemKind = 'missing_outbound' | 'empty_group'

/** Setting families, as the backend names them. */
export type ProblemField = 'route.final' | 'route.rules' | 'route.rule_set' | 'dns.servers'

export interface ConfigProblem {
  kind: ProblemKind
  field: ProblemField
  /** Exact config path: "route.final", "route.rules[3]". */
  where: string
  tag: string
}

/** What marks an error body as this kind of config error. */
export const REFERENCE_REASON = 'outbound_reference'

const KINDS: readonly string[] = ['missing_outbound', 'empty_group']

/** The page where each kind of setting is edited. */
const DESTINATIONS: Record<ProblemField, string> = {
  'route.final': '/dashboard/route/final-policy',
  'route.rules': '/dashboard/route/rules',
  'route.rule_set': '/dashboard/route/rule-sets',
  'dns.servers': '/dashboard/dns/servers',
}

const isField = (value: unknown): value is ProblemField =>
  typeof value === 'string' && Object.prototype.hasOwnProperty.call(DESTINATIONS, value)

/**
 * Extracts the problems from an error response's `data`, or returns [] when
 * the body is anything else.
 *
 * Strict on purpose. This decides whether a dialog interrupts the operator, and
 * an entry it cannot fully understand — a field this build has no page for, a
 * missing tag — would produce a row that says nothing and a button that goes
 * nowhere. Such entries are dropped; if none survive, the caller falls back to
 * showing the error's own message.
 */
export function parseConfigProblems(data: unknown): ConfigProblem[] {
  if (typeof data !== 'object' || data === null) return []
  const body = data as Record<string, unknown>
  if (body.reason !== REFERENCE_REASON || !Array.isArray(body.problems)) return []

  const problems: ConfigProblem[] = []
  for (const entry of body.problems) {
    if (typeof entry !== 'object' || entry === null) continue
    const { kind, field, where, tag } = entry as Record<string, unknown>
    if (typeof kind !== 'string' || !KINDS.includes(kind)) continue
    if (!isField(field) || typeof tag !== 'string' || tag === '') continue
    problems.push({
      kind: kind as ProblemKind,
      field,
      where: typeof where === 'string' && where !== '' ? where : field,
      tag,
    })
  }
  return problems
}

export function destinationFor(field: ProblemField): string {
  return DESTINATIONS[field]
}

export interface ProblemGroup {
  field: ProblemField
  to: string
  problems: ConfigProblem[]
}

/**
 * Groups problems by the page that fixes them, `route.final` first.
 *
 * First because it is the one that stops the service outright whatever traffic
 * arrives — "default outbound not found" — and because it is the common case:
 * one setting, one picker, fixed in a click.
 */
export function groupProblems(problems: readonly ConfigProblem[]): ProblemGroup[] {
  const order: ProblemField[] = ['route.final', 'route.rules', 'route.rule_set', 'dns.servers']
  return order
    .map((field) => ({ field, to: DESTINATIONS[field], problems: problems.filter((problem) => problem.field === field) }))
    .filter((group) => group.problems.length > 0)
}
