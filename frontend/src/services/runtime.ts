import type { ApiService } from './api'
import type { BasicResponse } from '../types/api'
import type { ConnectionsFrame, GroupDelays, NodeDelay, RuntimeProxies } from '../types/runtime'
import { openStream, type StreamHandlers } from './stream'

/**
 * A group test dials every member with its own 5s deadline, ten at a time, so
 * a large group with many dead nodes outlasts the shared 30s axios default
 * (the server's own ceiling is 90s). Without this the browser would abort a
 * test the server went on to finish.
 */
const GROUP_TEST_TIMEOUT_MS = 100_000

/**
 * The running sing-box: groups, latency tests, node switching, connections.
 *
 * Outbound tags travel in the request BODY, never the path — a tag is free
 * text with spaces, emoji, "|" and sometimes "/".
 */
export class RuntimeService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  async getProxies(): Promise<BasicResponse<RuntimeProxies>> {
    const response = await this.api.get<BasicResponse<RuntimeProxies>>('/runtime/proxies')
    return response.data
  }

  /** Switches a selector in the running process. Admin-only. */
  async select(group: string, name: string): Promise<BasicResponse<unknown>> {
    const response = await this.api.put<BasicResponse<unknown>>('/runtime/proxies/selection', { group, name })
    return response.data
  }

  async testNode(name: string): Promise<BasicResponse<NodeDelay>> {
    const response = await this.api.post<BasicResponse<NodeDelay>>('/runtime/proxies/delay', { name })
    return response.data
  }

  async testGroup(name: string): Promise<BasicResponse<GroupDelays>> {
    const response = await this.api.post<BasicResponse<GroupDelays>>('/runtime/proxies/group-delay', { name }, {
      timeout: GROUP_TEST_TIMEOUT_MS,
    })
    return response.data
  }

  /** Admin-only. */
  async closeConnection(id: string): Promise<BasicResponse<unknown>> {
    const response = await this.api.delete<BasicResponse<unknown>>(`/runtime/connections/${encodeURIComponent(id)}`)
    return response.data
  }

  /** Admin-only. */
  async closeAllConnections(): Promise<BasicResponse<unknown>> {
    const response = await this.api.delete<BasicResponse<unknown>>('/runtime/connections')
    return response.data
  }
}

/** Opens the live connections stream: one frame per second. */
export function openConnectionsStream(
  handlers: {
    onFrame: (frame: ConnectionsFrame) => void
    onError?: StreamHandlers['onError']
    onClose?: StreamHandlers['onClose']
  },
  signal: AbortSignal,
  /** `summary: true` asks for totals only — no rows. The Overview card's mode. */
  options: { summary?: boolean } = {},
): Promise<void> {
  return openStream(`/runtime/connections/stream${options.summary ? '?rows=0' : ''}`, {
    signal,
    onEvent: (name, data) => {
      if (name === 'frame') handlers.onFrame(data as ConnectionsFrame)
    },
    onError: handlers.onError,
    onClose: handlers.onClose,
  })
}
