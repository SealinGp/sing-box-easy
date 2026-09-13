import { describe, expect, test } from 'bun:test'

async function templateOf(filename: string): Promise<string> {
  const source = await Bun.file(new URL(filename, import.meta.url)).text()
  const template = source.match(/<template>([\s\S]*?)<\/template>/)?.[1]
  if (!template) throw new Error(`Missing template in ${filename}`)
  return template
}

describe('compact dashboard pages', () => {
  test('the shared Outbounds shell adds no title above its three child pages', async () => {
    expect(await templateOf('./Outbounds.vue')).not.toMatch(/<h1\b/)
  })

  test('the Inbounds page starts with its controls instead of a page title', async () => {
    expect(await templateOf('./Inbounds.vue')).not.toContain("$t('inbounds.title')")
  })

  test('the Settings page starts directly with its card grid', async () => {
    expect(await templateOf('./Settings.vue')).not.toContain("$t('settings.title')")
  })
})
