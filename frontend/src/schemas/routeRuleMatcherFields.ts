/**
 * Route rule matcher curation — the editorial layer over the generated
 * inventory.
 *
 * The generic machinery lives in `optionSchema.ts`. What is specific here is
 * the GROUPING, which is the load-bearing decision carried over verbatim from
 * `RouteRuleMatchers.vue` — it predates the schema and survives it unchanged:
 *
 *   CONTENT matchers (domain, domain_suffix, domain_keyword, domain_regex)
 *   describe *what* the traffic is. A rule set already expresses exactly these,
 *   so combining the two in one rule is the AND trap — "in the rule set AND
 *   also this domain" — and they are therefore presented as alternatives to
 *   `rule_set`.
 *
 *   CONTEXT matchers (inbound, protocol, network, port, process, user, wifi, …)
 *   describe *where the traffic came from*. A rule set cannot express them, so
 *   narrowing a rule set by `network: udp` or `port: 443` is a normal, correct
 *   thing to do. These are never folded away by the rule-set choice — only by
 *   their own emptiness.
 *
 * Collapsing every matcher against `rule_set` would have been the simpler code
 * and the wrong model: it would flag the most common correct route rule in
 * sing-box (a rule set narrowed by port or network) as a mistake.
 *
 * WHAT THE OLD FORM COULD EDIT
 * ────────────────────────────
 * 9 of the 37 matchers sing-box accepts. The 28 with no control included
 * `ip_cidr` — which this repo's OWN init wizard creates
 * (views/init-steps/ConfigureRoutes.vue) and the form could then never edit.
 *
 * TWO OF THE NINE WERE DEAD
 * ─────────────────────────
 * `geosite` and `geoip` had curated dropdowns with 12 and 7 options, promoted
 * as the primary content matchers. sing-box REMOVED them in 1.12.0 and answers
 * with a hard startup error, so all 19 values produced a rule that could not
 * run. They are not listed below — the generated inventory carries
 * `removed: '1.12.0'` and the §4 version gate withholds them, while still
 * rendering one that a loaded config already uses so it can be seen and
 * cleared. The replacement is a rule set.
 *
 * PORT IS NOT PORT_RANGE
 * ──────────────────────
 * `port` is `Listable[uint16]`; a range is a decode failure. The old field
 * deliberately kept range syntax as a string and its placeholder advertised
 * "8080-8090", which sing-box rejects twice over — wrong field, and the
 * separator is a colon. `port_range` is curated in beside it.
 */
import {
  ROUTE_RULE_MATCHER_INVENTORY,
  type RouteRuleMatcherFieldKey,
  type RouteRuleMatcherTypeName,
} from './routeRuleMatcherInventory.generated'
import { createSchema, type FieldCuration, type ResolvedField } from './optionSchema'
import { INTERFACE_TYPES } from './vocabularies'

/** The sections the matcher form renders. Order is render order. */
export const MATCHER_GROUPS = ['ruleSet', 'content', 'context'] as const
export type MatcherGroup = (typeof MATCHER_GROUPS)[number]

type MatcherKey = RouteRuleMatcherFieldKey<'default'>

/**
 * sing-box's sniffed protocol vocabulary. Not exhaustive — sing-box accepts any
 * sniffer name — but these are the ones a rule realistically matches on, and the
 * field stays free-text via `chips` so an unlisted one is still reachable.
 */
const PROTOCOLS = ['http', 'tls', 'quic', 'dns', 'stun', 'bittorrent', 'dtls', 'ssh', 'rdp'] as const

/** Clash API modes. `clash_mode` matches against the mode selected there. */
const CLASH_MODES = ['Global', 'Rule', 'Direct'] as const

const BY_TYPE: {
  [T in RouteRuleMatcherTypeName]: Partial<Record<RouteRuleMatcherFieldKey<T>, FieldCuration>>
} = {
  default: {
    // ── Rule set ────────────────────────────────────────────────────────────
    // Its own group: the alternative to the content matchers, not one of them.
    //
    // NOTE ON TIERS IN THIS FILE. Only `rule_set` is core; every other matcher
    // is `advanced`, including the common ones. That is deliberate and differs
    // from the other domains: SchemaFieldsEditor SEEDS `typical` fields on
    // mount, which is right for an inbound (you want its characteristic fields
    // waiting for you) and wrong for a condition — a rule that pre-opens four
    // empty condition boxes reads as though it already constrains something.
    // `advanced` reproduces what the hand-written form did: shown when the
    // loaded rule uses it, otherwise behind the "add a condition" row.
    rule_set: { tier: 'core', order: 10, group: 'ruleSet', control: 'rule-set' },
    rule_set_ip_cidr_match_source: { tier: 'advanced', order: 20, group: 'ruleSet' },

    // ── Content: what the traffic is ────────────────────────────────────────
    domain: { tier: 'advanced', order: 10, group: 'content', control: 'chips' },
    domain_suffix: { tier: 'advanced', order: 20, group: 'content', control: 'chips' },
    domain_keyword: { tier: 'advanced', order: 30, group: 'content', control: 'chips' },
    domain_regex: { tier: 'advanced', order: 40, group: 'content', control: 'chips' },
    // Destination address. `ip_cidr` is promoted because the init wizard emits
    // it and the form previously could not edit what it had created.
    ip_cidr: { tier: 'advanced', order: 50, group: 'content', control: 'chips' },
    ip_is_private: { tier: 'advanced', order: 60, group: 'content' },
    ip_version: {
      tier: 'advanced',
      order: 70,
      group: 'content',
      control: 'select',
      options: ['4', '6'],
    },

    // ── Context: where the traffic came from ────────────────────────────────
    inbound: { tier: 'advanced', order: 10, group: 'context', control: 'chips' },
    protocol: { tier: 'advanced', order: 20, group: 'context', control: 'chips' },
    network: {
      tier: 'advanced',
      order: 30,
      group: 'context',
      control: 'chips',
    },
    port: {
      tier: 'advanced',
      order: 40,
      group: 'context',
      control: 'chips',
      hintKey: 'route.rules.hints.port',
    },
    port_range: {
      tier: 'advanced',
      order: 50,
      group: 'context',
      control: 'chips',
      placeholder: '8080:8090',
      hintKey: 'route.rules.hints.portRange',
    },

    source_ip_cidr: { tier: 'advanced', order: 60, group: 'context', control: 'chips' },
    source_ip_is_private: { tier: 'advanced', order: 70, group: 'context' },
    source_port: { tier: 'advanced', order: 80, group: 'context', control: 'chips' },
    source_port_range: {
      tier: 'advanced',
      order: 90,
      group: 'context',
      control: 'chips',
      placeholder: '8080:8090',
    },

    clash_mode: {
      tier: 'advanced',
      order: 100,
      group: 'context',
      control: 'select',
      options: CLASH_MODES,
    },

    // Client-side identity. Only meaningful where sing-box can see the process,
    // i.e. running on the same host as the client.
    process_name: { tier: 'advanced', order: 110, group: 'context', control: 'chips' },
    process_path: { tier: 'advanced', order: 120, group: 'context', control: 'chips' },
    process_path_regex: { tier: 'advanced', order: 130, group: 'context', control: 'chips' },
    package_name: { tier: 'advanced', order: 140, group: 'context', control: 'chips' },
    user: { tier: 'advanced', order: 150, group: 'context', control: 'chips' },
    user_id: { tier: 'advanced', order: 160, group: 'context', control: 'chips' },
    auth_user: { tier: 'advanced', order: 170, group: 'context', control: 'chips' },
    client: { tier: 'advanced', order: 180, group: 'context', control: 'chips' },

    // Network conditions — mobile-oriented, and the reason network_type had to
    // stop being a number: it is a name (wifi/cellular/ethernet/other).
    network_type: { tier: 'advanced', order: 190, group: 'context', control: 'chips' },
    network_is_expensive: { tier: 'advanced', order: 200, group: 'context' },
    network_is_constrained: { tier: 'advanced', order: 210, group: 'context' },
    wifi_ssid: { tier: 'advanced', order: 220, group: 'context', control: 'chips' },
    wifi_bssid: { tier: 'advanced', order: 230, group: 'context', control: 'chips' },

    // Negates the whole rule. Context rather than content: it is about how the
    // rule is applied, not what it matches.
    invert: { tier: 'advanced', order: 240, group: 'context' },
  },
}

const schema = createSchema<RouteRuleMatcherTypeName>({
  inventory: ROUTE_RULE_MATCHER_INVENTORY,
  labelPrefix: 'route.rules.fields',
  byType: BY_TYPE as Partial<Record<RouteRuleMatcherTypeName, Record<string, FieldCuration>>>,
  // A route rule's `type` is the matcher SHAPE ("default"/"logical"), and the
  // matcher domain has only the one entry, so nothing here writes it.
  identityKeys: ['type'],
})

/**
 * Fields for one section.
 *
 * Anything uncurated resolves to `advanced` with no group — those land in
 * `context`, which is the safe default: an ungrouped matcher shown next to the
 * content matchers would be folded away by the rule-set choice, and a matcher
 * sing-box adds in a future version must never silently disappear because a
 * rule set is selected.
 */
export function resolveMatcherFields(group: MatcherGroup): ResolvedField[] {
  const all = schema.resolveFields('default')
  if (group === 'context') {
    return all.filter((f) => f.group === 'context' || f.group === undefined)
  }
  return all.filter((f) => f.group === group)
}

/** Every matcher key, for the "does this rule match anything at all" check. */
export const ALL_MATCHER_KEYS = Object.keys(
  ROUTE_RULE_MATCHER_INVENTORY.default,
) as MatcherKey[]

/**
 * Content matcher keys — the ones that are an alternative to a rule set, and so
 * the ones the mix warning is about. Derived from the curation rather than
 * hand-listed a second time.
 */
export const CONTENT_MATCHER_KEYS = resolveMatcherFields('content').map((f) => f.key)

export { ROUTE_RULE_MATCHER_INVENTORY, INTERFACE_TYPES, PROTOCOLS }
export type { RouteRuleMatcherTypeName }
