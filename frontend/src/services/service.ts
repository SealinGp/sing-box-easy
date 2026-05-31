import type { ApiService } from './api'
import type { BasicResponse, InitState, InstallTask, ServiceStatus, ServiceLogs } from '../types/api'

export class ServiceControlService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getServiceStatus(): Promise<BasicResponse<ServiceStatus>> {
    const response = await this.api.get<BasicResponse<ServiceStatus>>('/service/status')
    return response.data
  }

  // Fetch recent logs. Pass the cursor returned by the previous call to fetch
  // only newer lines (incremental polling for the live viewer).
  async getServiceLogs(lines = 300, cursor = ''): Promise<BasicResponse<ServiceLogs>> {
    const params = new URLSearchParams({ lines: String(lines) })
    if (cursor) params.set('cursor', cursor)
    const response = await this.api.get<BasicResponse<ServiceLogs>>(`/service/logs?${params.toString()}`)
    return response.data
  }

  async startService(): Promise<BasicResponse<any>> {
    const response = await this.api.post<BasicResponse<{ message: string }>>('/service/start')
    return response.data
  }

  async stopService(): Promise<BasicResponse<any>> {
    const response = await this.api.post<BasicResponse<{ message: string }>>('/service/stop')
    return response.data
  }

  async restartService(): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string }>>('/service/restart')
    return response.data
  }


  async getInitStatus(): Promise<BasicResponse<InitState>> {
    const response = await this.api.get<BasicResponse<InitState>>('/init/status')
    return response.data
  }

  async completeInit(): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string }>>('/init/complete')
    return response.data
  }

  async resetInit(): Promise<BasicResponse<{ message: string }>> {
    const response = await this.api.post<BasicResponse<{ message: string }>>('/init/reset')
    return response.data
  }



  async installSingBox(version?: string, beta?: boolean): Promise<BasicResponse<{ task_id: string; message: string }>> {
    const response = await this.api.post<BasicResponse<{ task_id: string; message: string }>>('/install', {
      version,
      beta,
    })
    return response.data
  }

  async getInstallTask(taskId: string): Promise<BasicResponse<InstallTask>> {
    const response = await this.api.get<BasicResponse<InstallTask>>(`/install/task/${taskId}`)
    return response.data
  }

  async getInstallStatus(): Promise<BasicResponse<{ installed: boolean; version: string; message: string }>> {
    const response = await this.api.get<BasicResponse<{ installed: boolean; version: string; message: string }>>('/install/status')
    return response.data
  }

  async updateSingBox(version?: string, beta?: boolean): Promise<BasicResponse<InstallTask>> {
    const response = await this.api.post<BasicResponse<InstallTask>>('/update', {
      version,
      beta,
    })
    return response.data
  }
}
