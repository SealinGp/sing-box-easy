import { ref, computed, onUnmounted } from 'vue'
import { githubAuthService } from '../services'
import type { GitHubAuthStatus, GitHubLoginSession } from '../types/githubauth'

/**
 * Drives GitHub sign-in via OAuth device flow.
 *
 * Flow: `signIn()` asks the server for a code, the user approves it on
 * github.com, and this composable polls until the server reports the outcome.
 * The token itself never reaches the browser.
 */

/** How often to ask the server whether the user has approved yet. */
const POLL_INTERVAL_MS = 2000

/**
 * Hard stop on polling. The server-side code expires after ~15 minutes; this
 * guarantees the interval is cleared even if a status response never resolves.
 */
const MAX_POLL_MS = 16 * 60 * 1000

export function useGitHubAuth() {
  const status = ref<GitHubAuthStatus>({
    configured: false,
    connected: false,
    login: '',
    pending_session: '',
  })
  const session = ref<GitHubLoginSession | null>(null)

  const loading = ref(false)
  const starting = ref(false)
  const signingOut = ref(false)
  const error = ref('')

  let pollTimer: ReturnType<typeof setInterval> | null = null
  let pollStartedAt = 0

  const isPending = computed(() => session.value?.status === 'pending')

  /** Countdown text for the code, e.g. "14:32". */
  const expiresLabel = computed(() => {
    const secs = session.value?.expires_in ?? 0
    if (secs <= 0) return ''
    const mins = Math.floor(secs / 60)
    return `${mins}:${String(secs % 60).padStart(2, '0')}`
  })

  function stopPolling() {
    if (pollTimer !== null) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  /** Load the connection state; also rejoins a login already in flight. */
  async function refresh() {
    loading.value = true
    error.value = ''
    try {
      status.value = await githubAuthService.getStatus()
      // A page reload during sign-in must not orphan the pending login.
      if (status.value.pending_session && !isPending.value) {
        await resumeSession(status.value.pending_session)
      }
    } catch (err) {
      error.value = messageOf(err)
    } finally {
      loading.value = false
    }
  }

  async function resumeSession(sessionId: string) {
    try {
      session.value = await githubAuthService.getSession(sessionId)
      if (session.value.status === 'pending') startPolling()
    } catch {
      // The session was evicted server-side; nothing to rejoin.
      session.value = null
    }
  }

  /** Request a device code and begin waiting for approval. */
  async function signIn() {
    starting.value = true
    error.value = ''
    try {
      session.value = await githubAuthService.startLogin()
      if (session.value.status === 'pending') startPolling()
    } catch (err) {
      error.value = messageOf(err)
    } finally {
      starting.value = false
    }
  }

  function startPolling() {
    stopPolling()
    pollStartedAt = Date.now()

    pollTimer = setInterval(async () => {
      const current = session.value
      if (!current) {
        stopPolling()
        return
      }

      if (Date.now() - pollStartedAt > MAX_POLL_MS) {
        stopPolling()
        error.value = 'Timed out waiting for GitHub authorization.'
        return
      }

      try {
        const next = await githubAuthService.getSession(current.id)
        session.value = next

        if (next.status !== 'pending') {
          stopPolling()
          if (next.status === 'authorized') {
            status.value = await githubAuthService.getStatus()
          } else if (next.error) {
            error.value = next.error
          }
        }
      } catch (err) {
        // A transient failure must not kill the loop — the user may still be
        // approving. Only a hard stop clears the interval.
        error.value = messageOf(err)
      }
    }, POLL_INTERVAL_MS)
  }

  /** Abort a pending login. */
  async function cancel() {
    const current = session.value
    stopPolling()
    session.value = null
    if (!current) return

    try {
      status.value = await githubAuthService.cancelLogin(current.id)
    } catch {
      // Already gone server-side; the local state is already cleared.
    }
  }

  /** Disconnect the account. */
  async function signOut() {
    signingOut.value = true
    error.value = ''
    try {
      status.value = await githubAuthService.signOut()
      session.value = null
    } catch (err) {
      error.value = messageOf(err)
    } finally {
      signingOut.value = false
    }
  }

  /** Dismiss a finished (failed/denied/expired) session panel. */
  function dismiss() {
    stopPolling()
    session.value = null
    error.value = ''
  }

  // A component unmounting mid-login must not leave an interval running.
  onUnmounted(stopPolling)

  return {
    status,
    session,
    loading,
    starting,
    signingOut,
    error,
    isPending,
    expiresLabel,
    refresh,
    signIn,
    cancel,
    signOut,
    dismiss,
  }
}

function messageOf(err: unknown): string {
  if (err instanceof Error) return err.message
  return String(err)
}
