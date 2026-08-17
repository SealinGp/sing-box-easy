import type { ApiService } from './api'
import type { BasicResponse, RouteRule, RuleSet } from '../types/api'

// One rule that references a rule-set tag, and how a cascade delete changes it.
export interface RuleSetReference {
  scope: 'route' | 'dns'
  index: number
  action: 'strip' | 'delete'
  rule_set: string[]
}

export class RouteService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getRouteRules(): Promise<BasicResponse<{ rules: RouteRule[] }>> {
    const response = await this.api.get<BasicResponse<{ rules: RouteRule[] }>>('/route/rules')
    return response.data
  }

  async addRouteRule(rule: RouteRule): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string }>>('/route/rules', rule)
    return response.data
  }

  // `order` is a permutation of the CURRENT indices, in the order the rules
  // should end up in — the rule bodies never leave the server, so a reorder
  // cannot lose a field the form does not know about.
  async reorderRouteRules(order: number[]): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/route/rules', { order })
    return response.data
  }

  async updateRouteRule(index: number, rule: RouteRule): Promise<BasicResponse<{ message: string; index: number }>> {
    const response = await this.api.put<BasicResponse<{ message: string; index: number }>>(`/route/rules/${index}`, rule)
    return response.data
  }

  async deleteRouteRule(index: number): Promise<BasicResponse<{ message: string; index: number }>> {
    const response = await this.api.delete<BasicResponse<{ message: string; index: number }>>(`/route/rules/${index}`)
    return response.data
  }

  async getRuleSets(): Promise<BasicResponse<{ rule_sets: RuleSet[] }>> {
    const response = await this.api.get<BasicResponse<{ rule_sets: RuleSet[] }>>('/route/rule-sets')
    return response.data
  }

  async addRuleSet(ruleSet: RuleSet): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string; tag: string }>>('/route/rule-sets', ruleSet)
    return response.data
  }

  async getRuleSetByTag(tag: string): Promise<BasicResponse<RuleSet>> {
    const response = await this.api.get<BasicResponse<RuleSet>>(`/route/rule-sets/${tag}`)
    return response.data
  }

  async updateRuleSet(tag: string, ruleSet: RuleSet): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string; tag: string }>>(`/route/rule-sets/${tag}`, ruleSet)
    return response.data
  }

  // Dry-run: how would deleting `tag` affect route.rules / dns.rules?
  async getRuleSetReferences(tag: string): Promise<
    BasicResponse<{ tag: string; references: RuleSetReference[]; route_count: number; dns_count: number }>
  > {
    const response = await this.api.get<
      BasicResponse<{ tag: string; references: RuleSetReference[]; route_count: number; dns_count: number }>
    >(`/route/rule-sets/${tag}/references`)
    return response.data
  }

  // cascade=true also scrubs the tag from route.rules / dns.rules matchers.
  async deleteRuleSet(
    tag: string,
    opts?: { cascade?: boolean },
  ): Promise<BasicResponse<{ message: string; tag: string }>> {
    const query = opts?.cascade ? '?cascade=true' : ''
    const response = await this.api.delete<BasicResponse<{ message: string; tag: string }>>(
      `/route/rule-sets/${tag}${query}`,
    )
    return response.data
  }

  async getRouteFinal(): Promise<
    BasicResponse<{ final: string; auto_detect_interface: boolean; default_domain_resolver: string }>
  > {
    const response = await this.api.get<
      BasicResponse<{ final: string; auto_detect_interface: boolean; default_domain_resolver: string }>
    >('/route/final')
    return response.data
  }

  async updateRouteFinal(final: string): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/route/final', { final })
    return response.data
  }

  // updateRouteSettings patches any subset of route-level policy fields. Omitted
  // fields are left unchanged on the server (partial update).
  async updateRouteSettings(payload: {
    final?: string
    auto_detect_interface?: boolean
    default_domain_resolver?: string
  }): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/route/final', payload)
    return response.data
  }
}
