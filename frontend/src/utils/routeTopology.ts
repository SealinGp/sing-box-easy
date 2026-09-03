/**
 * Derives the inbound → rule → exit topology from a saved sing-box config.
 *
 * Pure and total: every degenerate input — a null config, a rules array with
 * junk in it, a tag nothing defines — degrades into the model rather than
 * throwing, because this runs on the Overview page where a config the panel
 * cannot fully parse is exactly the config an operator most needs to look at.
 *
 * See `types/routeTopology.ts` for why exits are a set and why termination is
 * copied from the engine instead of reasoned about.
 */
import { ALL_MATCHER_KEYS } from '../schemas/routeRuleMatcherFields'
import { isFieldFilled } from '../schemas/optionSchema'
import type { SingBoxConfig } from '../types/api'
import type {
  FallthroughSource,
  RouteTopology,
  TopologyCondition,
  TopologyExit,
  TopologyInbound,
  TopologyRule,
} from '../types/routeTopology'

/**
 * Actions that stop rule matching.
 *
 * Mirrors `app/pkg/routeprobe/rule_meta.go:39-53`, which mirrors sing-box's
 * `route/route.go:478-484`. `direct` is deliberately absent — see the type
 * module's header.
 */
const TERMINAL_ACTIONS = new Set(['route', 'reject', 'hijack-dns'])

/** Group outbounds carry a member list; a leaf does not. */
const GROUP_TYPES = new Set(['selector', 'urltest'])

/** The exit id used when sing-box has to synthesise a direct outbound. */
const IMPLICIT_DIRECT_ID = 'direct'

/** How many values of one matcher to spell out before collapsing to a count. */
export const MAX_CONDITION_VALUES = 3

/** Anything list-like on the wire may arrive as a bare scalar. */
function toArray(value: unknown): string[] {
  if (value === undefined || value === null) return []
  const values = Array.isArray(value) ? value : [value]
  return values.filter((entry) => entry !== undefined && entry !== null).map(String)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/** The action a rule carries, defaulting to what sing-box assumes. */
function actionName(rule: Record<string, unknown>): string {
  const action = rule.action
  return typeof action === 'string' && action !== '' ? action : 'route'
}

/**
 * Renders a condition for display, collapsing a long value list to a count.
 *
 * A boolean matcher (`ip_is_private`) has no values — the key IS the statement,
 * so it prints alone rather than as an empty list.
 */
export function formatCondition(
  condition: TopologyCondition,
  max: number = MAX_CONDITION_VALUES,
): string {
  if (condition.values.length === 0) return condition.key
  const shown = condition.values.slice(0, max)
  const overflow = condition.values.length - shown.length
  return overflow > 0 ? `${shown.join(', ')} +${overflow}` : shown.join(', ')
}

/** Every tag that can legally appear as a rule's `outbound`, with its type. */
function collectDefinedTags(config: SingBoxConfig | null | undefined): Map<string, string> {
  const defined = new Map<string, string>()
  // Endpoints (wireguard and friends) are dialable by tag exactly like an
  // outbound, so leaving them out would report a working config as broken.
  for (const list of [config?.outbounds, (config as any)?.endpoints]) {
    for (const entry of Array.isArray(list) ? list : []) {
      if (!isRecord(entry)) continue
      const tag = typeof entry.tag === 'string' ? entry.tag : ''
      if (tag !== '') defined.set(tag, typeof entry.type === 'string' ? entry.type : '')
    }
  }
  return defined
}

/** Member count for a group outbound; undefined for a leaf. */
function memberCountOf(config: SingBoxConfig | null | undefined, tag: string): number | undefined {
  const entry = (config?.outbounds ?? []).find(
    (candidate) => isRecord(candidate) && candidate.tag === tag,
  )
  if (!isRecord(entry)) return undefined
  if (!GROUP_TYPES.has(String(entry.type))) return undefined
  return toArray(entry.outbounds).length
}

/**
 * Accumulates exits in first-reference order.
 *
 * Order matters for layout: a right column sorted by when the config first
 * mentions each exit keeps ribbons roughly parallel, where an alphabetical or
 * arbitrary order would cross most of them.
 */
class ExitTable {
  private readonly byId = new Map<string, TopologyExit>()
  private readonly defined: Map<string, string>
  private readonly config: SingBoxConfig | null | undefined

  constructor(defined: Map<string, string>, config: SingBoxConfig | null | undefined) {
    this.defined = defined
    this.config = config
  }

  /** Returns the exit's id, creating the node on first reference. */
  reference(
    id: string,
    seed: () => Omit<TopologyExit, 'id' | 'ruleIndices' | 'isFinal'>,
    ruleIndex?: number,
  ): string {
    let exit = this.byId.get(id)
    if (!exit) {
      exit = { id, ruleIndices: [], isFinal: false, ...seed() }
      this.byId.set(id, exit)
    }
    if (ruleIndex !== undefined) {
      // Immutable update: the array is replaced, never pushed into.
      this.byId.set(id, { ...exit, ruleIndices: [...exit.ruleIndices, ruleIndex] })
    }
    return id
  }

  /** References an outbound tag, resolving whether anything actually defines it. */
  referenceOutbound(tag: string, ruleIndex?: number): string {
    return this.reference(
      tag,
      () => {
        const type = this.defined.get(tag)
        return type === undefined
          ? { label: tag, kind: 'missing' as const }
          : {
              label: tag,
              kind: 'outbound' as const,
              type,
              memberCount: memberCountOf(this.config, tag),
            }
      },
      ruleIndex,
    )
  }

  markFinal(id: string): void {
    const exit = this.byId.get(id)
    if (exit) this.byId.set(id, { ...exit, isFinal: true })
  }

  list(): TopologyExit[] {
    return [...this.byId.values()]
  }
}

/**
 * Resolves the exit for one rule's action.
 *
 * Returns null for a non-terminal action: `sniff` and `resolve` change what the
 * rules below them see and then hand over, so no traffic leaves there.
 */
function exitForRule(
  rule: Record<string, unknown>,
  index: number,
  action: string,
  exits: ExitTable,
): { exitId: string | null; outcome?: string } {
  switch (action) {
    case 'route': {
      const tag = typeof rule.outbound === 'string' ? rule.outbound : ''
      // A route rule with no outbound is invalid, but the panel must still draw
      // it — that is precisely the config someone is here to diagnose.
      if (tag === '') return { exitId: null }
      return { exitId: exits.referenceOutbound(tag, index), outcome: tag }
    }
    case 'reject': {
      const method = typeof rule.method === 'string' && rule.method !== '' ? rule.method : 'default'
      return {
        exitId: exits.reference('reject', () => ({ label: 'reject', kind: 'reject', detail: method }), index),
        outcome: method,
      }
    }
    case 'hijack-dns':
      return {
        exitId: exits.reference('hijack-dns', () => ({ label: 'hijack-dns', kind: 'hijack-dns' }), index),
      }
    default:
      return {
        exitId: null,
        outcome: typeof rule.server === 'string' ? rule.server : undefined,
      }
  }
}

/**
 * Reproduces sing-box's fall-through, which has three cases that are easy to
 * conflate (`adapter/outbound/manager.go:59-75,289`).
 */
function resolveFallthrough(
  config: SingBoxConfig | null | undefined,
  exits: ExitTable,
): { exitId: string; source: FallthroughSource } {
  const final = config?.route?.final
  if (typeof final === 'string' && final !== '') {
    return { exitId: exits.referenceOutbound(final), source: 'final' }
  }

  const first = (config?.outbounds ?? []).find(
    (entry) => isRecord(entry) && typeof entry.tag === 'string' && entry.tag !== '',
  )
  if (isRecord(first)) {
    return { exitId: exits.referenceOutbound(String(first.tag)), source: 'first_outbound' }
  }

  return {
    exitId: exits.reference(IMPLICIT_DIRECT_ID, () => ({
      label: IMPLICIT_DIRECT_ID,
      kind: 'implicit',
    })),
    source: 'implicit_direct',
  }
}

function buildInbounds(config: SingBoxConfig | null | undefined): TopologyInbound[] {
  // Cast through `unknown[]`: the typed `Inbound` union has no common
  // `listen_port` (a tun inbound does not listen on a port), and narrowing it
  // per member here would duplicate the registry for no gain.
  return ((Array.isArray(config?.inbounds) ? config.inbounds : []) as unknown[])
    .filter(isRecord)
    .map((inbound) => ({
      tag: typeof inbound.tag === 'string' ? inbound.tag : '',
      type: typeof inbound.type === 'string' ? inbound.type : '',
      listenPort: typeof inbound.listen_port === 'number' ? inbound.listen_port : undefined,
    }))
}

/**
 * Filled matchers, in the generated inventory's order.
 *
 * Taken from the schema rather than a hand-written list, so a matcher a future
 * sing-box adds shows up here the moment the inventory is regenerated — the
 * same reason the rule form is schema-driven.
 */
function buildConditions(rule: Record<string, unknown>): TopologyCondition[] {
  return ALL_MATCHER_KEYS.filter(
    (key) => key !== 'invert' && isFieldFilled(rule[key as string]),
  ).map((key) => {
    const value = rule[key as string]
    // A boolean matcher carries its meaning in the key alone.
    return { key: key as string, values: typeof value === 'boolean' ? [] : toArray(value) }
  })
}

/**
 * Builds the whole topology.
 *
 * The one ordering constraint: rules are walked before the fall-through is
 * resolved, so `exits` comes out in first-reference order with the final exit
 * appended only if no rule already named it.
 */
export function buildRouteTopology(config: SingBoxConfig | null | undefined): RouteTopology {
  const exits = new ExitTable(collectDefinedTags(config), config)

  const rawRules = Array.isArray(config?.route?.rules) ? config.route.rules : []

  /**
   * Index of the first terminal rule that matches everything. A rule with no
   * items returns true for every connection (`rule_abstract.go:55`), so a
   * terminal one swallows the whole remainder of the list.
   */
  let swallowedFrom: number | null = null

  const rules: TopologyRule[] = []

  rawRules.forEach((raw, index) => {
    if (!isRecord(raw)) return

    const action = actionName(raw)
    const terminal = TERMINAL_ACTIONS.has(action)
    const conditions = buildConditions(raw)
    const inverted = isFieldFilled(raw.invert)
    const { exitId, outcome } = exitForRule(raw, index, action, exits)

    // An inverted condition-less rule matches NOTHING, not everything, so it
    // is excluded rather than treated as the strongest possible catch-all.
    const catchAll = terminal && conditions.length === 0 && !inverted

    rules.push({
      index,
      action,
      conditions,
      inverted,
      scopedInbounds: toArray(raw.inbound),
      terminal,
      exitId,
      outcome,
      catchAll,
      reachable: swallowedFrom === null,
    })

    if (catchAll && swallowedFrom === null) swallowedFrom = index
  })

  const fallthrough = resolveFallthrough(config, exits)
  exits.markFinal(fallthrough.exitId)

  return { inbounds: buildInbounds(config), rules, exits: exits.list(), fallthrough }
}
