/**
 * Route rule action curation — the editorial layer over the generated
 * inventory.
 *
 * Mirrors `dnsRuleActionFields.ts`; the generic machinery lives in
 * `optionSchema.ts`. Bound with `typeKey: 'action'` for the same reason: a route
 * rule's `type` is its matcher shape ("default"/"logical"), which varies
 * independently of what the rule does once it matches.
 *
 * WHAT THE OLD FORM COULD EDIT
 * ────────────────────────────
 * 9 fields across 6 of the 7 actions, each wired by a hand-written computed
 * get/set pair:
 *
 *   route          outbound            — the 10 embedded dial options missing
 *   route-options  override_address,
 *                  override_port,
 *                  network_strategy    — 7 missing
 *   direct         NOT OFFERED AT ALL  — all 17 missing
 *   reject         method              — no_drop missing
 *   sniff          sniffer, timeout    — complete
 *   resolve        server, strategy    — 3 missing
 *   hijack-dns     n/a                 — correct, it has no options
 *
 * Note the form only offered the dial options under `route-options`, but
 * sing-box accepts every one of them on `action: route` too — RouteActionOptions
 * embeds RawRouteOptionsActionOptions. The inventory has always known that; the
 * hand-written form did not.
 *
 * THE BUG THIS FIXES
 * ──────────────────
 * There was no pruning on the route path at all — no allowlist, no denylist,
 * nothing. The form seeds `{action: 'route', outbound: ''}`, so switching the
 * action shipped the previous action's fields. Reproduced against a running
 * panel:
 *
 *   POST {"action":"reject","outbound":"direct"}          -> 200, outbound SILENTLY DROPPED
 *   POST {"action":"sniff","outbound":"direct",...}       -> 400 unknown field "outbound"
 *
 * Silent data loss or a hard error depending on the combination.
 * `pruneForeignFields` derives the correct answer from the inventory, and keeps
 * every key no action owns — all 37 matchers, and `type`/`mode`/`rules` on a
 * logical rule.
 *
 * `detour` IS ABSENT FROM `direct` ON PURPOSE
 * ───────────────────────────────────────────
 * DirectActionOptions IS DialerOptions, so reflection finds `detour` — but
 * sing-box refuses it: "detour is not available in the current context". The
 * generator drops it via domain.ExcludedFields, so it cannot be curated here
 * even by mistake.
 */
import {
  ROUTE_RULE_ACTION_INVENTORY,
  type RouteRuleActionFieldKey,
  type RouteRuleActionTypeName,
} from './routeRuleActionInventory.generated'
import { createSchema, isFieldFilled, type FieldCuration } from './optionSchema'
import { DOMAIN_STRATEGIES, NETWORK_STRATEGIES } from './vocabularies'

/** Every field name across every action — so the shared map cannot hold a typo. */
type AnyRouteRuleActionFieldKey = {
  [T in RouteRuleActionTypeName]: RouteRuleActionFieldKey<T>
}[RouteRuleActionTypeName]

/**
 * What sing-box can sniff. `sniff` with an empty list enables every sniffer,
 * which is the common case, so this narrows rather than being required.
 */
const SNIFFERS = ['http', 'tls', 'quic', 'dns', 'stun', 'bittorrent', 'dtls', 'ssh', 'rdp'] as const

/**
 * Curation applying to any action possessing the field.
 *
 * Intersected with each action's real inventory rather than spread into it: a
 * spread bypasses TypeScript's excess-property check, so `outbound` could be
 * spread into `reject` and fail silently.
 */
const SHARED: Partial<Record<AnyRouteRuleActionFieldKey, FieldCuration>> = {
  // The embedded dial options, shared by `route` and `route-options`.
  override_address: { tier: 'advanced', order: 100, placeholder: '1.2.3.4' },
  override_port: { tier: 'advanced', order: 110, control: 'number' },
  network_strategy: {
    tier: 'advanced',
    order: 120,
    control: 'select',
    options: NETWORK_STRATEGIES,
  },
  fallback_delay: { tier: 'advanced', order: 130 },
  udp_connect: { tier: 'advanced', order: 140 },
  udp_disable_domain_unmapping: { tier: 'advanced', order: 150 },
  udp_timeout: { tier: 'advanced', order: 160, placeholder: '5m' },
  // Mutually exclusive with tls_record_fragment — see validateRouteRuleAction.
  tls_fragment: { tier: 'advanced', order: 170 },
  tls_fragment_fallback_delay: { tier: 'advanced', order: 180 },
  tls_record_fragment: { tier: 'advanced', order: 190 },
}

const BY_ACTION: {
  [T in RouteRuleActionTypeName]: Partial<Record<RouteRuleActionFieldKey<T>, FieldCuration>>
} = {
  route: {
    // The whole point of a route rule. Uses the same picker as an outbound's
    // detour, but unlike a detour an empty value is not "direct" — it is a rule
    // that routes nowhere.
    outbound: { tier: 'core', order: 10, control: 'outbound' },
  },

  // Its ten fields ARE the whole struct and sing-box rejects an all-empty one,
  // so the two most-reached-for knobs are promoted to give the form a body.
  'route-options': {
    override_address: { tier: 'typical', order: 10, placeholder: '1.2.3.4' },
    override_port: { tier: 'typical', order: 20, control: 'number' },
  },

  // DialerOptions, minus `detour`. Dials from here rather than through an
  // outbound, with this rule's own binding and timeouts.
  direct: {
    bind_interface: { tier: 'typical', order: 10, placeholder: 'eth0' },
    inet4_bind_address: { tier: 'advanced', order: 100 },
    inet6_bind_address: { tier: 'advanced', order: 110 },
    routing_mark: { tier: 'advanced', order: 120, control: 'number' },
    connect_timeout: { tier: 'advanced', order: 130, placeholder: '5s' },
    tcp_fast_open: { tier: 'advanced', order: 140 },
    tcp_multi_path: { tier: 'advanced', order: 150 },
    udp_fragment: { tier: 'advanced', order: 160 },
    reuse_addr: { tier: 'advanced', order: 170 },
    netns: { tier: 'advanced', order: 180 },
    protect_path: { tier: 'advanced', order: 190 },
    domain_resolver: { tier: 'advanced', order: 200, control: 'json' },
    network_strategy: {
      tier: 'advanced',
      order: 210,
      control: 'select',
      options: NETWORK_STRATEGIES,
    },
    network_type: { tier: 'advanced', order: 220, control: 'chips' },
    fallback_network_type: { tier: 'advanced', order: 230, control: 'chips' },
    fallback_delay: { tier: 'advanced', order: 240 },
  },

  reject: {
    method: {
      tier: 'core',
      order: 10,
      control: 'select',
      options: ['default', 'drop'],
      default: 'default',
    },
    no_drop: { tier: 'advanced', order: 100, hintKey: 'route.rules.hints.noDrop' },
  },

  // A behaviour, not a configuration: sing-box marshals it with no options at
  // all, so the editor shows an explicit empty state.
  'hijack-dns': {},

  sniff: {
    // Empty means "every sniffer", which is the usual intent — so this narrows
    // rather than being required.
    sniffer: {
      tier: 'typical',
      order: 10,
      control: 'chips',
      hintKey: 'route.rules.hints.sniffer',
    },
    timeout: { tier: 'advanced', order: 100, placeholder: '300ms' },
  },

  resolve: {
    // Reuses the DNS server picker built for DNS rule actions. Empty means the
    // default resolver rather than an error, so this is typical, not core.
    server: { tier: 'typical', order: 10, control: 'dns-server' },
    strategy: { tier: 'typical', order: 20, control: 'select', options: DOMAIN_STRATEGIES },
    disable_cache: { tier: 'advanced', order: 100 },
    rewrite_ttl: { tier: 'advanced', order: 110, control: 'number', placeholder: '60' },
    client_subnet: { tier: 'advanced', order: 120, placeholder: '1.2.3.0/24' },
  },
}

const schema = createSchema<RouteRuleActionTypeName>({
  inventory: ROUTE_RULE_ACTION_INVENTORY,
  labelPrefix: 'route.rules.fields',
  shared: SHARED as Record<string, FieldCuration>,
  byType: BY_ACTION as Partial<Record<RouteRuleActionTypeName, Record<string, FieldCuration>>>,
  typeKey: 'action',
  identityKeys: ['action'],
})

export const resolveRouteRuleActionFields = schema.resolveFields
export const pruneForeignFields = schema.pruneForeignFields
export const applyActionDefaults = schema.applyTypeDefaults
export const isRouteRuleAction = schema.isKnownType

/**
 * The action of a rule as loaded from config.json.
 *
 * An omitted `action` means "route" — RuleAction.UnmarshalJSON rewrites "" to
 * "route" before dispatching. The form surfaces that default but must not write
 * it back: a rule that omitted `action` keeps omitting it.
 */
export function actionOf(rule: Readonly<Record<string, unknown>>): RouteRuleActionTypeName {
  const action = rule.action
  return isRouteRuleAction(action) ? action : 'route'
}

/**
 * Constraints sing-box enforces while decoding or starting, which surface as an
 * opaque upstream string or — for the last one — not at all.
 *
 * Returns an i18n key, or undefined when the action is valid. Each was verified
 * with `sing-box check` against 1.13.11.
 */
export function validateRouteRuleAction(
  rule: Readonly<Record<string, unknown>>,
  knownOutbounds?: readonly string[],
): string | undefined {
  const action = actionOf(rule)

  if (action === 'route') {
    if (!isFieldFilled(rule.outbound)) return 'route.rules.errors.outboundRequired'
    // `sing-box check` PASSES for an outbound tag that does not exist, so this
    // is the only place it can be caught before the rule silently misroutes.
    if (knownOutbounds && !knownOutbounds.includes(rule.outbound as string)) {
      return 'route.rules.errors.outboundUnknown'
    }
  }

  if (action === 'route-options') {
    // "empty route option action" — the struct's UnmarshalJSON rejects an
    // all-zero value, so selecting the action and setting nothing is a hard
    // failure rather than a no-op.
    const hasAny = Object.keys(ROUTE_RULE_ACTION_INVENTORY['route-options']).some((key) =>
      isFieldFilled(rule[key]),
    )
    if (!hasAny) return 'route.rules.errors.routeOptionsEmpty'
  }

  // "`tls_fragment` and `tls_record_fragment` are mutually exclusive". Both live
  // on route and route-options.
  if (isFieldFilled(rule.tls_fragment) && isFieldFilled(rule.tls_record_fragment)) {
    return 'route.rules.errors.tlsFragmentExclusive'
  }

  // "no_drop is not available in current context"
  if (action === 'reject' && rule.method === 'drop' && isFieldFilled(rule.no_drop)) {
    return 'route.rules.errors.noDropWithDrop'
  }

  return undefined
}

export { ROUTE_RULE_ACTION_INVENTORY, SNIFFERS }
export type { RouteRuleActionTypeName }
export { ROUTE_RULE_ACTION_TYPE_NAMES } from './routeRuleActionInventory.generated'
