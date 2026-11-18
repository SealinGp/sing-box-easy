import { expect, test } from 'bun:test'
import { nodeDelayPoints } from './nodeDelayHistory'
import { layoutProbeChart } from './probeChart'

test('explicit outcome distinguishes a sub-millisecond success from a failure', () => {
  const points = nodeDelayPoints([
    { time: '2026-11-16T12:00:00Z', delay: 0, success: true },
    { time: '2026-11-16T12:01:00Z', delay: 0, success: false },
  ])
  expect(points.map(point => point.reachable)).toEqual([1, 0])
})

test('node history orders timestamps and keeps failures as chart gaps', () => {
  const points = nodeDelayPoints([
    { time: '2026-11-16T12:02:00Z', delay: 150 },
    { time: 'invalid', delay: 20 },
    { time: '2026-11-16T12:01:00Z', delay: 0 },
    { time: '2026-11-16T12:00:00Z', delay: 100 },
    { time: '2026-11-16T12:03:00Z', delay: -1 },
  ])
  expect(points.map(point => point.avg_ms)).toEqual([100, 0, 150])
  expect(points[1]!.reachable).toBe(0)
  const layout = layoutProbeChart(points, { width: 720, height: 132 })
  expect(layout.latency.segments).toHaveLength(2)
  expect(layout.latency.segments.map(segment => segment.points.length)).toEqual([1, 1])
})

test('missing history stays empty and a single sample stays a single point', () => {
  expect(nodeDelayPoints()).toEqual([])
  const points = nodeDelayPoints([{ time: '2026-11-16T12:00:00Z', delay: 90 }])
  const layout = layoutProbeChart(points, { width: 720, height: 132 })
  expect(layout.latency.segments[0]!.points).toHaveLength(1)
})
