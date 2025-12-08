import axios, { type AxiosInstance } from 'axios'

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
  }

  get<T>(url: string): Promise<{ data: T }> {
    return this.client.get<T>(url)
  }

  post<T>(url: string, data?: unknown, config?: { headers?: Record<string, string>; timeout?: number }): Promise<{ data: T }> {
    return this.client.post<T>(url, data, config)
  }

  put<T>(url: string, data?: unknown): Promise<{ data: T }> {
    return this.client.put<T>(url, data)
  }

  delete<T>(url: string): Promise<{ data: T }> {
    return this.client.delete<T>(url)
  }
}

export const apiService = new ApiService()
