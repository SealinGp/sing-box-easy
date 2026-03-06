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

  async validateConfig(config: SingBoxConfig): Promise<BasicResponse<any>> {
    const response = await this.api.post<BasicResponse<any>>(
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

  async updateConfig(config: SingBoxConfig): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/config', config)
    return response.data
  }

  // Alias for updateConfig for backwards compatibility
  async saveConfig(config: SingBoxConfig): Promise<BasicResponse<{ message: string }>> {
    return this.updateConfig(config)
  }
}
