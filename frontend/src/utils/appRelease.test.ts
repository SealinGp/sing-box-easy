import { expect, test } from 'bun:test'
import { isDifferentRelease } from './appRelease'

test('the current release cannot be offered for reinstall', () => {
  expect(isDifferentRelease('v1.2.3', 'v1.2.3')).toBe(false)
  expect(isDifferentRelease('1.2.3', ' v1.2.3 ')).toBe(false)
  expect(isDifferentRelease('1.2.3', '')).toBe(false)
})
test('other releases and unstamped builds remain installable', () => {
  expect(isDifferentRelease('1.2.3', '1.2.4')).toBe(true)
  expect(isDifferentRelease('1.2.3', '1.2.2')).toBe(true)
  expect(isDifferentRelease('1.2.3-rc.1', '1.2.3')).toBe(true)
  expect(isDifferentRelease('dev', '1.2.3')).toBe(true)
})
