import type { ApiService } from './api'
import type { BasicResponse, SingBoxConfig } from '../types/api'

export class ConfigService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getConfig(): Promise<BasicResponse<SingBoxConfig>> {
    const response = await this.api.get<BasicResponse<SingBoxConfig>>('/config')
    return response.data
  }

  async validateConfig(config: SingBoxConfig): Promise<BasicResponse<{ valid: boolean; error?: string }>> {
    const response = await this.api.post<BasicResponse<{ valid: boolean; error?: string }>>(
      '/config/validate',
      config
    )
    return response.data
  }

  async getBackupConfig(): Promise<BasicResponse<SingBoxConfig>> {
    const response = await this.api.get<BasicResponse<SingBoxConfig>>('/config/backup')
    return response.data
  }

  async rollbackConfig(): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string }>>('/config/rollback')
    return response.data
  }
}
