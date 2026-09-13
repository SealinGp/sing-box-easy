import { describe, expect, test } from 'bun:test'

import { writeTextToClipboard } from './clipboard'

describe('writeTextToClipboard', () => {
  test('uses the Clipboard API when it succeeds', async () => {
    const calls: string[] = []

    await writeTextToClipboard('opkg install sing-box-easy', {
      modern: async (text) => {
        calls.push(`modern:${text}`)
      },
      legacy: (text) => {
        calls.push(`legacy:${text}`)
        return true
      },
    })

    expect(calls).toEqual(['modern:opkg install sing-box-easy'])
  })

  test('uses the legacy fallback when the Clipboard API is unavailable', async () => {
    const calls: string[] = []

    await writeTextToClipboard('opkg install sing-box-easy', {
      legacy: (text) => {
        calls.push(text)
        return true
      },
    })

    expect(calls).toEqual(['opkg install sing-box-easy'])
  })

  test('uses the legacy fallback when the Clipboard API rejects', async () => {
    const calls: string[] = []

    await writeTextToClipboard('opkg install sing-box-easy', {
      modern: async () => {
        throw new Error('Clipboard access denied')
      },
      legacy: (text) => {
        calls.push(text)
        return true
      },
    })

    expect(calls).toEqual(['opkg install sing-box-easy'])
  })

  test('fails only when both clipboard methods fail', async () => {
    let error: unknown

    try {
      await writeTextToClipboard('opkg install sing-box-easy', {
        modern: async () => {
          throw new Error('Clipboard access denied')
        },
        legacy: () => false,
      })
    } catch (cause) {
      error = cause
    }

    expect(error).toBeInstanceOf(Error)
  })
})
