import type { ApiService } from './api'
import type { BasicResponse, Outbound } from '../types/api'

export class OutboundService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getOutbounds(): Promise<BasicResponse<{ outbounds: Outbound[] }>> {
    const response = await this.api.get<BasicResponse<{ outbounds: Outbound[] }>>('/outbounds')
    return response.data
  }

  async addOutbound(outbound: Outbound): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string; tag: string }>>('/outbounds', outbound)
    return response.data
  }

  async addOutboundsBatch(outbounds: Outbound[]): Promise<BasicResponse<{
    total: number
    added: number
    skipped: number
    results: Array<{ tag: string; added: boolean; reason?: string }>
  }>> {
    const response = await this.api.post<BasicResponse<{
      total: number
      added: number
      skipped: number
      results: Array<{ tag: string; added: boolean; reason?: string }>
    }>>('/outbounds/batch', { outbounds })
    return response.data
  }

  async getOutboundGroups(): Promise<BasicResponse<{ groups: Outbound[] }>> {
    const response = await this.api.get<BasicResponse<{ groups: Outbound[] }>>('/outbounds/groups')
    return response.data
  }

  async getOutboundByTag(tag: string): Promise<BasicResponse<Outbound>> {
    const response = await this.api.get<BasicResponse<Outbound>>(`/outbounds/${tag}`)
    return response.data
  }

  async updateOutbound(tag: string, outbound: Outbound): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string; tag: string }>>(`/outbounds/${tag}`, outbound)
    return response.data
  }

  async deleteOutbound(tag: string): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.delete<BasicResponse<{ message: string; tag: string }>>(`/outbounds/${tag}`)
    return response.data
  }

  async deleteOutboundsBatch(tags: string[]): Promise<BasicResponse<{
    message: string
    deleted_count: number
  }>> {
    const response = await this.api.delete<BasicResponse<{
      message: string
      deleted_count: number
    }>>('/outbounds/batch', { tags })
    return response.data
  }

  async updateOutboundMembers(tag: string, members: string[]): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string; tag: string }>>(`/outbounds/${tag}/members`, { members })
    return response.data
  }
}
