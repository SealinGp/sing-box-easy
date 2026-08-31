import { describe, expect, it } from 'bun:test'
import { formatHoursToDuration, isValidDuration, parseDurationToHours } from './dayjs'

describe('parseDurationToHours', () => {
  it('parses the units the form advertises', () => {
    expect(parseDurationToHours('24h')).toBe(24)
    expect(parseDurationToHours('1hour')).toBe(1)
    expect(parseDurationToHours('7d')).toBe(168)
    expect(parseDurationToHours('2w')).toBe(336)
    expect(parseDurationToHours('1mo')).toBe(720)
    // A bare number means hours.
    expect(parseDurationToHours('12')).toBe(12)
  })

  it('keeps sub-hour intervals as fractions', () => {
    // Rounding these to whole hours yielded 0, and `hoursSinceUpdate >= 0` is
    // true one second after a successful fetch — so every subscription on a
    // minute-scale cadence was painted permanently "outdated".
    expect(parseDurationToHours('5m')).toBeCloseTo(5 / 60, 10)
    expect(parseDurationToHours('30min')).toBeCloseTo(0.5, 10)
    expect(parseDurationToHours('90s')).toBeCloseTo(90 / 3600, 10)
    expect(parseDurationToHours('1s')).toBeGreaterThan(0)
  })

  it('rejects what it cannot understand', () => {
    for (const input of ['', undefined, 'soon', '24 hours later', '-5h', '1.5h']) {
      expect(parseDurationToHours(input)).toBeNull()
    }
    expect(isValidDuration('24h')).toBe(true)
    expect(isValidDuration('nonsense')).toBe(false)
  })
})

describe('formatHoursToDuration', () => {
  it('renders fractions as minutes', () => {
    expect(formatHoursToDuration(5 / 60)).toBe('5min')
    expect(formatHoursToDuration(0.5)).toBe('30min')
  })
})
