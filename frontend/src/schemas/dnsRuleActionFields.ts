/**
 * DNS rule action curation — the editorial layer over the generated inventory.
 *
 * Mirrors `dnsServerFields.ts`; the generic machinery lives in `optionSchema.ts`.
 * See that file for the tier model and why uncurated fields stay reachable.
 *
 * THE DISCRIMINATOR IS `action`, NOT `type`
 * ─────────────────────────────────────────
 * Every other domain keys off `type`. A DNS rule carries both: `type` selects
 * the matcher shape ("default" / "logical") and `action` selects what happens
 * once it matches. They vary independently — a logical rule has an action just
 * like a default one — so this schema is bound with `typeKey: 'action'` and must
 * never touch `type`.
 *
 * WHAT THE OLD FORM COULD EDIT
 * ────────────────────────────
 * 3 of the 15 fields the four actions own, through a `v-if` chain:
 *
 *   route          server only — strategy, disable_cache, rewrite_ttl and
 *                  client_subnet had no control at all
 *   route-options  NOTHING. Its four fields ARE the whole struct, so selecting
 *                  this action produced an empty form body
 *   reject         method only; no_drop was listed in EDITED_FIELDS but never
 *                  rendered, so it was read into nothing and written back as
 *                  nothing — destroyed by any edit
 *   predefined     rcode only, which made a predefined ANSWER unauthorable —
 *                  the one thing the action exists for
 *
 * Four defects fell out of that, all of which this file plus `pruneForeignFields`
 * remove structurally rather than patch:
 *
 *   1. Switching route -> route-options carried `server` along, which
 *      DNSRouteOptionsActionOptions has no field for.
 *   2. `no_drop` was destroyed on every edit (above).
 *   3. Changing the action on a LOGICAL rule dropped `type`, `mode` and every
 *      nested rule, silently rewriting it as a flat default rule.
 *   4. Opening a `predefined` rule with no `rcode` seeded NXDOMAIN, but absent
 *      means NOERROR — so opening and saving flipped the rule's behaviour.
 *
 * CONSTRAINTS SING-BOX ONLY ENFORCES AT DECODE
 * ────────────────────────────────────────────
 * Three of these pass a naive form and fail the save. They are validated in
 * `validateDNSRuleAction` below rather than expressed as tiers, because a tier
 * decides what is SHOWN and none of these are about visibility:
 *
 *   - a `route-options` with every field empty is rejected outright
 *     ("empty DNS route option action", option/rule_action.go)
 *   - `no_drop` with `method: "drop"` is rejected ("no_drop is not available in
 *     current context")
 *   - a `route` with no `server` is the one genuinely required field here
 */
import {
  DNS_RULE_ACTION_INVENTORY,
  type DNSRuleActionFieldKey,
  type DNSRuleActionTypeName,
} from './dnsRuleActionInventory.generated'
import { createSchema, isFieldFilled, type FieldCuration } from './optionSchema'

/** Every field name across every action — so the shared map cannot hold a typo. */
type AnyDNSRuleActionFieldKey = {
  [T in DNSRuleActionTypeName]: DNSRuleActionFieldKey<T>
}[DNSRuleActionTypeName]

/**
 * sing-box's domain strategy vocabulary, from option.DomainStrategy's own
 * MarshalJSON (option/types.go). `as_is` is accepted on read but marshals back
 * as "", so it is deliberately absent: removing the field is the same thing and
 * does not leave a value that changes shape on save.
 */
const DOMAIN_STRATEGIES = ['prefer_ipv4', 'prefer_ipv6', 'ipv4_only', 'ipv6_only'] as const

/**
 * The complete RCODE set sing-box accepts, uppercase only. Carried over from the
 * hand-written list in DNSRules.vue, which was already verified against 1.12.12.
 */
const RCODES = ['NOERROR', 'NXDOMAIN', 'SERVFAIL', 'REFUSED', 'FORMERR', 'NOTIMP'] as const

/**
 * Curation applying to any action possessing the field.
 *
 * Keyed by field name and intersected with each action's real inventory rather
 * than spread into it: a spread bypasses TypeScript's excess-property check, so
 * `server` could be spread into `reject` — which has no such field — and fail
 * silently.
 */
const SHARED: Partial<Record<AnyDNSRuleActionFieldKey, FieldCuration>> = {
  // Shared by route and route-options, which is most of what route-options is
  // for. Promoted because overriding the resolution strategy per-rule is the
  // common reason to reach for either action.
  strategy: { tier: 'typical', order: 20, control: 'select', options: DOMAIN_STRATEGIES },

  disable_cache: { tier: 'advanced', order: 100 },
  rewrite_ttl: { tier: 'advanced', order: 110, placeholder: '60' },
  client_subnet: { tier: 'advanced', order: 120, placeholder: '1.2.3.0/24' },
}

/**
 * Per-action curation. Overrides SHARED for the same key.
 *
 * Typed as a mapped type over `DNSRuleActionTypeName`, so each entry is checked
 * against that action's own field keys — naming a field the action does not have
 * is a compile error, not an input that never binds.
 */
const BY_ACTION: {
  [T in DNSRuleActionTypeName]: Partial<Record<DNSRuleActionFieldKey<T>, FieldCuration>>
} = {
  route: {
    // The one genuinely required field across all four actions. A tag must be
    // picked rather than typed: naming a DNS server that does not exist makes
    // sing-box reject the whole config.
    server: { tier: 'core', order: 10, control: 'dns-server' },
  },

  // Its four fields are the entire struct, and sing-box rejects an all-empty
  // one, so `strategy` is promoted to core here — the action is unusable without
  // at least one of them set, and this is the one people come for.
  'route-options': {
    strategy: { tier: 'core', order: 10, control: 'select', options: DOMAIN_STRATEGIES },
    disable_cache: { tier: 'typical', order: 20 },
  },

  reject: {
    // sing-box accepts exactly two methods (constant/rule.go). The list once
    // also offered success/refused/nxdomain, which it rejects outright, so the
    // modal could save a config that passed panel validation and then refused to
    // start. To answer with a specific rcode, use `predefined`.
    method: {
      tier: 'core',
      order: 10,
      control: 'select',
      options: ['default', 'drop'],
      default: 'default',
    },
    // Only meaningful with method "default": sing-box errors on drop + no_drop.
    no_drop: { tier: 'advanced', order: 100, hintKey: 'dns.rules.form.fields.noDropHint' },
  },

  predefined: {
    // NOERROR, not NXDOMAIN. An absent rcode means NOERROR, and this default is
    // only ever applied to a NEW rule — seeding NXDOMAIN on open is what silently
    // changed existing rules.
    rcode: { tier: 'core', order: 10, control: 'select', options: RCODES, default: 'NOERROR' },

    // The records to answer with, as resource-record strings. Without these the
    // action can only return a bare rcode, which is half of what it is for.
    answer: {
      tier: 'typical',
      order: 20,
      control: 'chips',
      placeholder: 'a.example. 3600 IN A 192.0.2.1',
      hintKey: 'dns.rules.form.fields.answerHint',
    },
    ns: { tier: 'advanced', order: 100, control: 'chips' },
    extra: { tier: 'advanced', order: 110, control: 'chips' },
  },
}

const schema = createSchema<DNSRuleActionTypeName>({
  inventory: DNS_RULE_ACTION_INVENTORY,
  labelPrefix: 'dns.rules.form.fields',
  shared: SHARED as Record<string, FieldCuration>,
  byType: BY_ACTION as Partial<Record<DNSRuleActionTypeName, Record<string, FieldCuration>>>,
  // `type` on a DNS rule is the matcher shape, not the action. Binding it here
  // would make an action change rewrite a logical rule into a default one.
  typeKey: 'action',
  // A DNS rule has no `tag`, and its conditions must survive pruning — which
  // pruneForeignFields handles by construction, so nothing needs listing.
  identityKeys: ['action'],
})

export const resolveDNSRuleActionFields = schema.resolveFields
export const pruneForeignFields = schema.pruneForeignFields
export const applyActionDefaults = schema.applyTypeDefaults
export const isDNSRuleAction = schema.isKnownType

/**
 * The action of a rule as loaded from config.json.
 *
 * An omitted `action` means "route" — DNSRuleAction.UnmarshalJSONContext rewrites
 * "" to "route" before dispatching — so the form must show such a rule as a route
 * rule rather than as having no action.
 */
export function actionOf(rule: Readonly<Record<string, unknown>>): DNSRuleActionTypeName {
  const action = rule.action
  return isDNSRuleAction(action) ? action : 'route'
}

/**
 * The constraints sing-box enforces at decode time, which `sing-box check` only
 * reports as an opaque upstream string.
 *
 * Returns an i18n key, or undefined when the action is valid. Kept here rather
 * than in the component so it sits next to the curation that explains it.
 */
export function validateDNSRuleAction(
  rule: Readonly<Record<string, unknown>>,
): string | undefined {
  const action = actionOf(rule)

  if (action === 'route' && !isFieldFilled(rule.server)) {
    return 'dns.rules.form.errors.serverRequired'
  }

  if (action === 'route-options') {
    // "empty DNS route option action" — the struct's UnmarshalJSON rejects an
    // all-zero value, so a rule with the action selected and nothing set is a
    // hard failure rather than a no-op.
    const hasAny = (['strategy', 'disable_cache', 'rewrite_ttl', 'client_subnet'] as const).some(
      (key) => isFieldFilled(rule[key]),
    )
    if (!hasAny) return 'dns.rules.form.errors.routeOptionsEmpty'
  }

  if (action === 'reject' && rule.method === 'drop' && isFieldFilled(rule.no_drop)) {
    return 'dns.rules.form.errors.noDropWithDrop'
  }

  return undefined
}

export { DNS_RULE_ACTION_INVENTORY }
export type { DNSRuleActionTypeName }
export { DNS_RULE_ACTION_TYPE_NAMES } from './dnsRuleActionInventory.generated'
