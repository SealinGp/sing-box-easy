import { describe, expect, test } from 'bun:test'
import { destinationFor, groupProblems, parseConfigProblems, type ConfigProblem } from './configProblems'

// The body the backend sends for issue #14's situation.
const body = {
  stage: 'panel_guard',
  reason: 'outbound_reference',
  problems: [
    { kind: 'missing_outbound', field: 'route.rules', where: 'route.rules[3]', tag: 'JP' },
    { kind: 'missing_outbound', field: 'route.final', where: 'route.final', tag: 'Other Nodes' },
    { kind: 'empty_group', field: 'dns.servers', where: 'dns.servers[0].detour', tag: 'hollow' },
  ],
}

describe('parseConfigProblems', () => {
  test('reads the backend body', () => {
    expect(parseConfigProblems(body)).toEqual(body.problems as ConfigProblem[])
  })

  test('any other error body is not a config problem', () => {
    expect(parseConfigProblems(null)).toEqual([])
    expect(parseConfigProblems('boom')).toEqual([])
    expect(parseConfigProblems({ stage: 'core_check', core_version: '1.12.25' })).toEqual([])
    expect(parseConfigProblems({ reason: 'something_else', problems: body.problems })).toEqual([])
    expect(parseConfigProblems({ reason: 'outbound_reference', problems: 'nope' })).toEqual([])
  })

  test('entries this build cannot act on are dropped, not shown as blank rows', () => {
    const problems = parseConfigProblems({
      reason: 'outbound_reference',
      problems: [
        { kind: 'missing_outbound', field: 'route.final', where: 'route.final', tag: 'ok' },
        { kind: 'missing_outbound', field: 'ntp.detour', where: 'ntp.detour', tag: 'future-field' },
        { kind: 'future_kind', field: 'route.final', where: 'route.final', tag: 'x' },
        { kind: 'missing_outbound', field: 'route.final', where: 'route.final', tag: '' },
        { kind: 'missing_outbound', field: 'toString', where: 'x', tag: 'prototype-key' },
        null,
        'junk',
      ],
    })
    expect(problems.map((problem) => problem.tag)).toEqual(['ok'])
  })

  test('a missing path falls back to the field name', () => {
    const [problem] = parseConfigProblems({
      reason: 'outbound_reference',
      problems: [{ kind: 'missing_outbound', field: 'route.final', tag: 'x' }],
    })
    expect(problem!.where).toBe('route.final')
  })
})

describe('destinations', () => {
  test('every field leads to the page that edits it', () => {
    expect(destinationFor('route.final')).toBe('/dashboard/route/final-policy')
    expect(destinationFor('route.rules')).toBe('/dashboard/route/rules')
    expect(destinationFor('route.rule_set')).toBe('/dashboard/route/rule-sets')
    expect(destinationFor('dns.servers')).toBe('/dashboard/dns/servers')
  })
})

describe('groupProblems', () => {
  test('one group per page, route.final first, empty groups omitted', () => {
    const groups = groupProblems(parseConfigProblems(body))
    expect(groups.map((group) => group.field)).toEqual(['route.final', 'route.rules', 'dns.servers'])
    expect(groups[0]!.to).toBe('/dashboard/route/final-policy')
    expect(groups[0]!.problems.map((problem) => problem.tag)).toEqual(['Other Nodes'])
  })
  test('no problems, no groups', () => {
    expect(groupProblems([])).toEqual([])
  })
})
