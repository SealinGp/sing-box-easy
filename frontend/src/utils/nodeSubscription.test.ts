import { expect, test } from 'bun:test'
import { nodeSubscriptionId } from './nodeSubscription'

test('reads the final ownership suffix and ignores manual nodes or empty IDs', () => {
  expect(nodeSubscriptionId('🇭🇰 香港 | BGP abc12345 | sub_1788274931')).toBe('sub_1788274931')
  expect(nodeSubscriptionId('old | sub_old abc12345 | sub_current')).toBe('sub_current')
  expect(nodeSubscriptionId('manual | node')).toBe('')
  expect(nodeSubscriptionId('manual | sub_')).toBe('')
})
