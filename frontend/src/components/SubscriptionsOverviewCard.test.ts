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

  test('the quality cell offers a compact original-style probe alongside its default size', async () => {
    const qualityCell = await sourceOf('./SubscriptionQualityCell.vue')
    expect(qualityCell).toContain("size?: 'small' | 'default'")
    expect(qualityCell).toContain("size: 'default'")
    expect(qualityCell).toContain(`v-if="size === 'small'"`)
    expect(qualityCell).toContain('size="xs"')
    expect(qualityCell).toContain('size="sm"')
    expect(qualityCell).toContain("$t('subProbe.column')")
  })

  test('the overview selects the compact quality-cell size', async () => {
    expect(await sourceOf('./SubscriptionsOverviewCard.vue')).toMatch(
      /<SubscriptionQualityCell\s+v-if="row\.probe"[\s\S]*?size="small"/,
    )
  })

})

describe('shared subscription quota details', () => {
  test('the quota module owns the progress bar and its accessible tooltip', async () => {
    const quota = await sourceOf('./SubscriptionQuotaDetails.vue')
    expect(quota).toContain('role="progressbar"')
    expect(quota).toContain('@mouseenter="showTooltip($event, \'quota\')"')
    expect(quota).toContain('@focusin="showTooltip($event, \'quota\')"')
    expect(quota).toMatch(/<Teleport to="body">[\s\S]*?role="tooltip"/)
  })

  test('the quota module owns fallback, expiry, and ellipsized plan extras', async () => {
    const quota = await sourceOf('./SubscriptionQuotaDetails.vue')
    expect(quota).toContain('showExpiry?: boolean')
    expect(quota).toMatch(/plan\.extras\.length[\s\S]*?class="[^"]*truncate[^"]*"/)
    expect(quota).toContain("formatPlanExtras(plan.extras)")
    expect(quota).toContain("activeTooltip === 'extras'")
    expect(quota).toContain("$t('overview.subscriptions.noPlanInfo')")
  })

  test('the overview delegates quota rendering and hides its duplicate expiry line', async () => {
    expect(await sourceOf('./SubscriptionsOverviewCard.vue')).toMatch(
      /<SubscriptionQuotaDetails[\s\S]*?:plan="row\.plan"[\s\S]*?:subscription-name="row\.name"[\s\S]*?:show-expiry="false"/,
    )
  })

  test('the subscriptions table uses the same quota module', async () => {
    expect(await sourceOf('../views/dashboard/Subscriptions.vue')).toMatch(
      /<SubscriptionQuotaDetails[\s\S]*?:plan="summarizePlan\(subscription\)"[\s\S]*?:subscription-name="subscription\.name"/,
    )
  })

  test('callers no longer own quota bars or plan-detail tooltip state', async () => {
    const overview = await sourceOf('./SubscriptionsOverviewCard.vue')
    const subscriptions = await sourceOf('../views/dashboard/Subscriptions.vue')
    expect(`${overview}\n${subscriptions}`).not.toContain('showQuotaTooltip')
    expect(`${overview}\n${subscriptions}`).not.toContain('quota-fill')
    expect(subscriptions).not.toContain('v-for="(entry, i) in subscription.info"')
  })
})
