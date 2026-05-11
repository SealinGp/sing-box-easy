import axios, { type AxiosInstance, AxiosError } from 'axios'
import { ApiError, Code, type BasicResponse } from '../types/api'

/**
 * The sing-box-easy backend always returns HTTP 200 with a BasicResponse
 * envelope `{ code, data, msg }`. A non-`Code.Success` value means the
 * request failed at the business level. Axios doesn't treat that as an
 * error by default, so we install a response interceptor that throws an
 * `ApiError` whenever the envelope reports failure. Every `catch (err)`
 * downstream can then rely on `err.message` carrying the server's `msg`.
 *
 * Network / transport failures (no response, 5xx, parse errors) still
 * surface as standard `AxiosError` with `err.message` populated.
 */
function isBasicResponse(v: unknown): v is BasicResponse {
  return (
    v !== null &&
    typeof v === 'object' &&
    'code' in (v as Record<string, unknown>) &&
    'msg' in (v as Record<string, unknown>)
  )
}

export class ApiService {
  private client: AxiosInstance

  constructor() {
    this.client = axios.create({
      baseURL: '/api/1.12.12',
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    // Throw ApiError on business-level failures so call sites only need
    // a single try/catch around the call — no need to also check code.
    this.client.interceptors.response.use(
      (response) => {
        const body = response.data
        if (isBasicResponse(body) && body.code !== Code.Success) {
          throw new ApiError(body.code, body.msg || 'Request failed')
        }
        return response
      },
      (error: AxiosError) => {
        // Map network/transport errors to a stable shape too. If the
        // server somehow returned a BasicResponse with a non-200 HTTP
        // status, propagate its msg as the error message.
        const body = error.response?.data
        if (isBasicResponse(body)) {
          return Promise.reject(new ApiError(body.code, body.msg || error.message))
        }
        return Promise.reject(error)
      },
    )
  }

  get<T>(url: string): Promise<{ data: T }> {
    return this.client.get<T>(url)
  }

  post<T>(
    url: string,
    data?: unknown,
    config?: { headers?: Record<string, string>; timeout?: number },
  ): Promise<{ data: T }> {
    return this.client.post<T>(url, data, config)
  }

  put<T>(url: string, data?: unknown): Promise<{ data: T }> {
    return this.client.put<T>(url, data)
  }

  delete<T>(url: string, data?: unknown): Promise<{ data: T }> {
    return this.client.delete<T>(url, data ? { data } : undefined)
  }
}

export const apiService = new ApiService()
