import { describe, expect, it } from 'bun:test'
import { managedPolicyDraft } from './managedPolicyDraft'
import type { ManagedPolicy, ManagedSnapshot } from '../services/trafficPolicy'

describe('managedPolicyDraft', () => {
  it('keeps edited blocks at their current position after other policies were inserted', () => {
    const policy: ManagedPolicy = { id: 'cn', name: 'CN', group_id: 'local', outbound: 'direct', mode: 'or',
      conditions: [{ kind: 'rule_set', values: ['cn'] }], route_index: 0, dns_index: 0,
      route_members: [{ rule_set: ['cn'], action: 'route', outbound: 'direct' }],
      dns_members: [{ action: 'evaluate', server: 'local', tag: 'cn_1' }, { action: 'respond', match_response: 'cn_1' }] }
    const snapshot: ManagedSnapshot = { revision: 'test', groups: [], policies: [], servers: [], outbounds: [], rule_sets: [], route_rules: [{ action: 'sniff' }, { action: 'route', outbound: 'direct', rule_set: ['cn'] }],
      dns_rules: [{ action: 'route', server: 'hosts' }, {}, ...policy.dns_members!] }
    const draft = managedPolicyDraft(policy, snapshot)
    expect(draft.route_index).toBe(1)
    expect(draft.dns_index).toBe(2)
    draft.conditions[0]!.values.push('extra')
    expect(policy.conditions[0]!.values).toEqual(['cn'])
    expect(policy.route_index).toBe(0)
  })

  it('preserves the repair position for missing or ambiguous blocks', () => {
    const policy = { route_index: 4, dns_index: 7, route_members: [{ action: 'route' }], dns_members: [] } as unknown as ManagedPolicy
    const snapshot = { route_rules: [{ action: 'route' }, { action: 'route' }], dns_rules: [] } as unknown as ManagedSnapshot
    expect(managedPolicyDraft(policy, snapshot).route_index).toBe(4)
    expect(managedPolicyDraft(policy, snapshot).dns_index).toBe(7)
  })
})
