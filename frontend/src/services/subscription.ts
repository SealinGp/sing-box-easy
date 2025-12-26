import type { ApiService } from './api'
import type { BasicResponse, Subscription, Outbound } from '../types/api'

export class SubscriptionService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getSubscriptions(): Promise<BasicResponse<{ subscriptions: Subscription[] }>> {
    const response = await this.api.get<BasicResponse<{ subscriptions: Subscription[] }>>('/subscriptions')
    return response.data
  }

  async addSubscription(subscription: Partial<Subscription>): Promise<BasicResponse<{ message: string; id: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string; id: string }>>('/subscriptions', subscription)
    return response.data
  }

  async getSubscriptionByID(id: string): Promise<BasicResponse<Subscription>> {
    const response = await this.api.get<BasicResponse<Subscription>>(`/subscriptions/${id}`)
    return response.data
  }

  async updateSubscription(id: string, subscription: Partial<Subscription>): Promise<BasicResponse<{ message: string; id: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string; id: string }>>(`/subscriptions/${id}`, subscription)
    return response.data
  }

  async deleteSubscription(id: string): Promise<BasicResponse<{ message: string; id: string }>> {
    const response = await this.api.delete<BasicResponse<{ message: string; id: string }>>(`/subscriptions/${id}`)
    return response.data
  }

  async updateSubscriptionContent(id: string): Promise<BasicResponse<{ message: string; id: string; node_count: number; nodes: Outbound[] }>> {
    const response = await this.api.post<BasicResponse<{ message: string; id: string; node_count: number; nodes: Outbound[] }>>(`/subscriptions/${id}/update`)
    return response.data
  }
}
