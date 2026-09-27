/**
 * Route rule matchers understood by the editor across supported core versions.
 *
 * Same overlay pattern as `dnsRuleActionInventory.ts`: the generated inventory
 * is pinned to the Go library (sing-box 1.12), while the installed binary is
 * routinely newer. Matchers a newer core ADDED live here until the dependency
 * is bumped and the schema regenerated — at which point the generated entry
 * takes over and the overlay line can be deleted.
 *
 * An older core fails config decode on these with `unknown field`, so each one
 * is gated on a capability the BACKEND derives from the installed binary
 * (`core.CapabilitiesForCoreVersion`, served by GET /core) — see
 * `MATCHER_CAPABILITIES` in routeRuleMatcherFields.ts. The version rule lives in
 * one place, next to the other version gates, rather than being re-derived here.
 *
 * The backend needs nothing for these: route rules are stored as raw JSON and
 * validated by the installed `sing-box check`.
 */
import { ROUTE_RULE_MATCHER_INVENTORY as GENERATED } from './routeRuleMatcherInventory.generated'
import type { OptionFieldInfo } from './optionSchema'

/**
 * https://sing-box.sagernet.org/configuration/route/rule/#source_mac_address
 *
 * Both are resolved from the host's neighbour table, so they only mean
 * something for clients on a network sing-box can see at layer 2 — the router
 * case this panel mostly runs in (OpenWrt, tun/tproxy on the LAN gateway).
 */
export const ROUTE_RULE_MATCHER_OVERLAY = {
  source_mac_address: { kind: 'list', item: 'string' },
  source_hostname: { kind: 'list', item: 'string' },
} as const satisfies Record<string, OptionFieldInfo>

export const ROUTE_RULE_MATCHER_INVENTORY = {
  default: {
    ...GENERATED.default,
    ...ROUTE_RULE_MATCHER_OVERLAY,
  },
} as const satisfies Record<string, Record<string, OptionFieldInfo>>

export type RouteRuleMatcherTypeName = keyof typeof ROUTE_RULE_MATCHER_INVENTORY

export type RouteRuleMatcherFieldKey<T extends RouteRuleMatcherTypeName> = Extract<
  keyof (typeof ROUTE_RULE_MATCHER_INVENTORY)[T],
  string
>

/**
 * Every list-shaped matcher key. sing-box accepts a scalar OR an array for all
 * of them on the wire (`"source_mac_address": "00:11:…"` is a one-element
 * list), so the loader coerces exactly these — derived here so a new list
 * matcher is covered without a hand-kept list.
 */
export const LIST_MATCHER_KEYS = (
  Object.entries(ROUTE_RULE_MATCHER_INVENTORY.default) as [string, OptionFieldInfo][]
)
  .filter(([, info]) => info.kind === 'list')
  .map(([key]) => key)
