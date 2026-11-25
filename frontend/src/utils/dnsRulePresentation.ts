import { ALL_ACTION_KEYS, actionOf } from '../schemas/dnsRuleActionFields'
import { routeConditionTree } from './routeRulePresentation'

const STRUCTURAL_KEYS = ['action', 'type', 'mode', 'rules', 'invert']
export function dnsConditionKeys(rule: Readonly<Record<string, unknown>>): string[] {
  return Object.keys(rule).filter(key => !ALL_ACTION_KEYS.includes(key) && !STRUCTURAL_KEYS.includes(key))
}

// DNS groups differ from route groups: ip_accept_any belongs to the destination family.
const DNS_ALTERNATIVES = [
  ['domain', 'domain_suffix', 'domain_keyword', 'domain_regex', 'geosite', 'geoip', 'ip_cidr', 'ip_is_private', 'ip_accept_any'],
  ['port', 'port_range'],
  ['source_geoip', 'source_ip_cidr', 'source_ip_is_private'],
  ['source_port', 'source_port_range'],
]
export function dnsConditionTree(rule: Record<string, unknown>) {
  return routeConditionTree(rule, { keys: dnsConditionKeys, alternatives: DNS_ALTERNATIVES })
}

export function dnsRuleOutcome(r: Record<string, unknown>, t: (key: string, values?: Record<string, string>) => string): string {
  switch (actionOf(r)) {
    case 'route':
      return r.server
        ? t('dns.rules.flow.then.route', { server: String(r.server) })
        : t('dns.rules.flow.then.routeIncomplete')
    case 'route-options':
      return t('dns.rules.flow.then.routeOptions')
    case 'reject':
      return r.method === 'drop'
        ? t('dns.rules.flow.then.rejectDrop')
        : t('dns.rules.flow.then.reject')
    case 'predefined':
      return r.rcode
        ? t('dns.rules.flow.then.predefinedRcode', { rcode: String(r.rcode) })
        : t('dns.rules.flow.then.predefined')
    case 'evaluate':
      return r.server
        ? t('dns.rules.flow.then.evaluate', { server: String(r.server), tag: String(r.tag || '') })
        : t('dns.rules.flow.then.evaluateIncomplete')
    case 'respond':
      return t('dns.rules.flow.then.respond', { tag: String(r.match_response || '') })
    default:
      return actionOf(r)
  }
}
