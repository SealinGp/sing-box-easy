import { expect, test } from 'bun:test'
import { readDNSRuleDraft, writeDNSRuleDraft, invalidDNSConditionGroup } from './dnsRuleDraft'

const logical = {
  action: 'predefined', mode: 'and', rcode: 'NOERROR', type: 'logical',
  rules: [{ query_type: ['AAAA', 'HTTPS'] }, { rule_set: ['geosite-netflix', 'geosite-youtube', 'sea-rulesets-ai'] }],
}

test('logical DNS rule round trips both child conditions and the predefined action', () => {
  expect(writeDNSRuleDraft(readDNSRuleDraft(logical))).toEqual(logical)
})
test('editing a child cannot mutate the loaded configuration or its sibling', () => {
  const draft = readDNSRuleDraft(logical)
  draft.rules[0].query_type.splice(0, 1)
  draft.rules[1].rule_set.push('custom-1')
  expect(logical.rules[0]!.query_type).toEqual(['AAAA', 'HTTPS'])
  expect(logical.rules[1]!.rule_set).toHaveLength(3)
  expect(writeDNSRuleDraft(draft).rules).toEqual([
    { query_type: ['HTTPS'] }, { rule_set: ['geosite-netflix', 'geosite-youtube', 'sea-rulesets-ai', 'custom-1'] },
  ])
})
test('nested groups preserve modes, inversion, unknown fields and numeric query types', () => {
  const raw = { type: 'logical', mode: 'or', invert: true, rules: [
    { type: 'logical', mode: 'and', rules: [{ query_type: 32768, future_matcher: { values: [1, 2] } }] },
    { domain_suffix: 'example.test', invert: true },
  ], action: 'predefined', answer: 'example.test. 60 IN A 192.0.2.1' }
  const draft = readDNSRuleDraft(raw)
  expect(draft.rules[0].rules[0].query_type).toEqual(['32768'])
  expect(writeDNSRuleDraft(draft)).toEqual({ ...raw, answer: [raw.answer], rules: [
    { type: 'logical', mode: 'and', rules: [{ query_type: [32768], future_matcher: { values: [1, 2] } }] },
    { domain_suffix: ['example.test'], invert: true },
  ] })
})
test('invalid nested groups block saving but ordinary catch-all conditions remain valid', () => {
  expect(invalidDNSConditionGroup(logical)).toBe(false)
  expect(invalidDNSConditionGroup({ type: 'logical', mode: 'and', rules: [] })).toBe(true)
  expect(invalidDNSConditionGroup({ type: 'logical', mode: 'xor', rules: [{}] })).toBe(true)
  expect(invalidDNSConditionGroup({ type: 'logical', mode: 'or', rules: [{ type: 'logical', mode: 'and', rules: [] }] })).toBe(true)
  expect(invalidDNSConditionGroup({})).toBe(false)
})

test('empty or inversion-only children cannot be saved as DNS matchers', () => {
  for (const child of [{}, { invert: true }, { query_type: [], domain: [] }]) {
    expect(invalidDNSConditionGroup({ type: 'logical', mode: 'and', rules: [child] })).toBe(true)
  }
  expect(invalidDNSConditionGroup({ type: 'logical', mode: 'or', rules: [{ ip_is_private: true }] })).toBe(false)
})
