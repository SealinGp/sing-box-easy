import { describe, expect, it } from 'bun:test'
import { subscriptionHealth } from './subscriptionHealth'

const HOUR = 60 * 60 * 1000
const now = Date.parse('2026-08-31T18:00:00Z')
const ago = (ms: number) => new Date(now - ms).toISOString()

describe('subscriptionHealth', () => {
  it('is "never" until the first successful fetch', () => {
    expect(subscriptionHealth({ auto_update: true, update_interval: '24h' }, now)).toBe('never')
  })

  it('is "outdated" only once the interval has actually elapsed', () => {
    const sub = { last_update: ago(2 * HOUR), auto_update: true, update_interval: '24h' }
    expect(subscriptionHealth(sub, now)).toBe('ok')
    expect(
      subscriptionHealth({ ...sub, last_update: ago(25 * HOUR) }, now),
    ).toBe('outdated')
  })

  // The bug the overview tooltip exposed: "5m" rounded to 0 hours, so a
  // subscription fetched seconds ago was already "outdated" — and the dot went
  // amber permanently for anyone on a minute-scale cadence.
  it('respects sub-hour intervals', () => {
    const sub = { last_update: ago(2 * 60 * 1000), auto_update: true, update_interval: '5m' }
    expect(subscriptionHealth(sub, now)).toBe('ok')
    expect(
      subscriptionHealth({ ...sub, last_update: ago(6 * 60 * 1000) }, now),
    ).toBe('outdated')
  })

  it('never flags a subscription that does not auto-update', () => {
    expect(
      subscriptionHealth(
        { last_update: ago(400 * HOUR), auto_update: false, update_interval: '24h' },
        now,
      ),
    ).toBe('ok')
  })

  it('treats an unparseable timestamp as not-stale rather than flagging it', () => {
    expect(
      subscriptionHealth(
        { last_update: 'not a date', auto_update: true, update_interval: '24h' },
        now,
      ),
    ).toBe('ok')
  })
})
