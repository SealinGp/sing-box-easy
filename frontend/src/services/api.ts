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

    // Request interceptor to dynamically inject the Authorization header from localStorage on every request
    this.client.interceptors.request.use(
      (config) => {
        const token = localStorage.getItem('sb_token')
        console.log(`[api.ts Request] URL: ${config.url}, Token exists: ${!!token}, Token value: ${token ? token.slice(0, 10) + '...' : 'none'}`)
        if (token) {
          config.headers = config.headers || {}
          config.headers['Authorization'] = `Bearer ${token}`
        }

        return config
      },
      (error) => {
        console.error('[api.ts Request Error]', error)
        return Promise.reject(error)
      }
    )

    // Throw ApiError on business-level failures so call sites only need
    // a single try/catch around the call — no need to also check code.
    this.client.interceptors.response.use(
      (response) => {
        const body = response.data
        console.log(`[api.ts Response Success] URL: ${response.config.url}, Code: ${body?.code}, Msg: ${body?.msg}`)

        if (isBasicResponse(body) && body.code !== Code.Success) {
          console.warn(`[api.ts Response Business Error] Code: ${body.code}, Msg: ${body.msg}`)

          if (body.code === Code.Unauthorized) {
            console.warn('[api.ts Response Business Error] Unauthorized. Clearing token.')

            this.clearToken()
            if (!window.location.pathname.endsWith('/login')) {

              console.warn(`[api.ts Response Business Error] Redirecting from ${window.location.pathname} to /login`)

              window.location.href = '/login'
            } else {
              console.warn('[api.ts Response Business Error] Already on login page, skipping redirect.')
            }
          }
          throw new ApiError(body.code, body.msg || 'Request failed')
        }
        return response
      },
      (error: AxiosError) => {
        const body = error.response?.data
        console.error(`[api.ts Response Network/HTTP Error] URL: ${error.config?.url}, Status: ${error.response?.status}`, error)
        if (isBasicResponse(body)) {
          console.warn(`[api.ts Response Network/HTTP Error BasicResponse] Code: ${body.code}, Msg: ${body.msg}`)
          if (body.code === Code.Unauthorized) {
            console.warn('[api.ts Response Network/HTTP Error BasicResponse] Unauthorized. Clearing token.')
            this.clearToken()
            if (!window.location.pathname.endsWith('/login')) {
              console.warn(`[api.ts Response Network/HTTP Error BasicResponse] Redirecting from ${window.location.pathname} to /login`)
              window.location.href = '/login'
            } else {
              console.warn('[api.ts Response Network/HTTP Error BasicResponse] Already on login page, skipping redirect.')
            }
          }
          return Promise.reject(new ApiError(body.code, body.msg || error.message))
        }
        if (error.response?.status === 401) {
          console.warn('[api.ts Response Network/HTTP Error 401] Unauthorized status. Clearing token.')
          this.clearToken()
          if (!window.location.pathname.endsWith('/login')) {
            console.warn(`[api.ts Response Network/HTTP Error 401] Redirecting from ${window.location.pathname} to /login`)
            window.location.href = '/login'
          } else {
            console.warn('[api.ts Response Network/HTTP Error 401] Already on login page, skipping redirect.')
          }
        }
        return Promise.reject(error)
      },
    )
  }

  setToken(token: string) {
    localStorage.setItem('sb_token', token)
  }

  clearToken() {
    localStorage.removeItem('sb_token')
  }

  getToken(): string | null {
    return localStorage.getItem('sb_token')
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

