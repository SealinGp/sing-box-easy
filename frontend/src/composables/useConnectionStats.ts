import { onBeforeUnmount, ref, shallowRef, watch, type Ref } from 'vue'
import { openConnectionsStream } from '../services/runtime'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import { pushSample, type TrafficSample } from '../utils/trafficChart'
import type { ConnectionTotals } from '../types/runtime'

const RECONNECT_DELAY_MS = 3_000

/**
 * Totals and a short rolling history for the Overview's connections card.
 *
 * Reads the connections stream in its totals-only mode (`?rows=0`): the card
 * charts three numbers a second and would otherwise be sent every connection
 * on the router to throw away.
 *
 * The history is the life of the card. sing-box keeps no speed history to ask
 * for, so the chart starts empty and fills from the right — the same as the
 * dashboard this replaces.
 */
export function useConnectionStats(enabled: Ref<boolean>, fallbackError: () => string) {
  const totals = shallowRef<ConnectionTotals | null>(null)
  const history = shallowRef<TrafficSample[]>([])
  const error = ref('')

  let controller: AbortController | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  const close = () => {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    controller?.abort()
    controller = null
  }

  const open = () => {
    close()
    if (!enabled.value) return
    const current = new AbortController()
    controller = current

    void openConnectionsStream(
      {
        onFrame: (frame) => {
          error.value = ''
          totals.value = frame.totals
          history.value = pushSample(history.value, {
            at: frame.at,
            down: frame.totals.down_rate,
            up: frame.totals.up_rate,
            connections: frame.totals.connections,
          })
        },
        onError: (err) => {
          error.value = apiErrorMessage(err, fallbackError())
        },
        onClose: () => {
          if (controller !== current) return
          controller = null
          if (enabled.value) reconnectTimer = setTimeout(open, RECONNECT_DELAY_MS)
        },
      },
      current.signal,
      { summary: true },
    )
  }

  watch(enabled, (on) => {
    if (on) {
      open()
      return
    }
    close()
    // A stopped sing-box has no speed. Keeping the last minute on screen would
    // draw traffic through a process that is not running.
    totals.value = null
    history.value = []
    error.value = ''
  }, { immediate: true })

  onBeforeUnmount(close)

  return { totals, history, error }
}
