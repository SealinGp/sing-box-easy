import { expect, test } from 'bun:test'
import { routeConditionGroups, routeConditionTree, routeRuleOutcome } from './routeRulePresentation'

test('default rules keep address and port alternatives together', () => {
  const groups = routeConditionGroups({ domain: ['a.example'], domain_suffix: 'example.org',
    ip_cidr: ['192.0.2.0/24'], port: 443, port_range: '8000:8080', network: 'tcp',
    source_ip_cidr: '10.0.0.0/8', source_ip_is_private: true,
    source_port: 1234, source_port_range: '9000:9002', outbound: 'proxy', invert: true })
  expect(groups.find(group => group.includes('domain'))).toEqual(['domain', 'domain_suffix', 'ip_cidr'])
  expect(groups.find(group => group.includes('port'))).toEqual(['port', 'port_range'])
  expect(groups.find(group => group.includes('source_ip_cidr'))).toEqual(['source_ip_cidr', 'source_ip_is_private'])
  expect(groups.find(group => group.includes('source_port'))).toEqual(['source_port', 'source_port_range'])
  expect(groups.flat()).not.toContain('outbound')
  expect(groups.flat()).not.toContain('invert')
  expect(groups).toContainEqual(['network'])
})

test('empty and false matchers do not turn unconditional rules into conditions', () => {
  expect(routeConditionGroups({ domain: [], ip_is_private: false, action: 'sniff' })).toEqual([])
  expect(routeConditionGroups({ rule_set: ['cn', 'private'] })).toEqual([['rule_set']])
})

test('an omitted action still describes routing and rejects retain their method', () => {
  const translate = (key: string, values?: Record<string, string>) => `${key}:${values?.outbound ?? ''}`
  expect(routeRuleOutcome({ outbound: '🤖 AI' }, translate)).toBe('route.rules.flow.then.route:🤖 AI')
  expect(routeRuleOutcome({ action: 'reject', method: 'drop' }, translate)).toBe('route.rules.flow.then.rejectDrop:')
})

test('nested groups keep NOT around the entire expression', () => {
  expect(routeConditionTree({ type: 'logical', mode: 'or', invert: true,
    rules: [{ domain: ['a.example'] }, { network: ['tcp'], port: [443] }] })).toEqual({
    kind: 'group', mode: 'not', children: [{ kind: 'group', mode: 'or', children: [
      { kind: 'field', key: 'domain', value: ['a.example'] },
      { kind: 'group', mode: 'and', children: [
        { kind: 'field', key: 'network', value: ['tcp'] },
        { kind: 'field', key: 'port', value: [443] },
      ] },
    ] }],
  })
})

test('mixed rule-set contents do not imply a known AND relationship', () => {
  const node = routeConditionTree({ domain: ['example.com'], rule_set: ['private'] })
  expect(node.kind === 'group' && node.mode).toBe('configured')
  expect(routeConditionTree({ rule_set: ['private'] })).toEqual({
    kind: 'field', key: 'rule_set', value: ['private'],
  })
})
