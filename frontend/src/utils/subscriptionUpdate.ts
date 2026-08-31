import type { SubscriptionUpdateResult } from '../types/api'

/**
 * A `vue-i18n` translator, narrowed to what this module needs. Passing it in
 * keeps the summary a pure function — usable from a composable, a component, or
 * a test, without an active i18n instance.
 */
export type Translate = (key: string, named?: Record<string, unknown>) => string

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
