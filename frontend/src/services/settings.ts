import type { ApiService } from './api'
import type { AppSettings, BasicResponse, UpdateSettingsResult } from '../types/api'
import type { NodeQualityRules } from '../utils/nodeQuality'

type QualitySettings = { node_quality: NodeQualityRules }

export class SettingsService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getSettings(): Promise<BasicResponse<AppSettings & QualitySettings>> {
    const response = await this.api.get<BasicResponse<AppSettings & QualitySettings>>('/settings')
    return response.data
  }

  async updateSettings(payload: {
    node_quality?: NodeQualityRules
    config_versions_keep?: number
    /** Empty string clears the override and falls back to app.yml. */
    github_oauth_client_id?: string
  }): Promise<BasicResponse<UpdateSettingsResult & QualitySettings>> {
    const response = await this.api.put<BasicResponse<UpdateSettingsResult & QualitySettings>>('/settings', payload)
    return response.data
  }
}
