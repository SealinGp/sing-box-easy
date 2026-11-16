import { expect, test } from 'bun:test'
import { DEFAULT_NODE_QUALITY as rules, nodeQualityErrors, rateNode } from './nodeQuality'

const samples = (delays: number[]) => delays.map((delay, i) => ({ time: new Date(1_800_000_000_000 + i * 60000).toISOString(), delay }))

test('partial windows use the actual denominator and include 500ms', () => {
  expect(rateNode(samples([500]), rules)).toMatchObject({ count: 1, passed: 1, level: 'good' })
  expect(rateNode(samples([100, 100, 100, 0]), rules)).toMatchObject({ count: 4, percent: 75, level: 'fair' })
  expect(rateNode(samples([100, 501, 0]), rules)).toMatchObject({ count: 3, passed: 1, level: 'poor' })
  expect(rateNode([], rules).level).toBe('unknown')
})

test('only the latest N valid observations affect a rating', () => {
  expect(rateNode(samples([0, 0, 0, 10, 20]), { ...rules, samples: 2 }).level).toBe('good')
  const data = samples([100, 0, 100, 0, 100])
  expect(rateNode(data, { ...rules, good_percent: 60 }).level).toBe('good')
  expect(rateNode(data, rules).level).toBe('fair')
})

test('reject invalid criteria and overlapping bands', () => {
  expect(Object.values(nodeQualityErrors(rules)).some(Boolean)).toBe(false)
  for (const change of [{ samples: 0 }, { samples: 1.5 }, { max_delay_ms: 0 }, { good_percent: 50 }, { fair_percent: 0 }, { good_percent: 101 }]) {
    expect(Object.values(nodeQualityErrors({ ...rules, ...change })).some(Boolean)).toBe(true)
  }
})
