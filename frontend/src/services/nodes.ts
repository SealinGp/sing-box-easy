import type { ApiService } from './api'
import type { BasicResponse, Outbound } from '../types/api'

export class NodesService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async parseNodes(subscription: string): Promise<BasicResponse<{ message: string; node_count: number; nodes: Outbound[] }>> {
    const response = await this.api.post<BasicResponse<{ message: string; node_count: number; nodes: Outbound[] }>>('/nodes/parse', {
      subscription: subscription,
    })
    return response.data
  }
}
