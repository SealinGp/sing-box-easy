export interface TrafficPolicyConfigSpec {
  conditions: { kind: string; value: string }[]
  mode: 'and' | 'or'
  outbound: string
  servers: string[]
  timeout: number
  fallback: 'strict' | 'general'
  tagPrefix: string
}
type Rule = Record<string, unknown>
/** Keep pasted text editable; separators become array entries only in output. */
export function matchValues(value: string): string[] {
  return [...new Set(value.split(/[,\n\r]+/).map(part => part.trim()).filter(Boolean))]
}
/** Proposed additions only: not a whole config and not a plan against live rules. */
export function buildTrafficPolicyConfig(spec: TrafficPolicyConfigSpec) {
  const allowed = new Set(['domain', 'domain_suffix', 'rule_set', 'ip_cidr'])
  if (!spec.conditions.length || spec.conditions.some(c => !allowed.has(c.kind) || !matchValues(c.value).length)) throw new Error('Complete all match conditions.')
  if (!spec.outbound.trim()) throw new Error('Choose an outbound.')
  if (!Number.isInteger(spec.timeout) || spec.timeout < 1 || spec.timeout > 30) throw new Error('Invalid timeout.')
  if (!/^[a-zA-Z0-9_-]+$/.test(spec.tagPrefix)) throw new Error('Invalid response tag prefix.')
  // Values within one matcher are alternatives. Only OR rows of the same
  // matcher may be combined; merging AND rows would broaden their meaning.
  const rows = spec.conditions.map(c => ({ kind: c.kind, values: matchValues(c.value) }))
  const grouped = spec.mode === 'or' ? [...new Set(rows.map(c => c.kind))].map(kind => ({
    kind, values: [...new Set(rows.filter(c => c.kind === kind).flatMap(c => c.values))],
  })) : rows
  const children = grouped.map(c => ({ [c.kind]: c.values }))
  const match: Rule = children.length === 1 ? children[0]! : { type: 'logical', mode: spec.mode, rules: children }
  const route: { rules: Rule[] } = { rules: [{ ...match, action: 'route', outbound: spec.outbound }] }
  // IP conditions cannot be copied into a pre-resolution DNS selection.
  if (!spec.servers.length || spec.conditions.some(c => c.kind === 'ip_cidr')) return { route }
  const servers = [...new Set(spec.servers)]
  const tags = servers.map((_, i) => `${spec.tagPrefix}_${i + 1}`)
  const dnsRules: Rule[] = servers.map((server, i) => ({ ...match, action: 'evaluate', server, tag: tags[i], timeout: `${spec.timeout}s` }))
  for (const tag of tags) dnsRules.push({ match_response: tag, response_rcode: 'NOERROR', action: 'respond', ...(servers.length > 1 ? { race: true } : {}) })
  // A matched rejection waits behind armed race rules, then prevents general fallback.
  if (spec.fallback === 'strict') dnsRules.push({ ...match, action: 'reject', method: 'default', no_drop: true })
  return { route, dns: { rules: dnsRules } }
}
