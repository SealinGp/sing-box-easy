import type { ApiService } from './api'
import type { BasicResponse } from '../types/api'
import type { DualStackRequest, DualStackResult } from '../types/dualstack'
import { openStream } from './stream'

/**
 * The dual-stack domain test.
 *
 * Its own service rather than a method on `dns` or `route`: the probe spans
 * both — DNS answers, routing prediction, and real traffic — and hanging it
 * off either one would misdescribe it.
 */
export class DiagnosticsService {
  private api: ApiService

  constructor(api: ApiService) {
    this.api = api
  }

  /**
   * Runs the probe in one call.
   *
   * The explicit timeout is not decoration. The shared axios default is 30s,
   * but this endpoint dials up to 10 domains over 2 families serially, each
   * bounded by the server's own per-dial ceiling — so a list of unreachable
   * addresses would be aborted by the browser and reported as failed while
   * the server was still completing it.
   */
  async probe(request: DualStackRequest): Promise<DualStackResult> {
    const response = await this.api.post<BasicResponse<DualStackResult>>(
      '/diagnostics/dual-stack',
      request,
      { timeout: 6 * 60 * 1000 },
    )
    return response.data.data
  }

  /**
   * The same probe, reported phase by phase.
   *
   * Worth the extra call site because the phases carry wildly different
   * latencies: the config read is instant, each resolve is one round trip,
   * and the traffic phase is a dial plus connection-table polls per address.
   * Unary, that is one silent multi-second wait.
   *
   * Aborting the signal cancels the probe server-side, which matters more
   * here than for the DNS probe: the remaining work opens real connections.
   */
  async probeStream(
    request: DualStackRequest,
    onStage: (stage: string, partial: DualStackResult) => void,
    signal: AbortSignal,
  ): Promise<DualStackResult | null> {
    let final: DualStackResult | null = null
    let failure: Error | null = null

    await openStream('/diagnostics/dual-stack/stream', {
      body: request,
      signal,
      onEvent: (name, data) => {
        const partial = data as DualStackResult
        if (name === 'done') final = partial
        else onStage(name, partial)
      },
      onError: (err) => {
        failure = err
      },
    })

    if (failure) throw failure
    return final
  }
}
