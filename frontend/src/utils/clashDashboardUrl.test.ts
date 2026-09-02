import { describe, expect, it } from 'bun:test'
import {
  clashDashboardHref,
  isLinkBlockedBySecret,
  parseListenAddress,
  resolveControllerEndpoint,
} from './clashDashboardUrl'

const HOST = 'panel.lan'

describe('parseListenAddress', () => {
  it('splits host:port', () => {
    expect(parseListenAddress('127.0.0.1:9090')).toEqual({ host: '127.0.0.1', port: '9090' })
  })

  it('keeps the brackets on an IPv6 literal', () => {
    expect(parseListenAddress('[::1]:9090')).toEqual({ host: '[::1]', port: '9090' })
  })

  it('reads a port-only bind as an empty host', () => {
    expect(parseListenAddress(':9090')).toEqual({ host: '', port: '9090' })
  })

  it('reports no port when there is no colon', () => {
    expect(parseListenAddress('9090')).toEqual({ host: '9090', port: '' })
  })
})

describe('resolveControllerEndpoint', () => {
  it('substitutes the page host for a wildcard bind', () => {
    // 0.0.0.0 is reachable for sing-box and meaningless to the browser.
    expect(resolveControllerEndpoint('0.0.0.0:9090', HOST)).toEqual({ host: HOST, port: '9090' })
    expect(resolveControllerEndpoint('[::]:9090', HOST)).toEqual({ host: HOST, port: '9090' })
    expect(resolveControllerEndpoint(':9090', HOST)).toEqual({ host: HOST, port: '9090' })
  })

  it('falls back to loopback when the page has no hostname', () => {
    expect(resolveControllerEndpoint('0.0.0.0:9090', '')).toEqual({ host: '127.0.0.1', port: '9090' })
  })

  it('leaves a real host alone', () => {
    expect(resolveControllerEndpoint('10.0.0.2:9090', HOST)).toEqual({ host: '10.0.0.2', port: '9090' })
  })

  it('is null when unset or portless', () => {
    expect(resolveControllerEndpoint(undefined, HOST)).toBeNull()
    expect(resolveControllerEndpoint('   ', HOST)).toBeNull()
    expect(resolveControllerEndpoint('127.0.0.1', HOST)).toBeNull()
  })
})

describe('clashDashboardHref', () => {
  it('links to the bare root when there is no dashboard and no secret', () => {
    expect(clashDashboardHref({ external_controller: '127.0.0.1:9090' }, HOST)).toBe(
      'http://127.0.0.1:9090',
    )
  })

  it('produces NO link when a secret guards a dashboard-less controller', () => {
    // Every URL we could emit would answer 401: sing-box requires an
    // Authorization header that an <a href> cannot send.
    expect(
      clashDashboardHref({ external_controller: '127.0.0.1:9090', secret: 'hunter2' }, HOST),
    ).toBe('')
    expect(
      isLinkBlockedBySecret({ external_controller: '127.0.0.1:9090', secret: 'hunter2' }, HOST),
    ).toBe(true)
  })

  it('targets /ui/ and seeds both dashboard conventions', () => {
    const href = clashDashboardHref(
      { external_controller: '127.0.0.1:9090', external_ui: 'ui', secret: 'hunter2' },
      HOST,
    )
    // yacd reads the query string, zashboard reads the hash — hence both.
    expect(href).toBe(
      'http://127.0.0.1:9090/ui/?hostname=127.0.0.1&port=9090&secret=hunter2' +
        '#/setup?hostname=127.0.0.1&port=9090&secret=hunter2',
    )
  })

  it('encodes a space as %20, not +, so the hash half survives', () => {
    // decodeURIComponent leaves `+` as a literal plus, which would corrupt the
    // secret the dashboard replays.
    const href = clashDashboardHref(
      { external_controller: '127.0.0.1:9090', external_ui: 'ui', secret: 'two words' },
      HOST,
    )
    expect(href).toContain('secret=two%20words')
    expect(href).not.toContain('+')
  })

  it('omits the secret parameter when there is none', () => {
    const href = clashDashboardHref(
      { external_controller: '127.0.0.1:9090', external_ui: 'ui' },
      HOST,
    )
    expect(href).not.toContain('secret')
    expect(isLinkBlockedBySecret({ external_controller: '127.0.0.1:9090', external_ui: 'ui' }, HOST)).toBe(
      false,
    )
  })

  it('is empty when the controller is not configured', () => {
    expect(clashDashboardHref({}, HOST)).toBe('')
    expect(isLinkBlockedBySecret({}, HOST)).toBe(false)
  })
})
