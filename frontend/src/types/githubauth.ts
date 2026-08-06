// GitHub sign-in (OAuth device flow) types.
// Mirrors app/pkg/githubauth + githubauth_handler.go.

/** Lifecycle of a device-flow login. */
export type GitHubLoginStatus = 'pending' | 'authorized' | 'denied' | 'expired' | 'failed'

/**
 * A pending or finished device-flow login.
 *
 * The device code is deliberately absent: it is the bearer credential for the
 * pending authorization and never leaves the server.
 */
export interface GitHubLoginSession {
  id: string
  /** The short code the user types on GitHub, e.g. "WDJB-MJHT". */
  user_code: string
  /** Where the user enters `user_code` — https://github.com/login/device. */
  verification_uri: string
  status: GitHubLoginStatus
  /** The signed-in account name; set once status is 'authorized'. */
  login: string
  /** Human-readable failure reason when status is denied/expired/failed. */
  error: string
  expires_at: string
  /** Seconds until the code expires; 0 once past. */
  expires_in: number
}

/** Current GitHub connection state. */
export interface GitHubAuthStatus {
  /**
   * False when the deployment has no OAuth client ID configured, so sign-in
   * cannot be offered at all.
   */
  configured: boolean
  connected: boolean
  /** Signed-in account name; empty when connected via GITHUB_TOKEN. */
  login: string
  /** ID of a login already in flight, so a reloaded page can rejoin it. */
  pending_session: string
}
