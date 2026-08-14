import type { Subscription } from '../types/api'
import { parseDurationToHours } from '../plugins/dayjs'

/**
 * How current a subscription's node list is.
 *
 * - `never`    — it has never been fetched, so its nodes do not exist yet.
 * - `outdated` — auto-update is on and the interval has elapsed.
 * - `ok`       — fetched, and not overdue.
 */
export type SubscriptionHealth = 'never' | 'outdated' | 'ok'

/** Fallback cadence when the stored interval is missing or unparseable. */
const DEFAULT_INTERVAL_HOURS = 24

/**
 * Classifies a subscription's freshness.
 *
 * "Outdated" only has meaning for auto-updating subscriptions: it mirrors the
 * backend's own refresh rule (AutoUpdater.shouldUpdate), which considers a
 * subscription due once time-since-last-update >= its update_interval. With
 * auto-update off there is no expected cadence, so it is never flagged.
 *
 * Shared by the Subscriptions page and the Overview card so the two cannot
 * drift into disagreeing about the same subscription.
 */
export const subscriptionHealth = (
  subscription: Pick<Subscription, 'last_update' | 'auto_update' | 'update_interval'>,
  now: number = Date.now(),
): SubscriptionHealth => {
  if (!subscription.last_update) return 'never'

  if (subscription.auto_update ?? false) {
    const lastUpdate = new Date(subscription.last_update).getTime()
    // An unparseable timestamp is not evidence of staleness; treat it as ok
    // rather than flagging every row when a provider sends an odd format.
    if (!Number.isNaN(lastUpdate)) {
      const hoursSinceUpdate = (now - lastUpdate) / (1000 * 60 * 60)
      const intervalHours = parseDurationToHours(subscription.update_interval) ?? DEFAULT_INTERVAL_HOURS
      if (hoursSinceUpdate >= intervalHours) return 'outdated'
    }
  }

  return 'ok'
}
