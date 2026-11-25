import { expect, test } from 'bun:test'
import { dnsConditionKeys, dnsConditionTree, dnsRuleOutcome } from './dnsRulePresentation'
import { isTerminalAction } from '../schemas/dnsRuleActionFields'

test('DNS query and response conditions remain visible while action options are excluded', () => {
  const keys = dnsConditionKeys({ action: 'respond', match_response: 'cn', response_rcode: 'NOERROR',
    query_type: ['A', 'AAAA'], race: true, disable_cache: true, server: 'dns', invert: true })
  expect(keys).toEqual(['match_response', 'response_rcode', 'query_type'])
  expect(dnsConditionTree({ action: 'respond', match_response: 'cn', response_rcode: 'NOERROR' })).toEqual({
    kind: 'group', mode: 'and', children: [
      { kind: 'field', key: 'match_response', value: 'cn' },
      { kind: 'field', key: 'response_rcode', value: 'NOERROR' },
    ],
  })
})

test('DNS address alternatives are OR; query type and other families are AND', () => {
  expect(dnsConditionTree({ domain: 'a.example', domain_suffix: ['example.org'], ip_accept_any: true,
    query_type: 'A', server: 'dns' })).toEqual({
    kind: 'group', mode: 'and', children: [
      { kind: 'group', mode: 'or', children: [
        { kind: 'field', key: 'domain', value: 'a.example' },
        { kind: 'field', key: 'domain_suffix', value: ['example.org'] },
        { kind: 'field', key: 'ip_accept_any', value: true },
      ] },
      { kind: 'field', key: 'query_type', value: 'A' },
    ],
  })
})

test('logical DNS nesting and inversion use DNS condition keys at every level', () => {
  expect(dnsConditionTree({ type: 'logical', mode: 'or', invert: true,
    rules: [{ query_type: ['HTTPS'] }, { response_rcode: 'NXDOMAIN', match_response: 'upstream' }] })).toEqual({
    kind: 'group', mode: 'not', children: [{
      kind: 'group', mode: 'or', children: [
        { kind: 'field', key: 'query_type', value: ['HTTPS'] },
        { kind: 'group', mode: 'and', children: [
          { kind: 'field', key: 'response_rcode', value: 'NXDOMAIN' },
          { kind: 'field', key: 'match_response', value: 'upstream' },
        ] },
      ],
    }],
  })
})

test('future DNS matchers are retained and empty rules remain unconditional', () => {
  expect(dnsConditionTree({ future_matcher: 'value', action: 'reject' })).toEqual({
    kind: 'field', key: 'future_matcher', value: 'value',
  })
  expect(dnsConditionTree({ domain: [], query_dnssec: false, action: 'reject' })).toEqual({
    kind: 'group', mode: 'and', children: [],
  })
})

test('outcomes preserve default route, response status and evaluation behavior', () => {
  const t = (key: string, values?: Record<string, string>) => key + JSON.stringify(values ?? {})
  expect(dnsRuleOutcome({ server: 'dns-cn' }, t)).toContain('"server":"dns-cn"')
  expect(dnsRuleOutcome({ action: 'predefined' }, t)).not.toContain('NXDOMAIN')
  expect(dnsRuleOutcome({ action: 'predefined', rcode: 'NXDOMAIN' }, t)).toContain('NXDOMAIN')
  expect(dnsRuleOutcome({ action: 'evaluate', server: 'dns-cn', tag: 'cn' }, t)).toContain('"tag":"cn"')
  expect(dnsRuleOutcome({ action: 'respond', match_response: 'cn' }, t)).toContain('"tag":"cn"')
  expect(isTerminalAction('evaluate')).toBe(false)
  expect(isTerminalAction('respond')).toBe(true)
})
