import type { NodeDelaySample } from '../types/runtime'
import type { ProbePoint } from '../types/subprobe'

/** Preserve failed tests as gaps, never as zero-latency successes. */
export function nodeDelayPoints(history: readonly NodeDelaySample[] = []): ProbePoint[] {
  return history
    .filter(sample => Number.isFinite(Date.parse(sample.time)) && Number.isFinite(sample.delay) && sample.delay >= 0)
    .map(sample => ({
      at: sample.time,
      total: 1,
      reachable: sample.delay > 0 ? 1 : 0,
      avg_ms: sample.delay,
      min_ms: sample.delay,
      max_ms: sample.delay,
    }))
    .sort((a, b) => Date.parse(a.at) - Date.parse(b.at))
}
