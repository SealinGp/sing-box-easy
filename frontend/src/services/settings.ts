import type { ApiService } from './api'
import type { AppSettings, BasicResponse, UpdateSettingsResult } from '../types/api'

export class SettingsService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getSettings(): Promise<BasicResponse<AppSettings>> {
    const response = await this.api.get<BasicResponse<AppSettings>>('/settings')
    return response.data
  }

  async updateSettings(payload: {
    config_versions_keep?: number
    /** Empty string clears the override and falls back to app.yml. */
    github_oauth_client_id?: string
  }): Promise<BasicResponse<UpdateSettingsResult>> {
    const response = await this.api.put<BasicResponse<UpdateSettingsResult>>('/settings', payload)
    return response.data
  }
}
