import type { ApiService } from './api'
import type { AppSettings, BasicResponse } from '../types/api'

export class SettingsService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getSettings(): Promise<BasicResponse<AppSettings>> {
    const response = await this.api.get<BasicResponse<AppSettings>>('/settings')
    return response.data
  }

  async updateSettings(
    payload: { config_versions_keep?: number }
  ): Promise<BasicResponse<{ config_versions_keep: number; limits: { min: number; max: number } }>> {
    const response = await this.api.put<
      BasicResponse<{ config_versions_keep: number; limits: { min: number; max: number } }>
    >('/settings', payload)
    return response.data
  }
}
