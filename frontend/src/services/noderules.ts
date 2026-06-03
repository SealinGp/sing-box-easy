import type { ApiService } from './api'
import type { BasicResponse } from '../types/api'
import type {
  Filter,
  Group,
  FilterInput,
  GroupInput,
  KeywordEntry,
  FilterTemplate,
  PreviewResult,
} from '../types/noderules'

export class NodeRulesService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getAll(): Promise<BasicResponse<{ filters: Filter[]; groups: Group[] }>> {
    const res = await this.api.get<BasicResponse<{ filters: Filter[]; groups: Group[] }>>('/node-rules')
    return res.data
  }

  async getKeywords(): Promise<BasicResponse<{ keywords: KeywordEntry[] }>> {
    const res = await this.api.get<BasicResponse<{ keywords: KeywordEntry[] }>>('/node-rules/keywords')
    return res.data
  }

  async getTemplates(): Promise<BasicResponse<{ templates: FilterTemplate[] }>> {
    const res = await this.api.get<BasicResponse<{ templates: FilterTemplate[] }>>('/node-rules/templates')
    return res.data
  }

  async applyTemplate(id: string): Promise<BasicResponse<Filter>> {
    const res = await this.api.post<BasicResponse<Filter>>(`/node-rules/templates/${id}/apply`)
    return res.data
  }

  async createFilter(input: FilterInput): Promise<BasicResponse<Filter>> {
    const res = await this.api.post<BasicResponse<Filter>>('/node-rules/filters', input)
    return res.data
  }

  async updateFilter(id: string, input: FilterInput): Promise<BasicResponse<Filter>> {
    const res = await this.api.put<BasicResponse<Filter>>(`/node-rules/filters/${id}`, input)
    return res.data
  }

  async deleteFilter(id: string): Promise<BasicResponse<{ message: string; id: string }>> {
    const res = await this.api.delete<BasicResponse<{ message: string; id: string }>>(`/node-rules/filters/${id}`)
    return res.data
  }

  async createGroup(input: GroupInput): Promise<BasicResponse<Group>> {
    const res = await this.api.post<BasicResponse<Group>>('/node-rules/groups', input)
    return res.data
  }

  async updateGroup(id: string, input: GroupInput): Promise<BasicResponse<Group>> {
    const res = await this.api.put<BasicResponse<Group>>(`/node-rules/groups/${id}`, input)
    return res.data
  }

  async deleteGroup(id: string): Promise<BasicResponse<{ message: string; id: string }>> {
    const res = await this.api.delete<BasicResponse<{ message: string; id: string }>>(`/node-rules/groups/${id}`)
    return res.data
  }

  async preview(): Promise<BasicResponse<PreviewResult>> {
    const res = await this.api.post<BasicResponse<PreviewResult>>('/node-rules/preview')
    return res.data
  }

  async apply(): Promise<BasicResponse<{ message: string; endpoints: number; filters: number; groups: number; unmatched: number }>> {
    const res = await this.api.post<BasicResponse<{ message: string; endpoints: number; filters: number; groups: number; unmatched: number }>>('/node-rules/apply')
    return res.data
  }
}
