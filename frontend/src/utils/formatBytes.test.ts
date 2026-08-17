import { describe, expect, it } from 'bun:test'
import { formatBytes } from './formatBytes'

describe('formatBytes', () => {
  it('keeps small values in bytes without a decimal', () => {
    expect(formatBytes(0)).toBe('0 B')
    expect(formatBytes(512)).toBe('512 B')
  })

  it('scales through binary units', () => {
    expect(formatBytes(1024)).toBe('1 KB')
    expect(formatBytes(1536)).toBe('1.5 KB')
    expect(formatBytes(1024 * 1024)).toBe('1 MB')
    expect(formatBytes(29.2 * 1024 ** 3)).toBe('29.2 GB')
    expect(formatBytes(3.6 * 1024 ** 4)).toBe('3.6 TB')
  })

  it('drops the trailing .0 so whole values read cleanly', () => {
    expect(formatBytes(2 * 1024 ** 3)).toBe('2 GB')
  })

  it('rounds to one decimal place', () => {
    expect(formatBytes(1024 ** 3 * 1.25)).toBe('1.3 GB')
  })

  it('returns a placeholder for values that are not usable numbers', () => {
    expect(formatBytes(Number.NaN)).toBe('—')
    expect(formatBytes(-1)).toBe('—')
    expect(formatBytes(Number.POSITIVE_INFINITY)).toBe('—')
  })
})
