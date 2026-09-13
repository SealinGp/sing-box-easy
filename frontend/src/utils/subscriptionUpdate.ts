import type { SubscriptionUpdateResult } from '../types/api'

/**
 * A `vue-i18n` translator, narrowed to what this module needs. Passing it in
 * keeps the summary a pure function — usable from a composable, a component, or
 * a test, without an active i18n instance.
 */
export type Translate = (key: string, named?: Record<string, unknown>) => string

/**
 * A successful manual refresh restarts sing-box before it resolves. On routers
 * whose system resolver points at sing-box, the process can exist just before
 * its DNS listener is ready; starting the next provider fetch in that window
 * fails with a connection-refused lookup. Keep the established one-second
 * recovery window shared by every "update all" entry point.
 */
export const SUBSCRIPTION_UPDATE_COOLDOWN_MS = 1000

const wait = (milliseconds: number) =>
  new Promise<void>((resolve) => setTimeout(resolve, milliseconds))

/**
 * Updates subscriptions one at a time with a service-recovery window between
 * them. The injected pause keeps the ordering contract deterministic in tests.
 * Callers own per-subscription error handling so one failed provider can still
 * allow the remaining subscriptions to run.
 */
export async function updateSubscriptionsSequentially<T>(
  subscriptions: readonly T[],
  update: (subscription: T) => Promise<void>,
  pause: (milliseconds: number) => Promise<void> = wait,
): Promise<void> {
  for (const [index, subscription] of subscriptions.entries()) {
    await update(subscription)
    if (index < subscriptions.length - 1) {
      await pause(SUBSCRIPTION_UPDATE_COOLDOWN_MS)
    }
  }
}

/**
 * Renders a refresh's 3-way diff as a short human line: "+3 added, ~1 updated".
 *
 * Shared by the Subscriptions page and the Overview card so the two cannot
 * drift into describing the same result differently — the counts come from one
 * backend response and should read the same wherever they land.
 *
 * "No changes" is a real, common outcome (a provider that has not touched its
 * node list), and saying so explicitly is the point: an empty string would read
 * as "nothing happened", which is exactly what a failed refresh looks like.
 */
export function summarizeUpdate(
  result: Pick<SubscriptionUpdateResult, 'added' | 'updated' | 'deleted'>,
  t: Translate,
): string {
  const parts: string[] = []
  if (result.added > 0) parts.push(t('subscriptions.notify.added', { n: result.added }))
  if (result.updated > 0) parts.push(t('subscriptions.notify.updated', { n: result.updated }))
  if (result.deleted > 0) parts.push(t('subscriptions.notify.removed', { n: result.deleted }))
  return parts.length > 0 ? parts.join(', ') : t('subscriptions.notify.noChanges')
}
