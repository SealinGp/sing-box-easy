import { describe, expect, test } from 'bun:test'
import { createMenu, isMenuActive, searchMenu } from './menu'

describe('navigation discovery', () => {
  test('account routes are absent without authentication', () => {
    expect(searchMenu(createMenu(key => key, false), 'users')).toHaveLength(0)
    expect(searchMenu(createMenu(key => key, true), 'users')).toHaveLength(1)
  })
  test('a subscription deep link does not also select the node list', () => {
    expect(isMenuActive('/dashboard/outbounds/list', '/dashboard/outbounds/subscriptions')).toBe(false)
    expect(isMenuActive('/dashboard/dns', '/dashboard/dns/diagnostics')).toBe(true)
    expect(isMenuActive('/dashboard/route', '/dashboard/route-other')).toBe(false)
  })
  test('search supports aliases and multiple terms across group and page', () => {
    const groups = createMenu(key => key, false)
    expect(searchMenu(groups, 'connections feeds')[0]?.path).toBe('/dashboard/outbounds/subscriptions')
    expect(searchMenu(groups, '  订阅  ')[0]?.path).toBe('/dashboard/outbounds/subscriptions')
    expect(searchMenu(groups, 'no-such-page')).toHaveLength(0)
  })
})
