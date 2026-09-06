/**
 * Geometry for the subscription quality chart.
 *
 * Pure and unit-tested, so the component stays markup — the pattern the route
 * flow diagram already uses (utils/flowLayout.ts). No chart library: this is
 * two line panels sharing an x-axis, and a dependency that renders it would be
 * larger than the panel's whole bundle for one screen.
 *
 * TWO PANELS, NOT TWO Y-AXES
 * ──────────────────────────
 * Availability and latency are unrelated quantities in unrelated units, and a
 * dual-axis chart forces the reader to work out which line belongs to which
 * scale before either number means anything. Stacked panels sharing an x-axis
 * cost a little vertical room and remove that question entirely.
 */
import type { ProbePoint } from '../types/subprobe'

export interface ChartSize {
  width: number
  height: number
}

/** A point placed on the canvas, with the sample it came from. */
export interface PlacedPoint {
  x: number
  y: number
  point: ProbePoint
}

/**
 * One unbroken run of samples. Several exist when sampling stopped and
 * restarted — see the gap rule below.
 */
export interface Segment {
  points: PlacedPoint[]
  /** `d` for the line. */
  line: string
  /** `d` for the filled area under the line, closed to the baseline. */
  area: string
}

export interface Panel {
  segments: Segment[]
  /** Top of this panel's value axis (100 for availability, ms for latency). */
  max: number
}

export interface AxisTick {
  x: number
  label: string
}

export interface ProbeChartLayout {
  isEmpty: boolean
  /** The drawable rectangle shared by both panels' x-axis. */
  plot: { x: number; y: number; width: number; height: number }
  availability: Panel
  latency: Panel
  xTicks: AxisTick[]
  /** Time span actually covered, for the caption. */
  from: number
  to: number
}

/* ── Geometry constants. One place, so the component invents no numbers. ──── */

const PAD_LEFT = 44 // room for the y-axis labels ("100%", "1.2 s")
const PAD_RIGHT = 8
const PAD_TOP = 8
const PAD_BOTTOM = 18 // room for the x-axis labels

/**
 * A gap wider than this multiple of the MEDIAN spacing breaks the line.
 *
 * Breaking matters more than it looks. The panel records nothing while sing-box
 * is down (a run it could not perform is not evidence of an outage), so a stop
 * leaves a hole — and a line drawn straight across it asserts an availability
 * that was never measured, exactly over the window where something was wrong.
 *
 * The median, not the configured interval: the series may be bucketed by the
 * server, and the client would otherwise have to reconstruct which bucket width
 * it was served at. 2.5 is loose enough to absorb one late sweep.
 */
const GAP_FACTOR = 2.5

/** Rounds a maximum up to a readable axis top. */
export function niceCeiling(value: number): number {
  if (!Number.isFinite(value) || value <= 0) return 50
  // A step of one tenth of the magnitude, floored at 50ms: round enough to
  // label, tight enough not to squash the line into the bottom third.
  // 43 → 50, 430 → 450, 1234 → 1300.
  const magnitude = Math.pow(10, Math.floor(Math.log10(value)))
  const step = Math.max(magnitude / 10, 50)
  return Math.ceil(value / step) * step
}

/**
 * Formats a latency for an axis or a readout.
 *
 * Rounded FIRST: these are derived means and arrive with floating-point noise,
 * so an unrounded value renders as "312.60000000000002 ms".
 */
export function formatLatency(ms: number): string {
  if (!Number.isFinite(ms) || ms <= 0) return '—'
  const rounded = Math.round(ms)
  if (rounded < 1000) return `${rounded} ms`
  return `${(rounded / 1000).toFixed(2)} s`
}

/** Formats availability as a whole percentage. */
export function formatAvailability(point: Pick<ProbePoint, 'reachable' | 'total'>): string {
  if (!point.total) return '—'
  return `${Math.round((point.reachable / point.total) * 100)}%`
}

/** The 0-1 availability ratio, guarding a zero denominator. */
export function availabilityRatio(point: Pick<ProbePoint, 'reachable' | 'total'>): number {
  if (!point.total) return 0
  return point.reachable / point.total
}

/**
 * Places both panels.
 *
 * `points` must be oldest-first, which is the order the API returns.
 */
export function layoutProbeChart(points: ProbePoint[], size: ChartSize): ProbeChartLayout {
  const plot = {
    x: PAD_LEFT,
    y: PAD_TOP,
    width: Math.max(size.width - PAD_LEFT - PAD_RIGHT, 1),
    height: Math.max(size.height - PAD_TOP - PAD_BOTTOM, 1),
  }

  const empty: ProbeChartLayout = {
    isEmpty: true,
    plot,
    availability: { segments: [], max: 100 },
    latency: { segments: [], max: 0 },
    xTicks: [],
    from: 0,
    to: 0,
  }
  if (points.length === 0) return empty

  const times = points.map((p) => new Date(p.at).getTime())
  if (times.some((t) => Number.isNaN(t))) {
    // A malformed timestamp would place every point at NaN and silently blank
    // the chart. Drop the bad rows rather than the whole series.
    const usable = points.filter((_, i) => !Number.isNaN(times[i]))
    if (usable.length === 0) return empty
    return layoutProbeChart(usable, size)
  }

  const from = times[0]!
  const to = times[times.length - 1]!
  // A single point (or several within the same second) has no span to scale
  // against; pin it to the middle rather than dividing by zero.
  const span = to - from
  const scaleX = (t: number) =>
    span > 0 ? plot.x + ((t - from) / span) * plot.width : plot.x + plot.width / 2

  const latencyValues = points.filter((p) => p.reachable > 0).map((p) => p.avg_ms)
  const latencyMax = niceCeiling(Math.max(...latencyValues, 0))

  const scaleY = (value: number, max: number) =>
    plot.y + plot.height - (max > 0 ? Math.min(value / max, 1) : 0) * plot.height

  const breaks = gapIndices(times)

  const availability = buildPanel(
    points,
    times,
    breaks,
    // Availability is drawn on an ABSOLUTE 0-100% axis, never scaled to the
    // data. A subscription that never dropped below 90% must not be drawn
    // hugging the floor, and one that never rose above 5% must not look full.
    () => true,
    (p) => availabilityRatio(p) * 100,
    100,
    scaleX,
    scaleY,
    plot,
  )

  const latency = buildPanel(
    points,
    times,
    breaks,
    // A run where nothing answered has no latency to plot. Its avg_ms is 0,
    // and drawing that would render a total outage as the fastest moment on
    // the chart — the exact opposite of what happened.
    (p) => p.reachable > 0,
    (p) => p.avg_ms,
    latencyMax,
    scaleX,
    scaleY,
    plot,
  )

  return {
    isEmpty: false,
    plot,
    availability,
    latency,
    xTicks: buildXTicks(from, to, scaleX),
    from,
    to,
  }
}

/**
 * gapIndices returns the indices that START a new segment.
 *
 * Derived from the median spacing so it works whatever bucket the server chose.
 */
function gapIndices(times: number[]): Set<number> {
  const breaks = new Set<number>()
  if (times.length < 3) return breaks

  const deltas: number[] = []
  for (let i = 1; i < times.length; i++) deltas.push(times[i]! - times[i - 1]!)

  const sorted = [...deltas].sort((a, b) => a - b)
  const median = sorted[Math.floor(sorted.length / 2)]
  if (!median) return breaks

  for (let i = 1; i < times.length; i++) {
    if (times[i]! - times[i - 1]! > median * GAP_FACTOR) breaks.add(i)
  }
  return breaks
}

/** Builds one panel's segments, honouring the shared breaks and a value filter. */
function buildPanel(
  points: ProbePoint[],
  times: number[],
  breaks: Set<number>,
  include: (point: ProbePoint) => boolean,
  value: (point: ProbePoint) => number,
  max: number,
  scaleX: (t: number) => number,
  scaleY: (value: number, max: number) => number,
  plot: ProbeChartLayout['plot'],
): Panel {
  const segments: Segment[] = []
  let current: PlacedPoint[] = []

  const flush = () => {
    if (current.length === 0) return
    segments.push({ points: current, line: linePath(current), area: areaPath(current, plot) })
    current = []
  }

  points.forEach((point, i) => {
    // A break in the shared time axis breaks BOTH panels, so the two never
    // disagree about when data exists.
    if (breaks.has(i)) flush()
    if (!include(point)) {
      // An excluded sample also interrupts the line: joining across it would
      // imply a measurement that was deliberately not plotted.
      flush()
      return
    }
    current.push({ x: scaleX(times[i]!), y: scaleY(value(point), max), point })
  })
  flush()

  return { segments, max }
}

function linePath(points: PlacedPoint[]): string {
  if (points.length === 1) {
    // A one-point "line" still needs a `d` the renderer can accept; a
    // zero-length line renders nothing, and the component draws the dot.
    const only = points[0]!
    return `M ${round(only.x)} ${round(only.y)}`
  }
  return points
    .map((p, i) => `${i === 0 ? 'M' : 'L'} ${round(p.x)} ${round(p.y)}`)
    .join(' ')
}

function areaPath(points: PlacedPoint[], plot: ProbeChartLayout['plot']): string {
  const baseline = round(plot.y + plot.height)
  const first = points[0]!
  const last = points[points.length - 1]!
  return `${linePath(points)} L ${round(last.x)} ${baseline} L ${round(first.x)} ${baseline} Z`
}

/** Time labels along the shared axis. */
function buildXTicks(from: number, to: number, scaleX: (t: number) => number): AxisTick[] {
  if (to <= from) {
    return [{ x: scaleX(from), label: formatTick(from, 0) }]
  }

  const span = to - from
  const count = 4
  const ticks: AxisTick[] = []
  for (let i = 0; i <= count; i++) {
    const t = from + (span * i) / count
    ticks.push({ x: scaleX(t), label: formatTick(t, span) })
  }
  return ticks
}

/**
 * A tick label shows the time for short spans and the date for long ones —
 * "14:20" is meaningless on a 30-day chart, and "Jan 3" is on a 1-hour one.
 */
function formatTick(time: number, span: number): string {
  const date = new Date(time)
  const pad = (n: number) => String(n).padStart(2, '0')
  if (span > 3 * 24 * 60 * 60 * 1000) {
    return `${date.getMonth() + 1}/${date.getDate()}`
  }
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

/** Two decimals is plenty for an SVG coordinate and keeps the DOM small. */
function round(n: number): number {
  return Math.round(n * 100) / 100
}
