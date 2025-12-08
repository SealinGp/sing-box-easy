import type { ApiService } from './api'
import type { BasicResponse, ClashAPI, CacheFile } from '../types/api'

export class ExperimentalService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getClashAPI(): Promise<BasicResponse<ClashAPI>> {
    const response = await this.api.get<BasicResponse<ClashAPI>>('/experimental/clash-api')
    return response.data
  }

  async updateClashAPI(clashAPI: ClashAPI): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/experimental/clash-api', clashAPI)
    return response.data
  }

  async getCacheFile(): Promise<BasicResponse<CacheFile>> {
    const response = await this.api.get<BasicResponse<CacheFile>>('/experimental/cache-file')
    return response.data
  }

  async updateCacheFile(cacheFile: CacheFile): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/experimental/cache-file', cacheFile)
    return response.data
  }
}
