import { describe, expect, test } from 'bun:test'
import { buildTrafficPolicyConfig, type TrafficPolicyConfigSpec } from './trafficPolicyConfig'
const spec: TrafficPolicyConfigSpec = { conditions: [{ kind: 'domain_suffix', value: 'xx.com' }], mode: 'or', outbound: 'Auto-select', servers: ['GoogleDoT', 'dns_google'], timeout: 3, fallback: 'strict', tagPrefix: 'policy_test' }
describe('traffic policy configuration preview', () => {
  test('launches all evaluates before race responses and scoped strict rejection', () => {
    const config = buildTrafficPolicyConfig(spec)
    expect(config.route.rules[0]).toEqual({ domain_suffix: ['xx.com'], action: 'route', outbound: 'Auto-select' })
    expect(config.dns!.rules.map(r => r.action)).toEqual(['evaluate', 'evaluate', 'respond', 'respond', 'reject'])
    expect(config.dns!.rules[2]).toEqual({ match_response: 'policy_test_1', response_rcode: 'NOERROR', action: 'respond', race: true })
    expect(config.dns!.rules[4]).toMatchObject({ domain_suffix: ['xx.com'], no_drop: true })
  })
  test('uses explicit logical children for both any and all; action stays on parent', () => {
    for (const mode of ['and', 'or'] as const) {
      const config = buildTrafficPolicyConfig({ ...spec, mode, conditions: [...spec.conditions, {kind:'rule_set',value:'sample-set'}] })
      expect(config.route.rules[0]).toMatchObject({type:'logical', mode, rules:[{domain_suffix:['xx.com']},{rule_set:['sample-set']}]})
      expect(config.dns!.rules[0]).toMatchObject({type:'logical', mode, action:'evaluate'})
    }
  })
  test('general fallback omits guard; single resolver uses no race', () => {
    const config = buildTrafficPolicyConfig({...spec, servers:['GoogleDoT'],fallback:'general'})
    expect(config.dns!.rules).toHaveLength(2)
    expect(config.dns!.rules[1]).not.toHaveProperty('race')
  })
  test('IP conditions and unchanged DNS produce no DNS modifications', () => {
    expect(buildTrafficPolicyConfig({...spec,conditions:[{kind:'ip_cidr',value:'192.0.2.0/24'}]})).not.toHaveProperty('dns')
    expect(buildTrafficPolicyConfig({...spec,servers:[]})).not.toHaveProperty('dns')
  })
  test('deduplicates resolvers and rejects incomplete drafts', () => {
    expect(buildTrafficPolicyConfig({...spec,servers:['GoogleDoT','GoogleDoT']}).dns!.rules).toHaveLength(3)
    expect(() => buildTrafficPolicyConfig({...spec,conditions:[]})).toThrow()
    expect(() => buildTrafficPolicyConfig({...spec,conditions:[{kind:'unknown',value:'x'}]})).toThrow()
  })
})

describe('multi-value matchers', () => {
  test('parses editable lists, trims and deduplicates values in every emitted match', () => {
    const config = buildTrafficPolicyConfig({...spec, conditions:[{kind:'domain_suffix',value:'xx.com, 1.com\nxx.com\n'}]})
    for (const rule of [config.route.rules[0], config.dns!.rules[0], config.dns!.rules[1], config.dns!.rules[4]]) {
      expect(rule).toHaveProperty('domain_suffix', ['xx.com','1.com'])
      expect(rule).not.toHaveProperty('type')
    }
  })
  test('compacts same-matcher OR rows, retaining distinct matcher branches', () => {
    const conditions = [{kind:'domain_suffix',value:'xx.com'}, {kind:'domain_suffix',value:'1.com,xx.com'}]
    expect(buildTrafficPolicyConfig({...spec,conditions}).route.rules[0]).toHaveProperty('domain_suffix',['xx.com','1.com'])
    expect(buildTrafficPolicyConfig({...spec,conditions:[...conditions,{kind:'rule_set',value:'set-a\nset-b'}]}).route.rules[0]).toMatchObject({type:'logical',mode:'or',rules:[{domain_suffix:['xx.com','1.com']},{rule_set:['set-a','set-b']}]})
  })
  test('never merges separate AND rows into an OR array', () => {
    expect(buildTrafficPolicyConfig({...spec,mode:'and',conditions:[{kind:'domain_suffix',value:'example.com,example.org'},{kind:'domain_suffix',value:'sub.example.com'}]}).route.rules[0]).toMatchObject({type:'logical',mode:'and',rules:[{domain_suffix:['example.com','example.org']},{domain_suffix:['sub.example.com']}]})
  })
  test('rejects separator-only input', () => {
    expect(() => buildTrafficPolicyConfig({...spec,conditions:[{kind:'domain_suffix',value:' , \n , '}]})).toThrow()
  })
})
