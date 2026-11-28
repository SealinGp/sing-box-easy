import { expect, test } from 'bun:test'
import { nextTab, restoreTabs } from './workspaceTabs'
test('restore only allowed unique destinations and metadata, never draft contents', () => {
  const raw = JSON.stringify([{ path: '/dashboard/dns/rules', pinned: true, password: 'private' }, { path: '/dashboard/dns/rules' }, { path: 'https://evil.test' }, null])
  expect(restoreTabs(raw, p => p.startsWith('/dashboard/'))).toEqual([{ path: '/dashboard/dns/rules', pinned: true }])
  expect(restoreTabs('invalid', () => true)).toEqual([])
  expect(restoreTabs('{}', () => true)).toEqual([])
})
test('closing keeps current page or finds nearest surviving neighbor', () => {
  const tabs = ['a', 'b', 'c', 'd'].map(path => ({ path, pinned: false }))
  expect(nextTab(tabs, new Set(['b']), 'c')).toBe('c')
  expect(nextTab(tabs, new Set(['b', 'c']), 'c')).toBe('a')
  expect(nextTab(tabs, new Set(['a', 'b']), 'a')).toBe('c')
  expect(nextTab(tabs, new Set(['a', 'b', 'c', 'd']), 'c')).toBe('/dashboard/overview')
})
