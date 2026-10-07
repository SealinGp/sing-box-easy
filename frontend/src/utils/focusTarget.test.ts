import { describe, expect, test } from 'bun:test'
import { FOCUS_PAGES, focusFromQuery, focusLink, splitRuleByRuleSets } from './focusTarget'

describe('focusLink', () => {
  test('carries the name as a query value, untouched', () => {
    // Tags are free text: spaces, emoji, pipes. The router encodes the query,
    // so the name must go in raw — pre-encoding it would double-encode.
    expect(focusLink(FOCUS_PAGES.proxies, '🤖 AI | x/y')).toEqual({
      path: '/dashboard/proxies',
      query: { focus: '🤖 AI | x/y' },
    })
  })
})

describe('focusFromQuery', () => {
  test('reads a plain string', () => {
    expect(focusFromQuery('Media')).toBe('Media')
  })
  test('takes the first of a repeated parameter', () => {
    expect(focusFromQuery(['Media', 'AI'])).toBe('Media')
  })
  test('is empty for anything else', () => {
    for (const value of [undefined, null, '', [], [null]]) {
      expect(focusFromQuery(value)).toBe('')
    }
  })
})

describe('splitRuleByRuleSets', () => {
  const known = new Set(['geosite-cn', 'geoip-cn', 'geosite-geolocation-!cn', 'geosite-google'])
  const linked = (rule: string) =>
    splitRuleByRuleSets(rule, known).filter((part) => part.ruleSet).map((part) => part.ruleSet)
  const rejoin = (rule: string) => splitRuleByRuleSets(rule, known).map((part) => part.text).join('')

  test('links each known set in a bracketed list', () => {
    // The shape sing-box 1.14 actually reports for a multi-set rule.
    const rule = 'rule_set=[geosite-cn geoip-cn sea-rulesets-direct] => route(➡️ 直连)'
    expect(linked(rule)).toEqual(['geosite-cn', 'geoip-cn'])
  })

  test('a set the config does not have stays plain text', () => {
    // A link that lands on a page with nothing highlighted is worse than none.
    expect(linked('rule_set=[sea-rulesets-direct] => route(x)')).toEqual([])
  })

  test('links a single unbracketed set', () => {
    expect(linked('rule_set=geosite-google => route(Google)')).toEqual(['geosite-google'])
  })

  test('keeps a tag with punctuation whole', () => {
    expect(linked('rule_set=[geosite-geolocation-!cn] => route(x)')).toEqual(['geosite-geolocation-!cn'])
  })

  test('never drops or reorders a character of the rule', () => {
    for (const rule of [
      'rule_set=[geosite-cn geoip-cn sea-rulesets-direct] => route(➡️ 直连)',
      'rule_set=geosite-google => route(Google)',
      'ip_version=6 => route(➡️ 直连)',
      'final',
      '',
    ]) {
      expect(rejoin(rule)).toBe(rule)
    }
  })

  test('a known tag outside a rule_set clause is not a link', () => {
    // "geosite-cn" as an outbound name is not a rule set.
    expect(linked('domain=[a.com] => route(geosite-cn)')).toEqual([])
  })

  test('a rule with no rule set is one plain part', () => {
    expect(splitRuleByRuleSets('final', known)).toEqual([{ text: 'final' }])
  })

  test('handles two rule_set clauses', () => {
    expect(linked('rule_set=[geosite-cn] rule_set=[geoip-cn] => route(x)')).toEqual(['geosite-cn', 'geoip-cn'])
  })
})
