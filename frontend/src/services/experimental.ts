import type { ApiService } from './api'
import type { BasicResponse, ClashAPI, CacheFile, V2RayAPI } from '../types/api'

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

  async getV2RayAPI(): Promise<BasicResponse<V2RayAPI>> {
    const response = await this.api.get<BasicResponse<V2RayAPI>>('/experimental/v2ray-api')
    return response.data
  }

  async updateV2RayAPI(v2rayAPI: V2RayAPI): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.put<BasicResponse<{ message: string }>>('/experimental/v2ray-api', v2rayAPI)
    return response.data
  }
}
