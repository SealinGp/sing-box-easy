import { describe, expect, test } from 'bun:test'

async function sourceOf(filename: string): Promise<string> {
  return Bun.file(new URL(filename, import.meta.url)).text()
}

describe('shared subscription quality interaction', () => {
  test('the quality cell owns the trend dialog', async () => {
    expect(await sourceOf('./SubscriptionQualityCell.vue')).toMatch(
      /<SubscriptionQualityDialog[\s\S]*?:subscription="subscription"/,
    )
  })

  test('the quality cell owns the action that opens the trend', async () => {
    expect(await sourceOf('./SubscriptionQualityCell.vue')).toContain('@click="openQuality"')
  })

  test('the overview delegates probe rendering to the quality cell', async () => {
    expect(await sourceOf('./SubscriptionsOverviewCard.vue')).toMatch(
      /<SubscriptionQualityCell\s+v-if="row\.probe"[\s\S]*?:subscription="row\.subscription"/,
    )
  })

  test('the subscriptions table passes its subscription through the shared interface', async () => {
    expect(await sourceOf('../views/dashboard/Subscriptions.vue')).toMatch(
      /<SubscriptionQualityCell[\s\S]*?:subscription="subscription"/,
    )
  })

  test('callers no longer own duplicate quality dialogs', async () => {
    const overview = await sourceOf('./SubscriptionsOverviewCard.vue')
    const subscriptions = await sourceOf('../views/dashboard/Subscriptions.vue')
    expect(`${overview}\n${subscriptions}`).not.toContain('SubscriptionQualityDialog')
  })

  test('the shared rendering has no separate chart icon', async () => {
    expect(await sourceOf('./SubscriptionQualityCell.vue')).not.toContain('ChartBarIcon')
  })
})
