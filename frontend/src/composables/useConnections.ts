import { onBeforeUnmount, ref, shallowRef, watch, type Ref } from 'vue'
import { openConnectionsStream } from '../services/runtime'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { trackFrame, type TrackedConnection } from '../utils/connectionsTable'
import type { ConnectionRow, ConnectionTotals } from '../types/runtime'

/**
 * Same delay as the Overview's live view, for the same reason: the commonest
 * drop is the restart the panel itself triggers on a config save.
 */
const RECONNECT_DELAY_MS = 3_000

/**
 * The live connection table, plus the closed-connection history derived from
 * it (see utils/connectionsTable.ts for why that is derived client-side).
 *
 * `enabled` is whether sing-box is running — the stream is answered by the
 * running process, so there is nothing to open while it is down.
 */
export function useConnections(enabled: Ref<boolean>, fallbackError: () => string) {
  const active = shallowRef<TrackedConnection[]>([])
  const closed = shallowRef<TrackedConnection[]>([])
  const totals = shallowRef<ConnectionTotals | null>(null)
  /** The last frame's sample time (unix ms): the clock durations are measured against. */
  const now = ref(Date.now())
  const error = ref('')
  const connecting = ref(false)
  /** Frozen display. The stream stays open so resuming is instant. */
  const paused = ref(false)

  let controller: AbortController | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  // The raw rows of the last APPLIED frame; the baseline for the next diff.
  let previous: ConnectionRow[] = []

  const clearReconnect = () => {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  const close = () => {
    clearReconnect()
    controller?.abort()
    controller = null
    connecting.value = false
  }

  const apply = (rows: ConnectionRow[], at: number, frameTotals: ConnectionTotals) => {
    const next = trackFrame(previous, closed.value, rows, at)
    previous = rows
    active.value = next.active
    closed.value = next.closed
    totals.value = frameTotals
    now.value = at
  }

  const open = () => {
    close()
    if (!enabled.value) return

    const current = new AbortController()
    controller = current
    connecting.value = true

    void openConnectionsStream(
      {
        onFrame: (frame) => {
          connecting.value = false
          error.value = ''
          // While paused nothing moves. On resume the next frame is diffed
          // against the frame that was frozen, so everything that ended in
          // between is filed as closed at once rather than lost.
          if (paused.value) return
          apply(frame.connections, frame.at, frame.totals)
        },
        onError: (err) => {
          error.value = apiErrorMessage(err, fallbackError())
        },
        onClose: () => {
          // A superseded stream closing must not schedule a reconnect.
          if (controller !== current) return
          connecting.value = false
          controller = null
          if (enabled.value) reconnectTimer = setTimeout(open, RECONNECT_DELAY_MS)
        },
      },
      current.signal,
    )
  }

  watch(enabled, (on) => {
    if (on) {
      open()
      return
    }
    close()
    // sing-box stopped: every connection it held is gone. They are filed as
    // closed rather than dropped, since "what was open when it went down" is
    // exactly what someone opens this page to find out.
    if (previous.length > 0) apply([], Date.now(), { ...(totals.value ?? emptyTotals()), connections: 0, down_rate: 0, up_rate: 0 })
    error.value = ''
  }, { immediate: true })

  onBeforeUnmount(close)

  const clearClosed = () => {
    closed.value = []
  }

  return { active, closed, totals, now, error, connecting, paused, clearClosed }
}

function emptyTotals(): ConnectionTotals {
  return { connections: 0, down_rate: 0, up_rate: 0, download_total: 0, upload_total: 0, memory: 0 }
}
