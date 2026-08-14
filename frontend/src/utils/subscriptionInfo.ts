import type { Subscription } from '../types/api'

/** One provider-supplied metadata entry, as stored on the subscription. */
export type SubInfoEntry = { key: string; value: string }

/**
 * What the UI can say about a subscription's plan.
 *
 * Labels are always the provider's own strings, shown verbatim. The derived
 * numbers (`usedRatio`, `daysUntilExpiry`) are best-effort and null whenever
 * they could not be established — a caller must degrade to the labels alone
 * rather than inventing a value.
 */
export interface PlanSummary {
  usedLabel: string | null
  totalLabel: string | null
  remainingLabel: string | null
  expiresLabel: string | null
  /** 0–1, only when both Used and Total parsed. Null otherwise. */
  usedRatio: number | null
  /** Negative once expired. Null when Expires is absent or unparseable. */
  daysUntilExpiry: number | null
  /** Entries we deliberately do not interpret, preserved in feed order. */
  extras: SubInfoEntry[]
  /** True when there is anything at all worth rendering. */
  hasAny: boolean
}

/**
 * The four keys the backend derives from the `subscription-userinfo` header.
 *
 * These are a fixed spec rather than provider labels, which is precisely why
 * they are safe to interpret. Everything else in `info` comes from in-feed
 * "info nodes" whose keys are provider-defined and localized (剩余流量,
 * 套餐到期, …). Guessing at those would mean inferring semantics from
 * arbitrary text in any language, so they are passed through as `extras` and
 * displayed exactly as the provider wrote them.
 */
const CANONICAL_KEYS = ['used', 'total', 'remaining', 'expires'] as const

/** Binary units, matching the backend's humanizeBytes (1024-based). */
const UNIT_MULTIPLIERS: Record<string, number> = {
  B: 1,
  K: 1024,
  M: 1024 ** 2,
  G: 1024 ** 3,
  T: 1024 ** 4,
  P: 1024 ** 5,
  E: 1024 ** 6,
}

// e.g. "750.00 GB", "4.59 TB", "512 B", and the "GiB" spelling some providers
// use. The comma alternative covers locales that group with it.
const SIZE_PATTERN = /^\s*([\d]+(?:[.,][\d]+)?)\s*([KMGTPE])?i?B\s*$/i

/**
 * Parses a humanized byte string back into a number.
 *
 * This round-trips the backend's own formatting, so it is intentionally strict:
 * anything that does not match the expected shape returns null and the caller
 * falls back to showing the label without a derived percentage.
 */
export const parseByteSize = (value: string): number | null => {
  const match = SIZE_PATTERN.exec(value)
  if (!match) return null

  const rawAmount = match[1]
  if (!rawAmount) return null

  const amount = Number.parseFloat(rawAmount.replace(',', '.'))
  if (!Number.isFinite(amount)) return null

  const unit = (match[2] ?? 'B').toUpperCase()
  const multiplier = UNIT_MULTIPLIERS[unit]
  if (multiplier === undefined) return null

  return amount * multiplier
}

/** Whole days from now until `date`; negative once it has passed. */
const daysUntil = (date: Date, now: Date): number => {
  const msPerDay = 24 * 60 * 60 * 1000
  // Compare calendar days, so an expiry later today reads as 0 rather than -1.
  const start = Date.UTC(now.getFullYear(), now.getMonth(), now.getDate())
  const end = Date.UTC(date.getFullYear(), date.getMonth(), date.getDate())
  return Math.round((end - start) / msPerDay)
}

/**
 * Parses the backend's `Expires` value, which is formatted as YYYY-MM-DD.
 *
 * Parsed as local time rather than via `new Date("2026-10-19")`, which the
 * spec treats as UTC midnight — that shifts the date backwards for anyone west
 * of Greenwich and would show a plan expiring "yesterday".
 */
const parseExpiryDate = (value: string): Date | null => {
  const match = /^\s*(\d{4})-(\d{2})-(\d{2})\s*$/.exec(value)
  if (!match) return null

  const [, year, month, day] = match
  if (!year || !month || !day) return null

  const date = new Date(Number(year), Number(month) - 1, Number(day))
  return Number.isNaN(date.getTime()) ? null : date
}

/**
 * Reduces a subscription's raw info entries into what the UI renders.
 *
 * `now` is injectable so the "days until expiry" arithmetic can be exercised
 * deterministically.
 */
export const summarizePlan = (
  subscription: Pick<Subscription, 'info'>,
  now: Date = new Date(),
): PlanSummary => {
  const entries = subscription.info ?? []

  const canonical = new Map<string, string>()
  const extras: SubInfoEntry[] = []

  for (const entry of entries) {
    const key = entry?.key?.trim() ?? ''
    const value = entry?.value?.trim() ?? ''
    const lowered = key.toLowerCase()

    if ((CANONICAL_KEYS as readonly string[]).includes(lowered) && value) {
      canonical.set(lowered, value)
    } else if (key || value) {
      extras.push({ key, value })
    }
  }

  const usedLabel = canonical.get('used') ?? null
  const totalLabel = canonical.get('total') ?? null
  const remainingLabel = canonical.get('remaining') ?? null
  const expiresLabel = canonical.get('expires') ?? null

  let usedRatio: number | null = null
  if (usedLabel && totalLabel) {
    const used = parseByteSize(usedLabel)
    const total = parseByteSize(totalLabel)
    // A zero total means "unlimited" upstream, so a ratio would be meaningless
    // (and a division by zero). The labels still render.
    if (used !== null && total !== null && total > 0) {
      usedRatio = Math.min(Math.max(used / total, 0), 1)
    }
  }

  let daysUntilExpiry: number | null = null
  if (expiresLabel) {
    const expiry = parseExpiryDate(expiresLabel)
    if (expiry) daysUntilExpiry = daysUntil(expiry, now)
  }

  return {
    usedLabel,
    totalLabel,
    remainingLabel,
    expiresLabel,
    usedRatio,
    daysUntilExpiry,
    extras,
    hasAny: canonical.size > 0 || extras.length > 0,
  }
}
