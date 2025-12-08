import type { ApiService } from './api'
import type { BasicResponse, RuleSet } from '../types/api'

export class TemplateService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getDefaultRuleSets(): Promise<BasicResponse<{ rule_sets: RuleSet[] }>> {
    const response = await this.api.get<BasicResponse<{ rule_sets: RuleSet[] }>>('/templates/rule-sets')
    return response.data
  }
}
