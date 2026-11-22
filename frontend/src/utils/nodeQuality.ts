import { nodeDelayPoints } from './nodeDelayHistory'
import type { NodeDelaySample } from '../types/runtime'

export interface NodeQualityRules {
  samples: number
  max_delay_ms: number
  good_percent: number
  /** Legacy wire field, retained for compatibility; no longer used for ratings. */
  fair_percent: number
}

export const DEFAULT_NODE_QUALITY: NodeQualityRules = {
  samples: 100, max_delay_ms: 500, good_percent: 97, fair_percent: 50,
}

export const NODE_QUALITY_WATERMARKS = [85, 93, 95, 97] as const

export function nodeQualityErrors(rules: NodeQualityRules) {
  return {
    samples: !Number.isInteger(rules.samples) || rules.samples < 1 || rules.samples > 100,
    max_delay_ms: !Number.isInteger(rules.max_delay_ms) || rules.max_delay_ms < 1 || rules.max_delay_ms > 60000,
    good_percent: !Number.isInteger(rules.good_percent) || rules.good_percent < 1 || rules.good_percent > 100,
  }
}

export function rateNode(history: readonly NodeDelaySample[] | undefined, rules: NodeQualityRules) {
  const points = nodeDelayPoints(history).slice(-rules.samples)
  const count = points.length
  const passed = points.filter(p => p.reachable > 0 && p.avg_ms <= rules.max_delay_ms).length
  // Compare before formatting: a displayed 97% must not promote 96.97%.
  const percent = count ? passed * 100 / count : 0
  const complete = count >= rules.samples
  const level = !count ? 'unknown' : !complete ? 'insufficient'
    : percent >= rules.good_percent ? 'good' : 'poor'
  return { count, passed, percent, complete, level } as const
}
