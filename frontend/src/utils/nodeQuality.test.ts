import { expect, test } from 'bun:test'
import { DEFAULT_NODE_QUALITY as rules, NODE_QUALITY_WATERMARKS, nodeQualityErrors, rateNode } from './nodeQuality'

const samples = (delays: number[]) => delays.map((delay, i) => ({ time: new Date(1_800_000_000_000 + i * 60000).toISOString(), delay }))
const windowWith = (passed: number, count = 100) => samples([...Array(passed).fill(500), ...Array(count - passed).fill(0)])

test('defaults use 100 probes, 500ms, and the 97% watermark', () => {
  expect(rules).toMatchObject({ samples: 100, max_delay_ms: 500, good_percent: 97 })
  expect(NODE_QUALITY_WATERMARKS).toEqual([85, 93, 95, 97])
})

test('partial windows show actual pass rates without recommending even perfect samples', () => {
  expect(rateNode(samples([500]), rules)).toMatchObject({ count: 1, passed: 1, percent: 100, level: 'insufficient', complete: false })
  expect(rateNode(samples([100, 100, 100, 0]), rules)).toMatchObject({ count: 4, percent: 75, level: 'insufficient' })
  expect(rateNode(windowWith(99, 99), rules).level).toBe('insufficient')
  expect(rateNode([], rules).level).toBe('unknown')
})

test('all four watermarks include the boundary and count failed probes in the denominator', () => {
  for (const threshold of NODE_QUALITY_WATERMARKS) {
    const criteria = { ...rules, good_percent: threshold }
    expect(rateNode(windowWith(threshold), criteria)).toMatchObject({ count: 100, passed: threshold, level: 'good' })
    expect(rateNode(windowWith(threshold - 1), criteria).level).toBe('poor')
  }
  const data = windowWith(100)
  data[0] = { ...data[0]!, delay: 501 }
  expect(rateNode(data, rules).passed).toBe(99)
  expect(rateNode([{ ...data[0]!, delay: 100, success: false }, ...data.slice(1)], rules).passed).toBe(99)
})

test('classification uses the unrounded rate', () => {
  expect(rateNode(windowWith(96, 99), { ...rules, samples: 99 })).toMatchObject({ level: 'poor', complete: true })
})

test('only the latest window of valid observations affects the rating', () => {
  const data = [...samples(Array(100).fill(0)), ...windowWith(97).map((p, i) => ({ ...p, time: new Date(1_900_000_000_000 + i * 60000).toISOString() }))]
  expect(rateNode(data, rules)).toMatchObject({ count: 100, passed: 97, level: 'good' })
  expect(rateNode([...data].reverse(), rules).level).toBe('good')
  expect(rateNode([{ time: 'invalid', delay: 1 }, ...windowWith(99, 99)], rules).level).toBe('insufficient')
})

test('legacy fair thresholds no longer change recommendations', () => {
  expect(rateNode(windowWith(96), { ...rules, fair_percent: 99 }).level).toBe('poor')
  expect(rateNode(windowWith(97), { ...rules, fair_percent: 99 }).level).toBe('good')
})

test('reject invalid windows, latency limits, and watermarks', () => {
  expect(Object.values(nodeQualityErrors(rules)).some(Boolean)).toBe(false)
  for (const change of [{ samples: 0 }, { samples: 101 }, { samples: 1.5 }, { max_delay_ms: 0 }, { good_percent: 0 }, { good_percent: 101 }, { good_percent: 97.5 }]) {
    expect(Object.values(nodeQualityErrors({ ...rules, ...change })).some(Boolean)).toBe(true)
  }
})
