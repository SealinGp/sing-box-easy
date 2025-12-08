import type { ApiService } from './api'
import type { BasicResponse, LogConfig } from '../types/api'

export class LogService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getLog(): Promise<BasicResponse<LogConfig>> {
    const response = await this.api.get<BasicResponse<LogConfig>>('/log')
    return response.data
  }

  async updateLog(log: LogConfig): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/log', log)
    return response.data
  }
}
