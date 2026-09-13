import { describe, expect, test } from 'bun:test'
import { formatPlanExtras } from './subscriptionInfo'

describe('formatPlanExtras', () => {
  test('keeps every provider entry in feed order', () => {
    expect(
      formatPlanExtras([
        { key: '套餐', value: 'Premium' },
        { key: '到期', value: '2026-10-01' },
        { key: '公告', value: '' },
      ]),
    ).toBe('套餐: Premium · 到期: 2026-10-01 · 公告')
  })

  test('replaces provider line breaks with ellipses', () => {
    expect(
      formatPlanExtras([
        { key: 'Notice\nEN', value: 'Line one\r\nLine two' },
      ]),
    ).toBe('Notice … EN: Line one … Line two')
  })
})
