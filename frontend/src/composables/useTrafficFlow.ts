/**
 * Owns one live traffic stream for the Overview diagram.
 *
 * Lifecycle, not rendering: open the SSE stream while enabled, hand each frame
 * to the overlay builder, reconnect after a failure while still enabled, and
 * tear everything down on disable or unmount. The filter is part of the stream
 * identity — the server applies it before aggregating — so changing it closes
 * the current stream and opens a new one.
 *
 * WHY RECONNECT RATHER THAN FLIP THE TOGGLE OFF
 * ─────────────────────────────────────────────
 * The stream's most common failure is sing-box restarting, which the panel
 * itself triggers on every config save. A toggle that turned itself off on
 * each restart would need re-enabling after every edit — precisely when the
 * operator wants to watch the effect. So while the operator has Live on, a
 * dropped stream is retried on a short delay and the last error is shown in
 * the meantime. The service-status gate in the card stops the retries once
 * sing-box is genuinely down.
 */
import { computed, onBeforeUnmount, ref, shallowRef, watch, type Ref } from 'vue'
import { openTrafficFlowStream } from '../services/traffic'
import { buildFlowOverlay, type FlowOverlay } from '../utils/flowOverlay'
import { apiErrorMessage } from '../utils/apiErrorMessage'
import type { TrafficFilter, TrafficFrame } from '../types/trafficFlow'

const RECONNECT_DELAY_MS = 3_000

export function useTrafficFlow(enabled: Ref<boolean>, filter: Ref<TrafficFilter>, fallbackError: () => string) {
  /** Latest frame, re-keyed for the diagram. shallowRef: replaced whole, once a second. */
  const overlay = shallowRef<FlowOverlay | null>(null)
  const frame = shallowRef<TrafficFrame | null>(null)
  const error = ref('')
  const connecting = ref(false)

  let controller: AbortController | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

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

  const open = () => {
    close()
    if (!enabled.value) return

    const current = new AbortController()
    controller = current
    connecting.value = true

    void openTrafficFlowStream(
      filter.value,
      {
        onFrame: (next) => {
          connecting.value = false
          error.value = ''
          frame.value = next
          overlay.value = buildFlowOverlay(next)
        },
        onError: (err) => {
          error.value = apiErrorMessage(err, fallbackError())
        },
        onClose: () => {
          // A close belonging to a superseded controller is not ours to react to.
          if (controller !== current) return
          connecting.value = false
          controller = null
          if (enabled.value) {
            reconnectTimer = setTimeout(open, RECONNECT_DELAY_MS)
          }
        },
      },
      current.signal,
    )
  }

  watch(enabled, (on) => {
    if (on) {
      open()
    } else {
      close()
      overlay.value = null
      frame.value = null
      error.value = ''
    }
  })

  // A new filter is a new stream. The overlay is kept until the first frame
  // of the new stream arrives, so the diagram does not blink to empty.
  watch(filter, () => {
    if (enabled.value) open()
  }, { deep: true })

  onBeforeUnmount(close)

  const active = computed(() => enabled.value && overlay.value !== null)

  return { overlay, frame, error, connecting, active }
}
