import type { ApiService } from './api'
import type { BasicResponse, DashboardTask } from '../types/api'

export class DashboardService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async downloadDashboard(
    targetDir?: string,
    downloadURL?: string,
    proxy?: string
  ): Promise<BasicResponse<{ task_id: string; message: string }>> {
    const response = await this.api.post<BasicResponse<{ task_id: string; message: string }>>('/dashboard/download', {
      target_dir: targetDir,
      download_url: downloadURL,
      proxy: proxy,
    })
    return response.data
  }

  async getDashboardTask(taskId: string): Promise<BasicResponse<DashboardTask>> {
    const response = await this.api.get<BasicResponse<DashboardTask>>(`/dashboard/task/${taskId}`)
    return response.data
  }

  async getDashboardStatus(): Promise<BasicResponse<{ installed: boolean; path: string }>> {
    const response = await this.api.get<BasicResponse<{ installed: boolean; path: string }>>('/dashboard/status')
    return response.data
  }

  async uploadDashboard(
    file: File,
    targetDir?: string,
    folderName?: string
  ): Promise<BasicResponse<{ task_id: string; message: string }>> {
    const formData = new FormData()
    formData.append('file', file)
    if (targetDir) {
      formData.append('target_dir', targetDir)
    }
    if (folderName) {
      formData.append('folder_name', folderName)
    }

    const response = await this.api.post<BasicResponse<{ task_id: string; message: string }>>(
      '/dashboard/upload',
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
        timeout: 60000,
      }
    )
    return response.data
  }
}
