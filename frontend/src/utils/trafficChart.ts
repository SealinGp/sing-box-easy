/**
 * Geometry for the Overview's connections card: a short rolling window of
 * speed and connection-count samples, one per second.
 *
 * Kept out of the `.vue` file, like the other charts here, so the parts that
 * are easy to get subtly wrong are tested directly: where a half-full window
 * sits, what the axis tops out at, and which sample a pointer is over.
 */

export interface TrafficSample {
  /** Sample time, unix ms. */
  at: number
  /** Bytes per second. */
  down: number
  up: number
  connections: number
}

/** One sample a second, so this is also the window in seconds. */
export const WINDOW_SAMPLES = 60

/**
 * Appends a sample and drops what has scrolled out of the window. Returns a
 * new array; the previous one is what the last render is still holding.
 */
export function pushSample(history: readonly TrafficSample[], sample: TrafficSample, limit = WINDOW_SAMPLES): TrafficSample[] {
  const next = [...history, sample]
  return next.length > limit ? next.slice(next.length - limit) : next
}

const BYTE_STEPS = [1, 2, 5, 10, 20, 50, 100, 200, 500]

/** The smallest axis the speed chart ever uses; below it the line is noise. */
export const MIN_RATE_CEILING = 10 * 1024

/**
 * Rounds a peak rate up to a readable axis top IN BINARY UNITS.
 *
 * A decimal "nice number" lands on values like 1,000,000 B/s, which the
 * formatter then prints as "976.6 KB". Stepping 1-2-5 within each power of
 * 1024 gives tops that format cleanly: 10 KB, 500 KB, 2 MB, 50 MB.
 */
export function rateCeiling(peak: number): number {
  if (!Number.isFinite(peak) || peak <= MIN_RATE_CEILING) return MIN_RATE_CEILING
  let unit = 1024
  while (peak > unit * 1024) unit *= 1024
  // `peak` is now within (unit, unit * 1024], so some step always fits; the
  // fallback is the next unit up, which is where 1024 × unit rounds to.
  const step = BYTE_STEPS.find((candidate) => candidate * unit >= peak)
  return step === undefined ? unit * 1024 : step * unit
}

/** Rounds a peak count up to a 1-2-5 axis top, never below 5. */
export function countCeiling(peak: number): number {
  if (!Number.isFinite(peak) || peak <= 5) return 5
  const magnitude = Math.pow(10, Math.floor(Math.log10(peak)))
  const step = [1, 2, 5, 10].find((candidate) => candidate * magnitude >= peak) ?? 10
  return step * magnitude
}

export interface ChartSize {
  width: number
  height: number
}

export interface SeriesPath {
  /** The line itself. Empty with fewer than two samples. */
  line: string
  /** The same line closed down to the baseline, for an area fill. */
  area: string
}

/**
 * X of sample `index` out of `count`.
 *
 * The window is anchored on the RIGHT: the newest sample is always at the
 * right edge and a half-full window leaves its empty half on the left. Spacing
 * is fixed by the window's capacity, not by how many samples have arrived —
 * stretching ten samples across the full width would make the first ten
 * seconds look like a minute, then visibly compress as the window filled.
 */
export function sampleX(index: number, count: number, width: number, capacity = WINDOW_SAMPLES): number {
  const step = width / Math.max(1, capacity - 1)
  return width - (count - 1 - index) * step
}

/** Y of `value` on an axis running 0 (bottom) to `ceiling` (top), clamped. */
export function valueY(value: number, ceiling: number, height: number): number {
  if (ceiling <= 0) return height
  const ratio = Math.min(1, Math.max(0, value / ceiling))
  return height - ratio * height
}

const round = (value: number) => Math.round(value * 100) / 100

export function seriesPath(values: readonly number[], ceiling: number, size: ChartSize, capacity = WINDOW_SAMPLES): SeriesPath {
  if (values.length < 2) return { line: '', area: '' }
  const points = values.map((value, index) =>
    `${round(sampleX(index, values.length, size.width, capacity))},${round(valueY(value, ceiling, size.height))}`)
  const line = `M${points.join('L')}`
  const firstX = round(sampleX(0, values.length, size.width, capacity))
  return { line, area: `${line}L${size.width},${size.height}L${firstX},${size.height}Z` }
}

/**
 * The sample under a pointer at horizontal `ratio` (0 = left edge, 1 = right),
 * or -1 when the pointer is over the part of the window that has no data yet.
 */
export function sampleIndexAt(ratio: number, count: number, capacity = WINDOW_SAMPLES): number {
  if (count === 0 || !Number.isFinite(ratio)) return -1
  const slots = Math.max(1, capacity - 1)
  const fromRight = Math.round((1 - Math.min(1, Math.max(0, ratio))) * slots)
  const index = count - 1 - fromRight
  return index >= 0 ? index : -1
}
