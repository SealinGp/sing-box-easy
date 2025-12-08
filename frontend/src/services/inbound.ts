import type { ApiService } from './api'
import type { BasicResponse, Inbound } from '../types/api'

export class InboundService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getInbounds(): Promise<BasicResponse<{ inbounds: Inbound[] }>> {
    const response = await this.api.get<BasicResponse<{ inbounds: Inbound[] }>>('/inbounds')
    return response.data
  }

  async addInbound(inbound: Inbound): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string; tag: string }>>('/inbounds', inbound)
    return response.data
  }

  async getInboundByTag(tag: string): Promise<BasicResponse<Inbound>> {
    const response = await this.api.get<BasicResponse<Inbound>>(`/inbounds/${tag}`)
    return response.data
  }

  async updateInbound(tag: string, inbound: Inbound): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string; tag: string }>>(`/inbounds/${tag}`, inbound)
    return response.data
  }

  async deleteInbound(tag: string): Promise<BasicResponse<{ message: string; tag: string }>> {
    const response = await this.api.delete<BasicResponse<{ message: string; tag: string }>>(`/inbounds/${tag}`)
    return response.data
  }
}
