import { ALL_MATCHER_KEYS } from '../schemas/routeRuleMatcherFields'
import { isFieldFilled } from '../schemas/optionSchema'
import { actionOf } from '../schemas/routeRuleActionFields'

// sing-box combines alternatives inside each of these address/port families.
const ALTERNATIVES = [
  ['domain', 'domain_suffix', 'domain_keyword', 'domain_regex', 'geosite', 'geoip', 'ip_cidr', 'ip_is_private'],
  ['port', 'port_range'],
  ['source_geoip', 'source_ip_cidr', 'source_ip_is_private'],
  ['source_port', 'source_port_range'],
]

interface ConditionOptions {
  keys: (rule: Record<string, unknown>) => readonly string[]
  alternatives: string[][]
}
const ROUTE_OPTIONS: ConditionOptions = { keys: () => ALL_MATCHER_KEYS, alternatives: ALTERNATIVES }

export function routeConditionGroups(rule: Record<string, unknown>, options = ROUTE_OPTIONS): string[][] {
  const keys: string[] = options.keys(rule).filter(key => key !== 'invert' && isFieldFilled(rule[key]))
  const emitted = new Set<string>()
  return keys.flatMap(key => {
    if (emitted.has(key)) return []
    const family = options.alternatives.find(group => group.includes(key)) ?? [key]
    const group = family.filter(member => keys.includes(member))
    group.forEach(member => emitted.add(member))
    return [group]
  })
}

export type RoutingCondition =
  | { kind: 'field'; key: string; value: unknown }
  | { kind: 'group'; mode: 'and' | 'or' | 'not' | 'configured'; children: RoutingCondition[] }

/** Preserve parentheses and inversion when presenting a rule as a sentence. */
export function routeConditionTree(rule: Record<string, unknown>, options = ROUTE_OPTIONS): RoutingCondition {
  let node: RoutingCondition
  if (rule.type === 'logical') {
    node = { kind: 'group', mode: rule.mode === 'or' ? 'or' : 'and',
      children: (Array.isArray(rule.rules) ? rule.rules : []).map(child => routeConditionTree(child, options)) }
  } else {
    const children: RoutingCondition[] = routeConditionGroups(rule, options).map(keys => {
      const fields: RoutingCondition[] = keys.map(key => ({ kind: 'field', key, value: rule[key] }))
      return fields.length === 1 ? fields[0]! : { kind: 'group', mode: 'or', children: fields }
    })
    // A single default rule inside a rule-set may merge its matcher families
    // into the outer rule. Without decoded set contents, don't invent AND.
    const mixedRuleSet = children.length > 1 && isFieldFilled(rule.rule_set)
    node = children.length === 1 ? children[0]! : {
      kind: 'group', mode: mixedRuleSet ? 'configured' : 'and', children,
    }
  }
  return rule.invert ? { kind: 'group', mode: 'not', children: [node] } : node
}

type Translate = (key: string, values?: Record<string, string>) => string

export function routeRuleOutcome(rule: Record<string, unknown>, t: Translate): string {
  switch (actionOf(rule)) {
    case 'route': return rule.outbound
      ? t('route.rules.flow.then.route', { outbound: String(rule.outbound) })
      : t('route.rules.flow.then.routeIncomplete')
    case 'reject': return t(rule.method === 'drop' ? 'route.rules.flow.then.rejectDrop' : 'route.rules.flow.then.reject')
    case 'resolve': return rule.server
      ? t('route.rules.flow.then.resolveWith', { server: String(rule.server) })
      : t('route.rules.flow.then.resolve')
    case 'route-options': return t('route.rules.flow.then.routeOptions')
    case 'direct': return t('route.rules.flow.then.direct')
    case 'hijack-dns': return t('route.rules.flow.then.hijackDns')
    case 'sniff': return t('route.rules.flow.then.sniff')
    default: return actionOf(rule)
  }
}
