import { describe, expect, it } from 'bun:test'
import { summarizeUpdate, updateSubscriptionsSequentially } from './subscriptionUpdate'

// Stands in for vue-i18n: echoes the key with its count so assertions read as
// the shape of the output rather than as translated prose.
const t = (key: string, named?: Record<string, unknown>) =>
  named && 'n' in named ? `${key}:${named.n}` : key

describe('summarizeUpdate', () => {
  it('lists only the non-zero parts, in add/update/remove order', () => {
    expect(summarizeUpdate({ added: 3, updated: 1, deleted: 2 }, t)).toBe(
      'subscriptions.notify.added:3, subscriptions.notify.updated:1, subscriptions.notify.removed:2',
    )
    expect(summarizeUpdate({ added: 3, updated: 0, deleted: 0 }, t)).toBe(
      'subscriptions.notify.added:3',
    )
    expect(summarizeUpdate({ added: 0, updated: 0, deleted: 5 }, t)).toBe(
      'subscriptions.notify.removed:5',
    )
  })

  it('says so when nothing changed', () => {
    // Never an empty string: "nothing to report" and "the refresh failed" must
    // not look the same to the reader.
    expect(summarizeUpdate({ added: 0, updated: 0, deleted: 0 }, t)).toBe(
      'subscriptions.notify.noChanges',
    )
  })
})

describe('updateSubscriptionsSequentially', () => {
  it('waits for the service cooldown between updates, but not after the last one', async () => {
    const events: string[] = []

    await updateSubscriptionsSequentially(
      ['first', 'second', 'third'],
      async (id) => {
        events.push(`update:start:${id}`)
        await Promise.resolve()
        events.push(`update:end:${id}`)
      },
      async (milliseconds) => {
        events.push(`wait:${milliseconds}`)
      },
    )

    expect(events).toEqual([
      'update:start:first',
      'update:end:first',
      'wait:1000',
      'update:start:second',
      'update:end:second',
      'wait:1000',
      'update:start:third',
      'update:end:third',
    ])
  })
})
