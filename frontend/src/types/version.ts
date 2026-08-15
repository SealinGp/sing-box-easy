// App version / self-update types. Mirrors app/pkg/appupdate + version_handler.go.

/** Lifecycle of a self-update task. */
export type UpdateTaskStatus = 'running' | 'completed' | 'failed' | 'restarting'

/**
 * A downloaded, checksum-verified OpenWrt package plus the commands that
 * install it. Present only on a finished `prepare-package` task: opkg installs
 * are never performed by the panel itself.
 */
export interface IpkPlan {
  version: string
  /** opkg architecture, e.g. "x86_64" — not Go's "amd64". */
  architecture: string
  path: string
  sha256: string
  /** False when the release published no .sha256 sidecar to check against. */
  verified: boolean
  size_bytes: number
  /** Installs the downloaded file. Always usable. */
  command: string
  /** Upgrade from a configured feed. Empty unless a feed provides the package. */
  feed_command: string
}

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
  /** Set only by a prepare-package task that finished successfully. */
  plan?: IpkPlan
}

/** How this install can be upgraded. */
export type SelfUpdateMethod = 'tarball' | 'opkg'

/**
 * Which upgrade path applies to this install. opkg-managed installs cannot
 * self-update — opkg's prerm stops this very service — so the UI offers a
 * prepare-and-copy flow instead of an Update button that always fails.
 */
export interface SelfUpdateInfo {
  method: SelfUpdateMethod
  /** True when the panel can complete the update on its own. */
  automatic: boolean
  /** opkg architecture of the installed package. Empty unless method is opkg. */
  architecture: string
  /**
   * Whether a configured opkg feed offers this package. Meaningless when
   * `feed_known` is false — the feed cache is tmpfs and starts empty every
   * boot until `opkg update` runs.
   */
  feed_provides: boolean
  feed_known: boolean
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
  self_update: SelfUpdateInfo
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
