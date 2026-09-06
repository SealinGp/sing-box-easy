import type { ApiService } from './api'
import type { BasicResponse } from '../types/api'
import type {
  ProbeHistoryResponse,
  ProbeNodesResponse,
  ProbeRange,
  ProbeSettings,
  ProbeStatusResponse,
} from '../types/subprobe'

/**
 * Subscription quality metrics.
 *
 * Lives under /subscription-probes rather than /subscriptions/... because that
 * path already owns a ":id" wildcard at the next segment, which a static
 * sibling collides with in the backend's router.
 */
export class SubProbeService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  /** Scheduler state + the newest sample for every subscription, in one call. */
  async getStatus(): Promise<BasicResponse<ProbeStatusResponse>> {
    const response = await this.api.get<BasicResponse<ProbeStatusResponse>>(
      '/subscription-probes/status',
    )
    return response.data
  }

  /**
   * One subscription's series. The server buckets wide ranges, so the reply is
   * a few hundred points whatever the sampling interval is.
   */
  async getHistory(id: string, range: ProbeRange): Promise<BasicResponse<ProbeHistoryResponse>> {
    const response = await this.api.get<BasicResponse<ProbeHistoryResponse>>(
      `/subscription-probes/${encodeURIComponent(id)}/history?range=${encodeURIComponent(range)}`,
    )
    return response.data
  }

  /** The most recent run's per-node detail. In-memory server-side. */
  async getNodes(id: string): Promise<BasicResponse<ProbeNodesResponse>> {
    const response = await this.api.get<BasicResponse<ProbeNodesResponse>>(
      `/subscription-probes/${encodeURIComponent(id)}/nodes`,
    )
    return response.data
  }

  /**
   * Probe one subscription now.
   *
   * Synchronous server-side, and genuinely slow: every node is URL-tested, and
   * a node that is DOWN costs the full per-node timeout rather than failing
   * fast. Measured on a real feed, 103 unreachable nodes took 65s at the
   * server's 8-way concurrency.
   *
   * Hence the explicit timeout. The shared client's default is 30s, which is
   * shorter than that — the browser would abort a probe the server then
   * completed and stored, so the operator would be told it failed while the
   * chart quietly gained the point. 11 minutes sits just above the handler's
   * own 10-minute ceiling, so the server's error is what surfaces rather than
   * a bare client abort that says nothing about why.
   */
  async run(id: string): Promise<BasicResponse<{ sample: Omit<import('../types/subprobe').ProbePoint, 'at'> }>> {
    const response = await this.api.post<
      BasicResponse<{ sample: Omit<import('../types/subprobe').ProbePoint, 'at'> }>
    >(`/subscription-probes/${encodeURIComponent(id)}/run`, undefined, { timeout: 11 * 60 * 1000 })
    return response.data
  }

  /** Persist the probe knobs. Omitted fields are left alone. */
  async updateSettings(payload: {
    interval_seconds?: number
    timeout_ms?: number
    retention_days?: number
    max_points?: number
  }): Promise<BasicResponse<ProbeSettings>> {
    const response = await this.api.put<BasicResponse<ProbeSettings>>(
      '/subscription-probes/settings',
      payload,
    )
    return response.data
  }
}
