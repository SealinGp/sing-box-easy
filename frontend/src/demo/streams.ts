import type { ConnectionsFrame } from '../types/runtime'
import type { TrafficFrame } from '../types/trafficFlow'
import { proxies } from './data'

export function connectionFrame(at: number): ConnectionsFrame {
  const down = Math.round((2.5 + Math.sin(at / 7000) * 1.2) * 1024 ** 2)
  const up = Math.round((.35 + Math.sin(at / 5000) * .12) * 1024 ** 2)
  return { at, connections: [], totals: { connections: Math.round(86 + Math.sin(at / 9000) * 15),
    down_rate: down, up_rate: up, download_total: 38.6 * 1024 ** 3, upload_total: 4.2 * 1024 ** 3, memory: 78 * 1024 ** 2 } }
}
export function trafficFrame(at: number): TrafficFrame {
  const { totals } = connectionFrame(at)
  const down = totals.down_rate, up = totals.up_rate, count = totals.connections
  return { at, totals: { down, up, connections: count, all: count, closed: 0 }, filtered: false, unmatched: 0,
    sources: [{ ip: '192.0.2.20', down, up, connections: count }],
    inbounds: [{ tag: 'tun-in', down, up, connections: count }],
    rules: [{ kind: 'rule', index: 6, exit: '自动选择', down, up, connections: count, hosts: [{ host: 'example.com', down }] }],
    exits: [{ tag: '自动选择', down, up, connections: count, via: [{ tag: proxies.groups[1]!.now, down, connections: count }] }] }
}
/** Only the standalone entry installs this; unknown fetches never reach a network. */
export async function demoFetch(input: RequestInfo | URL, init?: RequestInit) {
  const url = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url
  const path = new URL(url, 'https://demo.invalid').pathname
  const connections = path === '/api/v1/runtime/connections/stream'
  if (!connections && path !== '/api/v1/traffic/flow/stream') throw new Error('静态演示：安装后使用此操作。 / Static demo: install the app to use this action.')
  const signal = init?.signal ?? (input instanceof Request ? input.signal : undefined)
  const encoder = new TextEncoder()
  let cleanup = () => {}
  const body = new ReadableStream<Uint8Array>({
    start(controller) {
      let closed = false
      let timer: ReturnType<typeof setInterval> | undefined
      cleanup = () => {
        if (closed) return
        closed = true
        clearInterval(timer)
        signal?.removeEventListener('abort', cleanup)
        controller.close()
      }
      const send = (at: number) => {
        if (closed) return
        const data = connections ? connectionFrame(at) : trafficFrame(at)
        controller.enqueue(encoder.encode(`event: frame\ndata: ${JSON.stringify({ code: 0, msg: 'demo', data })}\n\n`))
      }
      if (signal?.aborted) { cleanup(); return }
      signal?.addEventListener('abort', cleanup, { once: true })
      const now = Date.now()
      for (let i = connections ? 59 : 0; i >= 0; i--) send(now - i * 1000)
      timer = setInterval(() => send(Date.now()), 1000)
    },
    cancel() { cleanup() },
  })
  return new Response(body, { headers: { 'Content-Type': 'text/event-stream' } })
}

export function installDemoStreams() {
  // Bun adds fetch.preconnect to the ambient type; the browser has no such member.
  window.fetch = demoFetch as typeof window.fetch
}
