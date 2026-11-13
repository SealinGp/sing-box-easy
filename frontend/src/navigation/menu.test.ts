import { describe, expect, test } from 'bun:test'
import { createMenu, isMenuActive, searchMenu } from './menu'

describe('navigation discovery', () => {
  test('user management remains discoverable independently of login mode', () => {
    expect(searchMenu(createMenu(key => key), 'users')[0]?.path).toBe('/dashboard/users')
  })
  test('a subscription deep link does not also select the node list', () => {
    expect(isMenuActive('/dashboard/outbounds/list', '/dashboard/outbounds/subscriptions')).toBe(false)
    expect(isMenuActive('/dashboard/dns', '/dashboard/dns/diagnostics')).toBe(true)
    expect(isMenuActive('/dashboard/route', '/dashboard/route-other')).toBe(false)
  })
  test('search supports aliases and multiple terms across group and page', () => {
    const groups = createMenu(key => key)
    expect(searchMenu(groups, 'connections feeds')[0]?.path).toBe('/dashboard/outbounds/subscriptions')
    expect(searchMenu(groups, '  订阅  ')[0]?.path).toBe('/dashboard/outbounds/subscriptions')
    expect(searchMenu(groups, 'no-such-page')).toHaveLength(0)
  })
})
