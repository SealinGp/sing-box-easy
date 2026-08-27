/**
 * The live log feed: server-pushed, with the old poll kept as a fallback.
 *
 * WHY THE FALLBACK IS NOT OPTIONAL
 * ────────────────────────────────
 * A stream can fail for reasons that have nothing to do with this application:
 * a reverse proxy that buffers `text/event-stream`, a corporate middlebox, an
 * old uhttpd in front of LuCI. The failure mode is silence — a log viewer that
 * shows nothing looks exactly like a sing-box that is logging nothing, and the
 * operator debugs the wrong thing. So a stream that will not start, or that
 * dies without the server saying why, drops back to the 1.5s poll that always
 * worked. Degrading to slower is fine; degrading to blank is not.
 *
 * THE HANDOVER
 * ────────────
 * The backlog comes from the unary endpoint, which returns a journald cursor;
 * the stream then resumes from it. That is the one case the poll could never
 * get right — only the systemd backend implements the cursor at all, so on
 * procd and file backends every poll re-read the same window and the client
 * diffed it. The stream has no such problem: the server sends each line once,
 * as it is written.
 */
import { onUnmounted, ref, watch, type Ref } from 'vue'
import { openStream } from '../services/stream'
import { serviceControlService } from '../services'
import type { LogSourceKind } from '../types/api'

/** How long the viewer keeps. Bounded so memory stays flat for a chatty proxy. */
export const MAX_LINES = 500
const INITIAL_LINES = 300
/** Fallback cadence — unchanged from the poll this replaces. */
const POLL_MS = 1500

export type LogSource = LogSourceKind | ''
export type LogTransport = 'stream' | 'poll' | 'idle'

/**
 * Which log is being watched.
 *
 * `singbox` is the proxy's own output, read from journald / logread / the
 * configured log.output file. `app` is this panel's log, held in an in-process
 * ring buffer. They differ in what they can promise — only the sing-box feed
 * survives a restart of the thing producing it — but not in their wire shape,
 * so everything below is written once.
 */
export type LogFeed = 'singbox' | 'app'

interface FeedEndpoints {
  tail: (lines: number, cursor: string) => Promise<{ lines: string[]; cursor: string; source: LogSource }>
  streamPath: (cursor: string) => string
}

const FEEDS: Record<LogFeed, FeedEndpoints> = {
  singbox: {
    tail: async (lines, cursor) => (await serviceControlService.getServiceLogs(lines, cursor)).data,
    streamPath: (cursor) => `/service/logs/stream?cursor=${encodeURIComponent(cursor)}`,
  },
  app: {
    // No cursor: a ring buffer has no resumable position, and the client
    // already reads an empty cursor as "re-read the window".
    tail: async (lines) => (await serviceControlService.getAppLogs(lines)).data,
    streamPath: () => '/system/logs/stream',
  },
}

export interface LogStreamOptions {
  /**
   * Which log to watch. A ref, so switching tabs tears the old feed down and
   * starts the new one — two feeds must never be appending into one buffer.
   */
  feed: Ref<LogFeed>
  /** Called with each batch of new lines, in arrival order. */
  onLines: (lines: string[]) => void
  /** Replaces the whole buffer — the initial window, and every poll refetch. */
  onReplace: (lines: string[]) => void
}

export function useLogStream(options: LogStreamOptions) {
  const source = ref<LogSource>('')
  const transport = ref<LogTransport>('idle')
  const errored = ref(false)
  const initialLoading = ref(true)

  let cursor = ''
  let controller: AbortController | null = null
  let pollTimer: ReturnType<typeof setInterval> | null = null
  /** Set while the caller has paused, so an in-flight retry does not restart. */
  let stopped = true

  // ── Fallback poll ─────────────────────────────────────────────────────────

  const pollOnce = async () => {
    // Captured before the await: a tab switch mid-flight must not let the old
    // feed's lines land in the new feed's buffer.
    const feed = options.feed.value
    try {
      const lineCount = cursor ? MAX_LINES : INITIAL_LINES
      const data = await FEEDS[feed].tail(lineCount, cursor)
      if (feed !== options.feed.value || stopped) return
      source.value = data.source
      if (cursor) {
        options.onLines(data.lines || [])
      } else {
        options.onReplace((data.lines || []).slice(-MAX_LINES))
      }
      if (data.cursor) cursor = data.cursor
      errored.value = false
    } catch {
      errored.value = true
    } finally {
      initialLoading.value = false
    }
  }

  const startPolling = () => {
    if (pollTimer !== null || stopped) return
    transport.value = 'poll'
    pollTimer = setInterval(() => {
      // A hidden tab has nothing to show and no reason to ask.
      if (typeof document !== 'undefined' && document.hidden) return
      void pollOnce()
    }, POLL_MS)
  }

  const stopPolling = () => {
    if (pollTimer !== null) {
      clearInterval(pollTimer)
      pollTimer = null
    }
  }

  // ── Stream ────────────────────────────────────────────────────────────────

  const startStream = () => {
    if (stopped) return
    controller = new AbortController()
    let sawServerEvent = false

    const feed = options.feed.value

    void openStream(FEEDS[feed].streamPath(cursor), {
      signal: controller.signal,
      onEvent: (name, data) => {
        // A late event from the feed we just switched away from.
        if (feed !== options.feed.value) return
        sawServerEvent = true
        const payload = (data ?? {}) as { lines?: string[]; cursor?: string; source?: LogSource }

        if (name === 'lines') {
          transport.value = 'stream'
          errored.value = false
          if (payload.cursor) cursor = payload.cursor
          if (payload.lines?.length) options.onLines(payload.lines)
          return
        }

        // The host has no readable log source. Polling will report the same
        // thing and the viewer already knows how to explain it.
        if (name === 'unsupported') {
          if (payload.source) source.value = payload.source
          stopStream()
          startPolling()
        }
        // `ended` — the follower stopped on its own. Fall back rather than
        // reconnect into the same dead end.
        if (name === 'ended') {
          stopStream()
          startPolling()
        }
      },
      onError: () => {
        // Never started, or died mid-flight. Either way the poll is what stands
        // between the operator and a blank screen.
        stopStream()
        startPolling()
      },
      onClose: () => {
        // A clean close with no event at all means the transport was accepted
        // but nothing came through — the signature of a buffering proxy.
        if (!sawServerEvent && !stopped && pollTimer === null) startPolling()
      },
    })
  }

  const stopStream = () => {
    controller?.abort()
    controller = null
  }

  // ── Lifecycle ─────────────────────────────────────────────────────────────

  const start = async () => {
    stopped = false
    // Seed the backlog first, so the viewer is never empty while the stream
    // negotiates, and so the stream has a cursor to resume from.
    await pollOnce()
    if (stopped) return
    if (source.value === 'none') {
      // Nothing to follow. Poll so the source note keeps refreshing if the
      // operator fixes their log config while the page is open.
      startPolling()
      return
    }
    startStream()
  }

  const stop = () => {
    stopped = true
    stopStream()
    stopPolling()
    transport.value = 'idle'
  }

  /**
   * Switching feeds is a full restart, not a filter.
   *
   * The cursor belongs to the feed that issued it, and the line buffer belongs
   * to the log being shown — carrying either across would splice two logs into
   * one view, which is the single most misleading thing this page could do.
   */
  watch(options.feed, () => {
    const wasRunning = !stopped
    stop()
    cursor = ''
    source.value = ''
    initialLoading.value = true
    if (wasRunning) void start()
  })

  onUnmounted(stop)

  return { source, transport, errored, initialLoading, start, stop, refetch: pollOnce }
}
