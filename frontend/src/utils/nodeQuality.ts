import { nodeDelayPoints } from './nodeDelayHistory'
import type { NodeDelaySample } from '../types/runtime'

export interface NodeQualityRules {
  samples: number
  max_delay_ms: number
  good_percent: number
  fair_percent: number
}

export const DEFAULT_NODE_QUALITY: NodeQualityRules = {
  samples: 10, max_delay_ms: 500, good_percent: 80, fair_percent: 50,
}

export function nodeQualityErrors(rules: NodeQualityRules) {
  return {
    samples: !Number.isInteger(rules.samples) || rules.samples < 1 || rules.samples > 100,
    max_delay_ms: !Number.isInteger(rules.max_delay_ms) || rules.max_delay_ms < 1 || rules.max_delay_ms > 60000,
    good_percent: !Number.isInteger(rules.good_percent) || rules.good_percent > 100 || rules.good_percent <= rules.fair_percent,
    fair_percent: !Number.isInteger(rules.fair_percent) || rules.fair_percent < 1 || rules.fair_percent >= rules.good_percent,
  }
}

export function rateNode(history: readonly NodeDelaySample[] | undefined, rules: NodeQualityRules) {
  const points = nodeDelayPoints(history).slice(-rules.samples)
  const count = points.length
  const passed = points.filter(p => p.reachable > 0 && p.avg_ms <= rules.max_delay_ms).length
  // Compare without rounding: 79.9% must not meet an 80% threshold.
  const percent = count ? passed * 100 / count : 0
  const level = !count ? 'unknown' : percent >= rules.good_percent ? 'good'
    : percent >= rules.fair_percent ? 'fair' : 'poor'
  return { count, passed, percent, level } as const
}
