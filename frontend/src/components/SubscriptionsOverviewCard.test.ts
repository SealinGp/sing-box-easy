import { describe, expect, test } from 'bun:test'

async function componentSource(): Promise<string> {
  return Bun.file(new URL('./SubscriptionsOverviewCard.vue', import.meta.url)).text()
}

describe('subscription overview quality action', () => {
  test('the probe summary itself opens the quality trend', async () => {
    expect(await componentSource()).toMatch(
      /<button\s+v-if="row\.probe"[\s\S]*?@click="openQuality\(row\.id\)"/,
    )
  })

  test('there is no separate chart icon action', async () => {
    expect(await componentSource()).not.toContain('ChartBarIcon')
  })
})
