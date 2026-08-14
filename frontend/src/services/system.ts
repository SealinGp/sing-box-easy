import { ApiService } from './api'
import type { SystemInfo, BasicResponse } from '../types/api'

/** Host and version details shown in the Settings "About" card. */
export class SystemService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getInfo(): Promise<SystemInfo> {
    const { data } = await this.api.get<BasicResponse<SystemInfo>>('/system/info')
    return data.data
  }
}
