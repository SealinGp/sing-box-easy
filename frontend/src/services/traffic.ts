import { openStream, type StreamHandlers } from './stream'
import type { TrafficFilter, TrafficFrame } from '../types/trafficFlow'

/**
 * Opens the live traffic stream.
 *
 * The filter travels as query parameters rather than a body because the
 * server applies it per connection before aggregating, so changing it means
 * a new stream — which is also the natural unit for "the operator typed a new
 * host to chase".
 */
export function openTrafficFlowStream(
  filter: TrafficFilter,
  handlers: {
    onFrame: (frame: TrafficFrame) => void
    onError?: StreamHandlers['onError']
    onClose?: StreamHandlers['onClose']
  },
  signal: AbortSignal,
): Promise<void> {
  const params = new URLSearchParams()
  if (filter.sourceIp.trim()) params.set('source_ip', filter.sourceIp.trim())
  if (filter.host.trim()) params.set('host', filter.host.trim())
  const query = params.toString()

  return openStream(`/traffic/flow/stream${query ? `?${query}` : ''}`, {
    signal,
    onEvent: (name, data) => {
      if (name === 'frame') handlers.onFrame(data as TrafficFrame)
    },
    onError: handlers.onError,
    onClose: handlers.onClose,
  })
}
