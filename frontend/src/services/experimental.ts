import type { ApiService } from './api'
import type { BasicResponse, ClashAPI, CacheFile, V2RayAPI } from '../types/api'

/**
 * Cache-file fields that must be omitted rather than sent blank.
 *
 * sing-box types `rdrc_timeout` as a duration and its parser rejects an empty
 * string outright (`time: invalid duration ""`), failing the whole request.
 * Form inputs submit "" for "left untouched", so the empty value has to be
 * dropped. Both keys are `omitempty` server-side, which makes an absent key
 * exactly equivalent to an unset one.
 */
const CACHE_FILE_OMIT_WHEN_BLANK = ['rdrc_timeout', 'cache_id'] as const

/** Returns a copy with blank optional fields removed. Never mutates the input. */
const withoutBlankOptionals = (cacheFile: CacheFile): CacheFile => {
  const cleaned: CacheFile = { ...cacheFile }
  for (const key of CACHE_FILE_OMIT_WHEN_BLANK) {
    if (cleaned[key] === '') delete cleaned[key]
  }
  return cleaned
}

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
    const response = await this.api.put<BasicResponse<{ message: string }>>(
      '/experimental/cache-file',
      withoutBlankOptionals(cacheFile),
    )
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
