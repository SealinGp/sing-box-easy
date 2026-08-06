import type { ApiService } from './api'
import type { BasicResponse } from '../types/api'
import type { GitHubAuthStatus, GitHubLoginSession } from '../types/githubauth'

/**
 * GitHub sign-in via OAuth device flow.
 *
 * The credential never travels through the browser: `startLogin` returns only
 * a short user code, the user approves it on github.com, and the server
 * exchanges it for a token it stores itself.
 */
export class GitHubAuthService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  /** Whether sign-in is available and an account is connected. */
  async getStatus(): Promise<GitHubAuthStatus> {
    const response = await this.api.get<BasicResponse<GitHubAuthStatus>>('/github/auth/status')
    return normalizeStatus(response.data?.data)
  }

  /**
   * Begin a login. Returns the code to show the user. If a login is already
   * in flight the server returns that same session rather than a new code.
   */
  async startLogin(): Promise<GitHubLoginSession> {
    const response = await this.api.post<BasicResponse<GitHubLoginSession>>('/github/auth/device')
    return normalizeSession(response.data?.data)
  }

  /** Poll a pending login until `status` leaves 'pending'. */
  async getSession(sessionId: string): Promise<GitHubLoginSession> {
    const response = await this.api.get<BasicResponse<GitHubLoginSession>>(
      `/github/auth/device/${sessionId}`,
    )
    return normalizeSession(response.data?.data)
  }

  /** Abort a pending login. */
  async cancelLogin(sessionId: string): Promise<GitHubAuthStatus> {
    const response = await this.api.delete<BasicResponse<GitHubAuthStatus>>(
      `/github/auth/device/${sessionId}`,
    )
    return normalizeStatus(response.data?.data)
  }

  /** Disconnect, returning update checks to anonymous rate-limited access. */
  async signOut(): Promise<GitHubAuthStatus> {
    const response = await this.api.delete<BasicResponse<GitHubAuthStatus>>('/github/auth')
    return normalizeStatus(response.data?.data)
  }
}

/**
 * Coerce an untrusted payload into a complete GitHubAuthStatus.
 *
 * A server that predates these endpoints answers 404 with no envelope, which
 * would otherwise hand the card an `undefined` status and crash its render.
 * Treating an unusable payload as "not configured" degrades to the explanatory
 * state instead.
 */
function normalizeStatus(raw: unknown): GitHubAuthStatus {
  const data = (raw ?? {}) as Partial<GitHubAuthStatus>
  return {
    configured: data.configured === true,
    connected: data.connected === true,
    login: typeof data.login === 'string' ? data.login : '',
    pending_session: typeof data.pending_session === 'string' ? data.pending_session : '',
  }
}

/** Coerce an untrusted payload into a complete GitHubLoginSession. */
function normalizeSession(raw: unknown): GitHubLoginSession {
  const data = (raw ?? {}) as Partial<GitHubLoginSession>
  return {
    id: typeof data.id === 'string' ? data.id : '',
    user_code: typeof data.user_code === 'string' ? data.user_code : '',
    verification_uri:
      typeof data.verification_uri === 'string' && data.verification_uri
        ? data.verification_uri
        : 'https://github.com/login/device',
    // An unrecognized status must not read as 'pending', or the UI would poll
    // a session that will never resolve.
    status: isLoginStatus(data.status) ? data.status : 'failed',
    login: typeof data.login === 'string' ? data.login : '',
    error: typeof data.error === 'string' ? data.error : '',
    expires_at: typeof data.expires_at === 'string' ? data.expires_at : '',
    expires_in: typeof data.expires_in === 'number' ? data.expires_in : 0,
  }
}

const LOGIN_STATUSES = ['pending', 'authorized', 'denied', 'expired', 'failed'] as const

function isLoginStatus(value: unknown): value is GitHubLoginSession['status'] {
  return typeof value === 'string' && (LOGIN_STATUSES as readonly string[]).includes(value)
}
