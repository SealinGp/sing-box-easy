import { computed, readonly, ref } from 'vue'
import { versionService } from '../services'
import type { IpkPlan, ReleaseInfo, SelfUpdateInfo, UpdateTask, VersionStatus } from '../types/version'

/**
 * App version / self-update state.
 *
 * State lives at module scope so the sidebar badge and the Settings page share
 * one source of truth (and one poll loop) instead of each running their own.
 */

/** Where the update flow currently is, from the user's point of view. */
export type UpdatePhase =
  | 'idle'
  | 'updating' // downloading + installing
  | 'preparing' // downloading + verifying an .ipk for a manual install
  | 'prepared' // .ipk on disk, waiting for the operator to run the command
  | 'restarting' // files swapped, process re-executing
  | 'waiting' // polling until the server answers again
  | 'done' // about to reload the page
  | 'failed'

/** How often to poll the update task while it runs. */
const TASK_POLL_INTERVAL_MS = 1000

/** How often to probe the server while it restarts. */
const SERVER_PROBE_INTERVAL_MS = 2000

/** Give up waiting for the restarted server after this long. */
const SERVER_PROBE_TIMEOUT_MS = 120_000

/** Grace period before reloading, so the user can read the success state. */
const RELOAD_DELAY_MS = 1200

const status = ref<VersionStatus | null>(null)
const releases = ref<ReleaseInfo[]>([])
const task = ref<UpdateTask | null>(null)
const phase = ref<UpdatePhase>('idle')
const errorMessage = ref('')
const checking = ref(false)
const loadingReleases = ref(false)

let pollTimer: ReturnType<typeof setTimeout> | null = null

// Shared with the rest of the module-level state above: every caller of
// useAppUpdate() must see the same in-flight status request, otherwise the
// de-duplication in refreshStatus() only ever covers a single component.
let statusInFlight: Promise<VersionStatus | null> | null = null

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms))

const clearPoll = () => {
  if (pollTimer !== null) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

export function useAppUpdate() {
  /**
   * True while work is in flight and the UI must stay locked. `prepared` is
   * excluded: the download finished and the operator now needs the controls
   * back to pick another version or re-prepare.
   */
  const busy = computed(
    () => phase.value !== 'idle' && phase.value !== 'failed' && phase.value !== 'prepared',
  )

  /**
   * How this install can be upgraded. Defaults to the automatic tarball path
   * so the UI behaves as before until the backend says otherwise.
   */
  const selfUpdate = computed<SelfUpdateInfo>(
    () => status.value?.self_update ?? { method: 'tarball', automatic: true, architecture: '', feed_provides: false, feed_known: false },
  )

  /** True when opkg owns this install and the panel must not install itself. */
  const isOpkgManaged = computed(() => selfUpdate.value.method === 'opkg')

  /** The prepared package, once a prepare task has finished. */
  const plan = computed<IpkPlan | null>(() => task.value?.plan ?? null)

  /** Progress percentage for the bar; 100 once we're waiting on the restart. */
  const progress = computed(() => {
    if (phase.value === 'restarting' || phase.value === 'waiting' || phase.value === 'done') {
      return 100
    }
    return task.value?.progress ?? 0
  })

  /**
   * True when a strictly newer release exists. A dev build (no version stamp)
   * cannot be compared, so it never shows the "out of date" badge even though
   * the backend permits updating it.
   */
  const hasUpdate = computed(
    () => Boolean(status.value?.has_update) && Boolean(status.value?.current_known),
  )

  /**
   * True when the backend is willing to install something newer. Unlike
   * `hasUpdate` this also covers unstamped builds, which the backend treats as
   * "anything is newer" — those installs genuinely should be updated, so the
   * sidebar still prompts for them.
   */
  const updateOffered = computed(
    () => Boolean(status.value?.has_update) && Boolean(status.value?.latest_version),
  )

  const currentVersion = computed(() => status.value?.current_version ?? '')
  const latestVersion = computed(() => status.value?.latest_version ?? '')

  /**
   * Fetch the version status. Never throws — a GitHub outage or rate limit is
   * surfaced through `status.check_error` so the page still renders.
   *
   * Unforced calls share one in-flight request. Landing directly on Settings
   * mounts the navigation shell and the embedded update panel in the same
   * tick, and both ask for the status: without this they fire two identical
   * requests, and whichever returns first clears `checking`, stopping the
   * "Checking…" spinner while the other is still running. An explicit
   * `force` (the "Check again" button) always issues a fresh request.
   */
  const refreshStatus = (force = false): Promise<VersionStatus | null> => {
    if (!force && statusInFlight) return statusInFlight
    statusInFlight = doRefreshStatus(force).finally(() => {
      statusInFlight = null
    })
    return statusInFlight
  }

  const doRefreshStatus = async (force: boolean): Promise<VersionStatus | null> => {
    checking.value = true
    try {
      const next = await versionService.getStatus(force)
      status.value = next

      // Adopt an update started by another browser tab / session so this view
      // shows live progress instead of an idle button.
      if (next.updating && next.running_task_id && phase.value === 'idle') {
        void resumeTask(next.running_task_id)
      }
      return next
    } catch (err) {
      errorMessage.value = err instanceof Error ? err.message : String(err)
      return null
    } finally {
      checking.value = false
    }
  }

  /** Load the release list for the version picker. Throws on failure. */
  const loadReleases = async (force = false): Promise<ReleaseInfo[]> => {
    loadingReleases.value = true
    try {
      const list = await versionService.listReleases(force)
      releases.value = list.releases
      return list.releases
    } finally {
      loadingReleases.value = false
    }
  }

  /**
   * Start an update. Pass a tag to pin a specific release, or omit it for the
   * latest. Resolves as soon as the task is accepted; progress then arrives
   * through the polled `task` ref.
   */
  const startUpdate = async (version?: string): Promise<void> => {
    clearPoll()
    errorMessage.value = ''
    phase.value = 'updating'

    try {
      const started = await versionService.startUpdate(version)
      task.value = started
      pollTask(started.id)
    } catch (err) {
      phase.value = 'failed'
      errorMessage.value = err instanceof Error ? err.message : String(err)
      throw err
    }
  }

  /**
   * Download and verify the .ipk for an opkg-managed install. Resolves once
   * the task is accepted; the file lands on the router and the finished task
   * carries the command to run.
   */
  const preparePackage = async (version?: string): Promise<void> => {
    clearPoll()
    errorMessage.value = ''
    phase.value = 'preparing'

    try {
      const started = await versionService.preparePackage(version)
      task.value = started
      pollPrepare(started.id)
    } catch (err) {
      phase.value = 'failed'
      errorMessage.value = err instanceof Error ? err.message : String(err)
      throw err
    }
  }

  /**
   * Poll a prepare task. Deliberately separate from `pollTask`: nothing
   * restarts here, so a transport failure is a real error rather than the
   * expected restart signal, and completion must not reload the page.
   */
  const pollPrepare = (taskId: string) => {
    const tick = async () => {
      try {
        const next = await versionService.getTask(taskId)
        task.value = next

        if (next.status === 'failed') {
          phase.value = 'failed'
          errorMessage.value = next.error || next.message
          return
        }

        if (next.status === 'completed') {
          phase.value = next.plan ? 'prepared' : 'failed'
          if (!next.plan) {
            errorMessage.value = 'The package finished downloading but no install command was returned.'
          }
          return
        }

        pollTimer = setTimeout(tick, TASK_POLL_INTERVAL_MS)
      } catch (err) {
        phase.value = 'failed'
        errorMessage.value = err instanceof Error ? err.message : String(err)
      }
    }

    pollTimer = setTimeout(tick, TASK_POLL_INTERVAL_MS)
  }

  /** Attach to an already-running task (e.g. started in another tab). */
  const resumeTask = async (taskId: string): Promise<void> => {
    clearPoll()
    phase.value = 'updating'
    pollTask(taskId)
  }

  /**
   * Poll a task until it reaches a terminal state.
   *
   * A failed poll is NOT treated as an error: once the new binary is installed
   * the process re-executes, so the server legitimately stops answering
   * mid-poll. That transition is indistinguishable from a normal restart, so a
   * transport failure after install simply moves us to the waiting phase.
   */
  const pollTask = (taskId: string) => {
    const tick = async () => {
      try {
        const next = await versionService.getTask(taskId)
        task.value = next

        if (next.status === 'failed') {
          phase.value = 'failed'
          errorMessage.value = next.error || next.message
          return
        }

        if (next.status === 'completed' || next.status === 'restarting') {
          phase.value = 'restarting'
          await waitForServer()
          return
        }

        pollTimer = setTimeout(tick, TASK_POLL_INTERVAL_MS)
      } catch {
        // The server went away — that is the expected restart signal.
        phase.value = 'restarting'
        await waitForServer()
      }
    }

    pollTimer = setTimeout(tick, TASK_POLL_INTERVAL_MS)
  }

  /** Probe the API until the restarted server answers, then reload the page. */
  const waitForServer = async (): Promise<void> => {
    phase.value = 'waiting'
    const deadline = Date.now() + SERVER_PROBE_TIMEOUT_MS

    while (Date.now() < deadline) {
      await sleep(SERVER_PROBE_INTERVAL_MS)
      try {
        await versionService.getStatus(false)
        phase.value = 'done'
        // The bundle on disk was replaced too, so a full reload is required to
        // pick up the new frontend assets.
        await sleep(RELOAD_DELAY_MS)
        window.location.reload()
        return
      } catch {
        // Still restarting — keep probing.
      }
    }

    phase.value = 'failed'
    errorMessage.value =
      'The server did not come back within the expected time. Reload the page manually to check.'
  }

  /** Reset after a failure so the user can retry. */
  const reset = () => {
    clearPoll()
    phase.value = 'idle'
    task.value = null
    errorMessage.value = ''
  }

  return {
    status: readonly(status),
    releases: readonly(releases),
    task: readonly(task),
    phase: readonly(phase),
    errorMessage: readonly(errorMessage),
    checking: readonly(checking),
    loadingReleases: readonly(loadingReleases),
    busy,
    progress,
    hasUpdate,
    updateOffered,
    currentVersion,
    latestVersion,
    selfUpdate,
    isOpkgManaged,
    plan,
    refreshStatus,
    loadReleases,
    startUpdate,
    preparePackage,
    reset,
  }
}
