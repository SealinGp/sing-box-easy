// Locale-aware relative time formatting (e.g. "14 minutes ago" / "14分钟前")
// built on the platform Intl.RelativeTimeFormat so we don't ship a date lib.

interface Division {
  amount: number
  unit: Intl.RelativeTimeFormatUnit
}

const DIVISIONS: Division[] = [
  { amount: 60, unit: 'second' },
  { amount: 60, unit: 'minute' },
  { amount: 24, unit: 'hour' },
  { amount: 7, unit: 'day' },
  { amount: 4.34524, unit: 'week' },
  { amount: 12, unit: 'month' },
  { amount: Number.POSITIVE_INFINITY, unit: 'year' },
]

/**
 * Format a past/future instant relative to now.
 *
 * @param fromMs  the instant in epoch milliseconds
 * @param locale  BCP-47 locale (e.g. "en", "zh"); falls back to "en"
 * @param nowMs   reference "now" in epoch ms (injectable for testing)
 * @returns e.g. "14 minutes ago"; negative durations are past.
 */
export function formatRelativeTime(fromMs: number, locale = 'en', nowMs = Date.now()): string {
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' })
  let duration = (fromMs - nowMs) / 1000 // seconds; negative => in the past
  for (const division of DIVISIONS) {
    if (Math.abs(duration) < division.amount) {
      return rtf.format(Math.round(duration), division.unit)
    }
    duration /= division.amount
  }
  return rtf.format(Math.round(duration), 'year')
}
