import type { ApiService } from './api'
import type { BasicResponse, RouteRule, RuleSet } from '../types/api'

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

  async deleteRuleSet(tag: string): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.delete<BasicResponse<{ message: string; tag: string }>>(`/route/rule-sets/${tag}`)
    return response.data
  }

  async getRouteFinal(): Promise<BasicResponse<{ final: string }>> {
    const response = await this.api.get<BasicResponse<{ final: string }>>('/route/final')
    return response.data
  }

  async updateRouteFinal(final: string): Promise<BasicResponse<{ message: string; final: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string; final: string }>>('/route/final', { final })
    return response.data
  }
}
