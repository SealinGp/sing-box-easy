import type { ApiService } from './api'
import type { BasicResponse, ConfigVersion, SingBoxConfig } from '../types/api'

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

  // --- Config version history ---

  async listVersions(): Promise<BasicResponse<{ versions: ConfigVersion[] }>> {
    const response = await this.api.get<BasicResponse<{ versions: ConfigVersion[] }>>('/config/versions')
    return response.data
  }

  async getVersion(id: number): Promise<BasicResponse<SingBoxConfig>> {
    const response = await this.api.get<BasicResponse<SingBoxConfig>>(`/config/versions/${id}`)
    return response.data
  }

  async rollbackToVersion(id: number): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string }>>(`/config/versions/${id}/rollback`)
    return response.data
  }

  async deleteVersion(id: number): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.delete<BasicResponse<{ message: string }>>(`/config/versions/${id}`)
    return response.data
  }
}
