import { describe, expect, it } from 'bun:test'
import {
  buildParallelResolveGroup,
  findOrphanRespondRules,
  suggestGroupName,
  validateParallelResolveGroup,
} from './parallelResolve'

const CONDITIONS = { domain_suffix: ['deepseek.com', 'owolist.cn'] }

describe('buildParallelResolveGroup', () => {
  it('emits every evaluate before any respond', () => {
    // Not cosmetic. sing-box requires a `respond` to have a PRECEDING
    // top-level evaluate, and if it is reached without one the request FAILS
    // rather than falling through — so interleaving the pairs would break
    // resolution outright for the domains this group covers.
    const rules = buildParallelResolveGroup({
      conditions: CONDITIONS,
      servers: ['dns_router', 'aliyun', 'tencent'],
      group: 'direct_domain',
    })

    const actions = rules.map((rule) => rule.action)
    expect(actions).toEqual([
      'evaluate', 'evaluate', 'evaluate',
      'respond', 'respond', 'respond',
    ])
  })

  it('matches the hand-written config shape', () => {
    const rules = buildParallelResolveGroup({
      conditions: CONDITIONS,
      servers: ['dns_router', 'aliyun', 'tencent'],
      group: 'direct_domain',
      timeout: '3s',
    })

    expect(rules[0]).toEqual({
      domain_suffix: ['deepseek.com', 'owolist.cn'],
      action: 'evaluate',
      server: 'dns_router',
      tag: 'direct_domain_dns_router',
      timeout: '3s',
    })
    expect(rules[3]).toEqual({
      match_response: 'direct_domain_dns_router',
      response_rcode: 'NOERROR',
      action: 'respond',
      race: true,
    })
  })

  it('pairs each respond with its own evaluate tag, in the same order', () => {
    const rules = buildParallelResolveGroup({
      conditions: CONDITIONS,
      servers: ['a', 'b', 'c'],
      group: 'g',
    })
    const evaluateTags = rules.filter((r) => r.action === 'evaluate').map((r) => r.tag)
    const respondTags = rules.filter((r) => r.action === 'respond').map((r) => r.match_response)
    expect(respondTags).toEqual(evaluateTags)
  })

  it('repeats the conditions on every evaluate', () => {
    // Each evaluate is matched independently; conditions on only the first
    // would send one query and skip the rest.
    const rules = buildParallelResolveGroup({
      conditions: CONDITIONS,
      servers: ['a', 'b'],
      group: 'g',
    })
    for (const rule of rules.filter((r) => r.action === 'evaluate')) {
      expect(rule.domain_suffix).toEqual(['deepseek.com', 'owolist.cn'])
    }
  })

  it('carries no conditions on the respond rules', () => {
    // A respond selects by match_response, not by the query. Repeating the
    // domain conditions there would be inert at best and, combined with race,
    // misleading about what decides the winner.
    const rules = buildParallelResolveGroup({
      conditions: CONDITIONS,
      servers: ['a', 'b'],
      group: 'g',
    })
    for (const rule of rules.filter((r) => r.action === 'respond')) {
      expect(rule.domain_suffix).toBeUndefined()
    }
  })

  it('omits an empty timeout rather than writing one', () => {
    const rules = buildParallelResolveGroup({
      conditions: CONDITIONS,
      servers: ['a', 'b'],
      group: 'g',
      timeout: '   ',
    })
    expect('timeout' in rules[0]!).toBe(false)
  })

  it('drops empty condition lists', () => {
    // A catch-all race is legitimate — the production config has one — but it
    // must be spelled as no conditions, not as empty arrays that sing-box
    // would reject.
    const rules = buildParallelResolveGroup({
      conditions: { domain: [], domain_suffix: ['a.test'], geosite: [] },
      servers: ['a', 'b'],
      group: 'g',
    })
    expect(rules[0]).toEqual({
      domain_suffix: ['a.test'],
      action: 'evaluate',
      server: 'a',
      tag: 'g_a',
    })
  })

  it('sanitises server tags into usable response tags', () => {
    // A server tag may contain spaces or punctuation; the response tag is a
    // key referenced by match_response and must stay stable and readable.
    const rules = buildParallelResolveGroup({
      conditions: CONDITIONS,
      servers: ['Google DoT', '阿里 DNS'],
      group: 'grp',
    })
    const tags = rules.filter((r) => r.action === 'evaluate').map((r) => r.tag)
    expect(tags[0]).toBe('grp_Google_DoT')
    // Non-ASCII is kept: sing-box tags are free-form and the operator chose it.
    expect(tags[1]).toBe('grp_阿里_DNS')
  })

  it('keeps tags unique when two servers sanitise to the same thing', () => {
    // Both collapse to `g_a_b`. Two evaluates sharing a tag would leave one
    // respond rule pointed at the other server's response.
    const rules = buildParallelResolveGroup({
      conditions: CONDITIONS,
      servers: ['a b', 'a/b'],
      group: 'g',
    })
    const tags = rules.filter((r) => r.action === 'evaluate').map((r) => r.tag)
    expect(tags).toEqual(['g_a_b', 'g_a_b_2'])
  })
})

describe('validateParallelResolveGroup', () => {
  const base = { conditions: CONDITIONS, servers: ['a', 'b'], group: 'g' }

  it('accepts a well-formed group', () => {
    expect(validateParallelResolveGroup(base, [])).toBeNull()
  })

  it('requires at least two servers', () => {
    // One server is not a race; it is a plain `route` rule, and generating
    // an evaluate/respond pair for it adds two rules to do one rule's job.
    expect(validateParallelResolveGroup({ ...base, servers: ['a'] }, [])).toBe(
      'dns.rules.parallel.errors.needTwoServers',
    )
  })

  it('requires a group name', () => {
    expect(validateParallelResolveGroup({ ...base, group: '  ' }, [])).toBe(
      'dns.rules.parallel.errors.needGroup',
    )
  })

  it('rejects a group whose tags would collide with existing rules', () => {
    // match_response selects by tag. A duplicate tag silently re-points an
    // existing respond rule at this group's response.
    const existing = [{ action: 'evaluate', tag: 'g_a' }]
    expect(validateParallelResolveGroup(base, existing)).toBe(
      'dns.rules.parallel.errors.tagExists',
    )
  })

  it('allows a catch-all group with no conditions', () => {
    // The production config races three resolvers unconditionally as its
    // final fallback; refusing that would make the generator unable to
    // produce the very pattern it is modelled on.
    expect(validateParallelResolveGroup({ ...base, conditions: {} }, [])).toBeNull()
  })
})

describe('findOrphanRespondRules', () => {
  it('flags a respond whose evaluate is missing', () => {
    // The failure this guards against is not a fall-through: sing-box fails
    // the request outright when a respond is reached with no evaluated
    // response, so an orphan breaks resolution rather than degrading it.
    const rules = [
      { action: 'respond', match_response: 'ghost' },
    ]
    expect(findOrphanRespondRules(rules)).toEqual([0])
  })

  it('flags a respond that sits ABOVE its evaluate', () => {
    // Order is the requirement, not mere presence.
    const rules = [
      { action: 'respond', match_response: 'g_a' },
      { action: 'evaluate', tag: 'g_a', server: 'a' },
    ]
    expect(findOrphanRespondRules(rules)).toEqual([0])
  })

  it('accepts a correctly ordered group', () => {
    const rules = [
      { action: 'evaluate', tag: 'g_a', server: 'a' },
      { action: 'evaluate', tag: 'g_b', server: 'b' },
      { action: 'respond', match_response: 'g_a', race: true },
      { action: 'respond', match_response: 'g_b', race: true },
    ]
    expect(findOrphanRespondRules(rules)).toEqual([])
  })

  it('treats match_response:true as referring to the latest untagged evaluate', () => {
    // sing-box: `match_response: true` references the response of the latest
    // evaluate WITHOUT a tag. A tagged evaluate does not satisfy it.
    expect(
      findOrphanRespondRules([
        { action: 'evaluate', server: 'a' },
        { action: 'respond', match_response: true },
      ]),
    ).toEqual([])

    expect(
      findOrphanRespondRules([
        { action: 'evaluate', tag: 'tagged', server: 'a' },
        { action: 'respond', match_response: true },
      ]),
    ).toEqual([1])
  })

  it('ignores rules that are not respond actions', () => {
    expect(findOrphanRespondRules([{ action: 'route', server: 'x' }])).toEqual([])
  })
})

describe('suggestGroupName', () => {
  it('derives a name from the first domain condition', () => {
    expect(suggestGroupName({ domain_suffix: ['deepseek.com', 'owolist.cn'] })).toBe('deepseek_com')
  })

  it('falls back when there is nothing to derive from', () => {
    expect(suggestGroupName({})).toBe('parallel')
  })

  it('prefers a rule set tag when that is the condition', () => {
    // Hyphens survive: they are valid in sing-box tags and `geosite-cn_aliyun`
    // reads better than a name mangled for no reason.
    expect(suggestGroupName({ rule_set: ['geosite-cn'] })).toBe('geosite-cn')
  })
})
