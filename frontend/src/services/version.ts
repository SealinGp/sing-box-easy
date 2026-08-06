import type { ApiService } from './api'
import type { BasicResponse } from '../types/api'
import type { ReleaseList, UpdateTask, VersionStatus } from '../types/version'

/**
 * App version + self-update API.
 *
 * `startUpdate` replaces the binary and frontend bundle on disk and then
 * restarts the process, so the server becomes briefly unreachable. Callers are
 * expected to poll `getTask` until it reports `completed`/`restarting` and then
 * wait for the server to come back before reloading.
 */
export class VersionService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  /** Current version vs. the newest published release. */
  async getStatus(refresh = false): Promise<VersionStatus> {
    const response = await this.api.get<BasicResponse<VersionStatus>>(
      `/version${refresh ? '?refresh=true' : ''}`,
    )
    return response.data.data
  }

  /** Published releases, newest first, for the version picker. */
  async listReleases(refresh = false): Promise<ReleaseList> {
    const response = await this.api.get<BasicResponse<ReleaseList>>(
      `/version/releases${refresh ? '?refresh=true' : ''}`,
    )
    return response.data.data
  }

  /** Start an update. Omit `version` to install the latest release. */
  async startUpdate(version?: string): Promise<UpdateTask> {
    const response = await this.api.post<BasicResponse<UpdateTask>>('/version/update', {
      version: version ?? '',
    })
    return response.data.data
  }

  /** Poll a running update. */
  async getTask(taskId: string): Promise<UpdateTask> {
    const response = await this.api.get<BasicResponse<UpdateTask>>(`/version/task/${taskId}`)
    return response.data.data
  }
}
