// Smart DNS sync — pure, testable logic shared by the Smart Routing Rule wizard.
//
// Problem it solves: in a "whitelist" sing-box setup, sending a domain to a
// proxied outbound is only half the job. If that domain's DNS still resolves
// through a poison-prone resolver, the connection dials a GFW-injected IP and
// hangs. This module decides which clean DNS server to recommend and how to
// reconcile the domain into `dns.rules` (append vs create) WITHOUT ever writing
// into a rule that carries extra AND-conditions (clash_mode/rule_set/…), which
// is exactly the trap that silently breaks such rules.

import type { DNSRule, DNSServer, Outbound } from '../types/api'

export type SmartMatchType = 'domain' | 'domain_suffix' | 'rule_set'

// Outbound types that resolve & dial locally — no proxy, so no pollution risk.
const NON_PROXIED_OUTBOUND_TYPES = ['direct', 'block', 'dns'] as const

// DNS server types that resist GFW poisoning (encrypted transports). A plain
// udp/tcp/local server can be poisoned unless it is tunneled (has a detour).
const CLEAN_DNS_TYPES = ['tls', 'https', 'h3', 'quic'] as const

// Every matcher field a sing-box DNS rule may carry. Used to prove a rule is
// "pure" for a single field before we append into it — see isPureFieldRule.
const MATCHER_FIELDS = [
  'inbound', 'ip_version', 'query_type', 'network', 'auth_user', 'protocol',
  'domain', 'domain_suffix', 'domain_keyword', 'domain_regex', 'geosite',
  'source_geoip', 'geoip', 'source_ip_cidr', 'source_ip_is_private', 'ip_cidr',
  'ip_is_private', 'source_port', 'source_port_range', 'port', 'port_range',
  'process_name', 'process_path', 'package_name', 'user', 'user_id', 'clash_mode',
  'wifi_ssid', 'wifi_bssid', 'rule_set', 'rule_set_ip_cidr_match_source',
  'invert', 'outbound',
] as const

function hasValue(v: unknown): boolean {
  if (v === undefined || v === null) return false
  if (Array.isArray(v)) return v.length > 0
  if (typeof v === 'string') return v.trim() !== ''
  if (typeof v === 'boolean') return v === true
  return true
}

function uniqClean(values: string[]): string[] {
  return Array.from(new Set(values.map((v) => v.trim()).filter(Boolean)))
}

/**
 * True when the outbound is proxied (anything that isn't direct/block/dns),
 * meaning its domains need clean DNS. An unknown tag errs toward `true` so we
 * prompt rather than silently risk pollution.
 */
export function isProxiedOutbound(tag: string | undefined, outbounds: Outbound[]): boolean {
  if (!tag) return false
  const ob = outbounds.find((o) => o.tag === tag)
  if (!ob) return true
  return !(NON_PROXIED_OUTBOUND_TYPES as readonly string[]).includes((ob as { type?: string }).type ?? '')
}

/** Encrypted transport or tunneled via a detour ⇒ resistant to poisoning. */
export function isCleanDnsServer(server: DNSServer): boolean {
  const s = server as { type?: string; detour?: string }
  if (s.type && (CLEAN_DNS_TYPES as readonly string[]).includes(s.type)) return true
  return !!s.detour
}

/**
 * A rule is "pure" for `field` when its ONLY matcher is that field and it's a
 * route action. Appending a suffix here is safe; appending into a rule that
 * also matches clash_mode/rule_set/etc. would inherit those AND-conditions and
 * quietly stop matching (the original Shopify bug).
 */
export function isPureFieldRule(rule: DNSRule, field: SmartMatchType): boolean {
  const r = rule as Record<string, unknown>
  if ((r.type as string) === 'logical') return false
  if (((r.action as string) || 'route') !== 'route') return false
  if (!hasValue(r[field])) return false
  return MATCHER_FIELDS.every((k) => k === field || !hasValue(r[k]))
}

/**
 * Recommend a DNS server for a new clean-resolution rule:
 *   1. the server already used by a pure rule on the same field, else
 *   2. the first encrypted/tunneled (clean) server, else
 *   3. the first server as a last resort.
 */
export function recommendDnsServer(
  field: SmartMatchType,
  dnsRules: DNSRule[],
  servers: DNSServer[],
): string | undefined {
  const pure = dnsRules.find((r) => isPureFieldRule(r, field) && hasValue((r as Record<string, unknown>).server))
  if (pure) return (pure as Record<string, unknown>).server as string
  return (servers.find(isCleanDnsServer) ?? servers[0])?.tag
}

export interface DnsReconcileResult {
  /** append → patch existing rule at `index`; create → add `rule`; noop → nothing new. */
  op: 'append' | 'create' | 'noop'
  index?: number
  rule: DNSRule
  addedValues: string[]
}

/**
 * Decide how to map `values` (under `field`) → a clean `server` into dns.rules.
 * Appends into a pure same-field/same-server rule when one exists; otherwise
 * builds a fresh rule, mirroring the strategy of a sibling pure rule (default
 * ipv4_only). Never mutates its inputs.
 */
export function reconcileDnsRule(
  field: SmartMatchType,
  values: string[],
  server: string,
  dnsRules: DNSRule[],
  opts?: { strategy?: string },
): DnsReconcileResult {
  const clean = uniqClean(values)

  const index = dnsRules.findIndex(
    (r) => isPureFieldRule(r, field) && (r as Record<string, unknown>).server === server,
  )

  if (index >= 0) {
    const existing = dnsRules[index] as Record<string, unknown>
    const current = (existing[field] as string[]) ?? []
    const added = clean.filter((v) => !current.includes(v))
    const rule = { ...existing, [field]: [...current, ...added] } as DNSRule
    return { op: added.length ? 'append' : 'noop', index, rule, addedValues: added }
  }

  const sibling = dnsRules.find((r) => isPureFieldRule(r, field)) as Record<string, unknown> | undefined
  const strategy = opts?.strategy ?? (sibling?.strategy as string | undefined) ?? 'ipv4_only'
  const rule: Record<string, unknown> = { [field]: clean, server }
  if (strategy) rule.strategy = strategy
  return { op: 'create', rule: rule as DNSRule, addedValues: clean }
}
