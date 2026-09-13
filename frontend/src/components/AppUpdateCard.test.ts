import { describe, expect, test } from 'bun:test'

describe('AppUpdateCard clipboard behavior', () => {
  test('uses the shared clipboard fallback for update commands', async () => {
    const source = await Bun.file(new URL('./AppUpdateCard.vue', import.meta.url)).text()

    expect(source).toContain("import { writeTextToClipboard } from '../utils/clipboard'")
    expect(source).toContain('await writeTextToClipboard(command)')
    expect(source).not.toContain('navigator.clipboard.writeText(command)')
  })
})
