import { expect, test } from 'bun:test'
import { createDNSResponsePrefix } from './dnsResponseTag'
import { buildTrafficPolicyConfig } from './trafficPolicyConfig'

test('short response tags retain exact evaluate/respond references', () => {
  const prefix = createDNSResponsePrefix()
  expect(prefix).toMatch(/^dns_[A-Za-z0-9_-]{10}$/)
  const config = buildTrafficPolicyConfig({conditions:[{kind:'domain_suffix',value:'example.com'}],mode:'or',outbound:'proxy',servers:['a','b'],timeout:3,fallback:'general',tagPrefix:prefix})
  expect(config.dns!.rules[0]!.tag).toBe(`${prefix}_1`)
  expect(config.dns!.rules[1]!.tag).toBe(`${prefix}_2`)
  expect(config.dns!.rules[2]!.match_response).toBe(config.dns!.rules[0]!.tag)
  expect(config.dns!.rules[3]!.match_response).toBe(config.dns!.rules[1]!.tag)
})
