import { isFieldFilled } from '../schemas/optionSchema'
import { dnsConditionKeys } from './dnsRulePresentation'
import { readQueryTypes, writeQueryTypes } from './dnsQueryType'

export type DNSRuleDraft = Record<string, any>
const conditions = ['rule_set', 'domain', 'domain_suffix', 'domain_keyword', 'geosite']
const records = ['answer', 'ns', 'extra']
const list = (value: unknown): unknown[] => value == null ? [] : Array.isArray(value) ? value : [value]

/** Detach every nested value from the live configuration before editing. */
export function readDNSRuleDraft(rule: DNSRuleDraft): DNSRuleDraft {
  const draft: DNSRuleDraft = JSON.parse(JSON.stringify(rule))
  if (draft.type === 'logical') {
    draft.rules = (draft.rules ?? []).map(readDNSRuleDraft)
  } else {
    for (const key of conditions) draft[key] = list(draft[key])
    draft.query_type = readQueryTypes(draft.query_type)
  }
  for (const key of records) if (key in draft) draft[key] = list(draft[key])
  return draft
}

/** Normalize only known form fields, retaining unknown options at every depth. */
export function writeDNSRuleDraft(draft: DNSRuleDraft): DNSRuleDraft {
  const rule = { ...draft }
  if (Array.isArray(rule.rules)) rule.rules = rule.rules.map(writeDNSRuleDraft)
  if (Array.isArray(rule.query_type)) rule.query_type = writeQueryTypes(rule.query_type)
  for (const [key, value] of Object.entries(rule)) {
    if (value == null || value === '' || (Array.isArray(value) && !value.length)) delete rule[key]
  }
  return rule
}

export function invalidDNSConditionGroup(rule: DNSRuleDraft, child = false): boolean {
  if (rule.type !== 'logical') return child && !dnsConditionKeys(rule).some(key => isFieldFilled(rule[key]))
  return !['and', 'or'].includes(rule.mode) || !Array.isArray(rule.rules) || !rule.rules.length || rule.rules.some((nested: DNSRuleDraft) => invalidDNSConditionGroup(nested, true))
}
