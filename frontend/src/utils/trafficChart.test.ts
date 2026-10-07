import { describe, expect, test } from 'bun:test'
import {
  MIN_RATE_CEILING, countCeiling, pushSample, rateCeiling, sampleIndexAt, sampleX, seriesPath, valueY,
  type TrafficSample,
} from './trafficChart'

const sample = (at: number): TrafficSample => ({ at, down: at, up: 0, connections: 1 })
const KB = 1024
const MB = 1024 * 1024

describe('pushSample', () => {
  test('appends, caps at the limit, and never mutates the input', () => {
    const history = [sample(1), sample(2), sample(3)]
    const next = pushSample(history, sample(4), 3)
    expect(next.map((s) => s.at)).toEqual([2, 3, 4])
    expect(history.map((s) => s.at)).toEqual([1, 2, 3])
  })
})

describe('rateCeiling', () => {
  test('an idle link gets the floor, not a zero-height axis', () => {
    expect(rateCeiling(0)).toBe(MIN_RATE_CEILING)
    expect(rateCeiling(300)).toBe(MIN_RATE_CEILING)
    expect(rateCeiling(Number.NaN)).toBe(MIN_RATE_CEILING)
  })
  test('tops out on values that format cleanly in binary units', () => {
    expect(rateCeiling(11 * KB)).toBe(20 * KB)
    expect(rateCeiling(420 * KB)).toBe(500 * KB)
    expect(rateCeiling(600 * KB)).toBe(1 * MB)
    expect(rateCeiling(1.2 * MB)).toBe(2 * MB)
    expect(rateCeiling(7.8 * MB)).toBe(10 * MB)
    expect(rateCeiling(130 * MB)).toBe(200 * MB)
  })
  test('the ceiling is never below the peak, including exactly on a step', () => {
    for (const peak of [10 * KB + 1, 500 * KB, 1 * MB, 1 * MB + 1, 999 * MB, 3.3 * 1024 * MB]) {
      expect(rateCeiling(peak)).toBeGreaterThanOrEqual(peak)
    }
  })
})

describe('countCeiling', () => {
  test('1-2-5 steps with a floor of 5', () => {
    expect(countCeiling(0)).toBe(5)
    expect(countCeiling(4)).toBe(5)
    expect(countCeiling(8)).toBe(10)
    expect(countCeiling(87)).toBe(100)
    expect(countCeiling(101)).toBe(200)
    expect(countCeiling(1300)).toBe(2000)
    expect(countCeiling(100)).toBe(100)
  })
})

describe('sampleX', () => {
  test('the newest sample sits on the right edge whatever the fill', () => {
    expect(sampleX(0, 1, 590, 60)).toBe(590)
    expect(sampleX(9, 10, 590, 60)).toBe(590)
    expect(sampleX(59, 60, 590, 60)).toBe(590)
  })
  test('spacing comes from capacity, so a half-full window leaves the left empty', () => {
    expect(sampleX(8, 10, 590, 60)).toBe(580)
    expect(sampleX(0, 10, 590, 60)).toBe(500)
    expect(sampleX(0, 60, 590, 60)).toBe(0)
  })
})

describe('valueY', () => {
  test('zero is the baseline, the ceiling is the top, and overshoot is clamped', () => {
    expect(valueY(0, 100, 80)).toBe(80)
    expect(valueY(100, 100, 80)).toBe(0)
    expect(valueY(50, 100, 80)).toBe(40)
    expect(valueY(500, 100, 80)).toBe(0)
    expect(valueY(-5, 100, 80)).toBe(80)
  })
})

describe('seriesPath', () => {
  test('a single sample draws nothing rather than a dot pretending to be a trend', () => {
    expect(seriesPath([5], 10, { width: 100, height: 50 })).toEqual({ line: '', area: '' })
  })
  test('the area closes along the baseline under the drawn span only', () => {
    const path = seriesPath([0, 10], 10, { width: 100, height: 50 }, 11)
    expect(path.line).toBe('M90,50L100,0')
    expect(path.area).toBe('M90,50L100,0L100,50L90,50Z')
  })
})

describe('sampleIndexAt', () => {
  test('the right edge is the newest sample and the left edge the oldest slot', () => {
    expect(sampleIndexAt(1, 60, 60)).toBe(59)
    expect(sampleIndexAt(0, 60, 60)).toBe(0)
  })
  test('over the part of the window with no data yet there is no sample', () => {
    expect(sampleIndexAt(0, 10, 60)).toBe(-1)
    expect(sampleIndexAt(1, 10, 60)).toBe(9)
    expect(sampleIndexAt(1, 0, 60)).toBe(-1)
  })
  test('out-of-range pointers clamp to the edges', () => {
    expect(sampleIndexAt(1.4, 60, 60)).toBe(59)
    expect(sampleIndexAt(-0.2, 60, 60)).toBe(0)
  })
})
