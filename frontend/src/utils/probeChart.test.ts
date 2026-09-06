import { describe, expect, it } from 'bun:test'
import { layoutProbeChart, niceCeiling, formatLatency } from './probeChart'
import type { ProbePoint } from '../types/subprobe'

/** Builds a point `minutesAgo` before a fixed epoch. */
const at = (minutesAgo: number) =>
  new Date(Date.UTC(2026, 0, 1, 12, 0, 0) - minutesAgo * 60_000).toISOString()

const point = (minutesAgo: number, reachable: number, total: number, avg: number): ProbePoint => ({
  at: at(minutesAgo),
  total,
  reachable,
  avg_ms: avg,
  min_ms: avg,
  max_ms: avg,
})

const SIZE = { width: 600, height: 200 }

describe('layoutProbeChart', () => {
  it('returns an empty layout for no points rather than NaN geometry', () => {
    const layout = layoutProbeChart([], SIZE)
    expect(layout.isEmpty).toBe(true)
    expect(layout.availability.segments).toEqual([])
    expect(layout.latency.segments).toEqual([])
    // A zero-width scale would divide by zero downstream.
    expect(Number.isFinite(layout.plot.width)).toBe(true)
  })

  it('places a single point without dividing by a zero time span', () => {
    const layout = layoutProbeChart([point(0, 2, 2, 100)], SIZE)
    expect(layout.isEmpty).toBe(false)
    const segment = layout.availability.segments[0]!
    expect(segment.points).toHaveLength(1)
    expect(Number.isFinite(segment.points[0]!.x)).toBe(true)
    expect(Number.isFinite(segment.points[0]!.y)).toBe(true)
  })

  it('maps availability onto a fixed 0-100% axis, oldest on the left', () => {
    const layout = layoutProbeChart(
      [point(60, 0, 4, 0), point(30, 2, 4, 100), point(0, 4, 4, 100)],
      SIZE,
    )
    const segment = layout.availability.segments[0]!
    expect(segment.points).toHaveLength(3)

    // Oldest first, left to right.
    expect(segment.points[0]!.x).toBeLessThan(segment.points[2]!.x)

    // 0% sits on the baseline, 100% at the top. SVG y grows downward, so a
    // higher availability must have a SMALLER y.
    expect(segment.points[0]!.y).toBeGreaterThan(segment.points[2]!.y)
    expect(segment.points[1]!.y).toBeGreaterThan(segment.points[2]!.y)

    // The axis is absolute, not relative to the data: a subscription that
    // never dropped below 90% must not be drawn touching the floor.
    const allUp = layoutProbeChart([point(30, 9, 10, 100), point(0, 10, 10, 100)], SIZE)
    const upSegment = allUp.availability.segments[0]!
    expect(upSegment.points[0]!.y).toBeLessThan(layout.plot.y + layout.plot.height * 0.5)
  })

  it('breaks the line where sampling stopped instead of drawing across the gap', () => {
    // Two 10-minute-spaced runs, a four-hour hole, then two more. The hole is
    // sing-box being down — joining across it would draw an availability the
    // panel never measured.
    const layout = layoutProbeChart(
      [
        point(300, 4, 4, 100),
        point(290, 4, 4, 100),
        point(50, 4, 4, 100),
        point(40, 4, 4, 100),
      ],
      SIZE,
    )
    expect(layout.availability.segments).toHaveLength(2)
    expect(layout.availability.segments[0]!.points).toHaveLength(2)
    expect(layout.availability.segments[1]!.points).toHaveLength(2)
    // The same break must apply to both panels, or the two charts would
    // disagree about when the data exists.
    expect(layout.latency.segments).toHaveLength(2)
  })

  it('keeps evenly-spaced samples in one segment', () => {
    const points = [0, 10, 20, 30, 40].map((m) => point(m, 4, 4, 100)).reverse()
    const layout = layoutProbeChart(points, SIZE)
    expect(layout.availability.segments).toHaveLength(1)
    expect(layout.availability.segments[0]!.points).toHaveLength(5)
  })

  it('scales latency to a rounded ceiling above the peak', () => {
    const layout = layoutProbeChart([point(30, 4, 4, 120), point(0, 4, 4, 430)], SIZE)
    expect(layout.latency.max).toBeGreaterThanOrEqual(430)
    // Rounded, so the axis label is readable rather than "437ms".
    expect(layout.latency.max % 50).toBe(0)
    // The peak must not sit exactly on the top edge, where it reads as clipped.
    const segment = layout.latency.segments[0]!
    expect(segment.points[1]!.y).toBeGreaterThan(layout.plot.y)
  })

  it('excludes runs where nothing answered from the latency line', () => {
    // A run with zero reachable nodes has no latency. Plotting its 0 would
    // draw a dead subscription as an infinitely fast one.
    const layout = layoutProbeChart(
      [point(20, 4, 4, 300), point(10, 0, 4, 0), point(0, 4, 4, 300)],
      SIZE,
    )
    const plotted = layout.latency.segments.flatMap((s) => s.points)
    expect(plotted).toHaveLength(2)

    // Availability, by contrast, MUST include it — that run is the whole point.
    const availabilityPoints = layout.availability.segments.flatMap((s) => s.points)
    expect(availabilityPoints).toHaveLength(3)
  })

  it('produces an area path under the availability line for the fill', () => {
    const layout = layoutProbeChart([point(20, 4, 4, 100), point(0, 2, 4, 100)], SIZE)
    const segment = layout.availability.segments[0]!
    expect(segment.line).toStartWith('M')
    expect(segment.area).toStartWith('M')
    expect(segment.area).toEndWith('Z')
  })

  it('emits x ticks inside the plot area', () => {
    const layout = layoutProbeChart(
      Array.from({ length: 24 }, (_, i) => point(i * 60, 4, 4, 100)).reverse(),
      SIZE,
    )
    expect(layout.xTicks.length).toBeGreaterThan(1)
    for (const tick of layout.xTicks) {
      expect(tick.x).toBeGreaterThanOrEqual(layout.plot.x - 0.01)
      expect(tick.x).toBeLessThanOrEqual(layout.plot.x + layout.plot.width + 0.01)
      expect(tick.label).not.toBe('')
    }
  })
})

describe('niceCeiling', () => {
  it('rounds up to a readable step', () => {
    expect(niceCeiling(0)).toBe(50)
    expect(niceCeiling(43)).toBe(50)
    expect(niceCeiling(430)).toBe(450)
    expect(niceCeiling(1234)).toBe(1300)
  })
})

describe('formatLatency', () => {
  it('keeps milliseconds readable and drops noise', () => {
    expect(formatLatency(0)).toBe('—')
    expect(formatLatency(312)).toBe('312 ms')
    // Derived means carry floating-point noise; a chart axis must not.
    expect(formatLatency(312.6)).toBe('313 ms')
    expect(formatLatency(1240)).toBe('1.24 s')
  })
})
