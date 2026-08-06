// App version / self-update types. Mirrors app/pkg/appupdate + version_handler.go.

/** Lifecycle of a self-update task. */
export type UpdateTaskStatus = 'running' | 'completed' | 'failed' | 'restarting'

/** Progress of a single self-update run. */
export interface UpdateTask {
  id: string
  status: UpdateTaskStatus
  message: string
  error: string
  /** 0-100. Download occupies the 5-60 band, install the rest. */
  progress: number
  from_version: string
  to_version: string
}

/** Running version compared against the newest published release. */
export interface VersionStatus {
  current_version: string
  /**
   * False for local/dev builds with no version stamp. The UI still allows an
   * update in that case, but cannot claim the install is out of date.
   */
  current_known: boolean
  latest_version: string
  latest_url: string
  latest_notes: string
  published_at: string
  has_update: boolean
  prerelease: boolean
  asset_name: string
  /** True while an update task is in flight. */
  updating: boolean
  /**
   * Non-empty when GitHub could not be reached (rate limit, offline). The
   * current version is still valid; only the comparison is unavailable.
   */
  check_error: string
  running_task_id: string | null
}

/** A single published release the user can pick as an update target. */
export interface ReleaseInfo {
  tag: string
  name: string
  prerelease: boolean
  published_at: string
  url: string
  notes: string
  is_current: boolean
  is_newer: boolean
}

export interface ReleaseList {
  current_version: string
  current_known: boolean
  releases: ReleaseInfo[]
}
