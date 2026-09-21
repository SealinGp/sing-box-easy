import { describe, expect, it } from 'bun:test'
import { groupRuleSetIssues } from './ruleSetIssues'
import type { RuleSetStatus } from '../types/ruleSet'

const set = (tag: string, reason?: string, detail?: string): RuleSetStatus =>
  ({ tag, state: 'unevaluated', reason, detail }) as RuleSetStatus

describe('groupRuleSetIssues', () => {
  it('collapses one shared cause into a single entry', () => {
    // The case that prompted this: fourteen remote sets, one missing cache
    // file, fourteen identical lines of UI for one problem.
    const missing = 'stat /etc/sing-box/cache.db: no such file or directory'
    const issues = groupRuleSetIssues([
      set('geosite-google', 'cache_unavailable', missing),
      set('geosite-cn', 'cache_unavailable', missing),
      set('sea-rulesets-ai', 'cache_unavailable', missing),
    ])

    expect(issues).toHaveLength(1)
    expect(issues[0]!.tags).toEqual(['geosite-google', 'geosite-cn', 'sea-rulesets-ai'])
    expect(issues[0]!.detail).toBe(missing)
  })

  it('keeps different details apart even when the reason matches', () => {
    // Same reason, different paths. Merging would attribute both to whichever
    // detail arrived first, and the detail is the actionable part.
    const issues = groupRuleSetIssues([
      set('a', 'file_missing', 'open /etc/a.srs: no such file'),
      set('b', 'file_missing', 'open /etc/b.srs: no such file'),
    ])

    expect(issues).toHaveLength(2)
    expect(issues.map((issue) => issue.detail)).toEqual([
      'open /etc/a.srs: no such file',
      'open /etc/b.srs: no such file',
    ])
  })

  it('separates distinct reasons', () => {
    const issues = groupRuleSetIssues([
      set('a', 'cache_unavailable', 'x'),
      set('b', 'unknown_tag'),
      set('c', 'cache_unavailable', 'x'),
    ])

    expect(issues).toHaveLength(2)
    expect(issues[0]!.tags).toEqual(['a', 'c'])
    expect(issues[1]!.reason).toBe('unknown_tag')
  })

  it('ignores healthy sets', () => {
    // A set that loaded has no reason and is not a problem to report.
    expect(groupRuleSetIssues([set('ok'), set('also-ok', '')])).toEqual([])
  })

  it('lists a repeated tag once', () => {
    // The same set is reached from many rules, so the caller may hand us
    // duplicates; the UI must not print the tag three times.
    const issues = groupRuleSetIssues([
      set('geosite-cn', 'not_cached'),
      set('geosite-cn', 'not_cached'),
    ])
    expect(issues[0]!.tags).toEqual(['geosite-cn'])
  })

  it('handles an empty list', () => {
    expect(groupRuleSetIssues([])).toEqual([])
  })
})
