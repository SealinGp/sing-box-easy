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

  test('plan extras use one ellipsized line with a full hover and focus tooltip', async () => {
    const overview = await sourceOf('./SubscriptionsOverviewCard.vue')
    expect(overview).toMatch(/row\.plan\.extras\.length[\s\S]*?class="[^"]*truncate[^"]*"/)
    expect(overview).toContain('@mouseenter="showExtrasTooltip($event, row.id)"')
    expect(overview).toContain('@focusin="showExtrasTooltip($event, row.id)"')
    expect(overview).toMatch(
      /<Teleport to="body">[\s\S]*?extrasTooltipRow\.plan\.extras[\s\S]*?role="tooltip"/,
    )
  })
})
