import { describe, expect, test } from 'bun:test'
import { columnCount, dealIntoColumns, displayName, latencyTier, sortMembers, stripFitsOneLine, tierShares, visibleGroups, withGroupDelays, withNodeDelay } from './proxyGroups'
import type { RuntimeGroup, RuntimeMember } from '../types/runtime'

const member = (name: string, delay: number, group = false): RuntimeMember => ({ name, delay, group, type: 'VLESS' })
const group = (name: string, members: RuntimeMember[], now = members[0]?.name ?? ''): RuntimeGroup => ({
  name, type: 'URLTest', now, delay: 0, switchable: false, members,
})

describe('latencyTier', () => {
  test('zero and invalid values are "none", never "good"', () => {
    expect(latencyTier(0)).toBe('none')
    expect(latencyTier(-1)).toBe('none')
    expect(latencyTier(Number.NaN)).toBe('none')
  })
  test('thresholds', () => {
    expect(latencyTier(1)).toBe('good')
    expect(latencyTier(399)).toBe('good')
    expect(latencyTier(400)).toBe('fair')
    expect(latencyTier(799)).toBe('fair')
    expect(latencyTier(800)).toBe('poor')
  })
})

describe('displayName', () => {
  test('strips the fingerprint and subscription id from a subscription tag', () => {
    expect(displayName('🇭🇰香港高速01|BGP|CMCU a1b2c3d4 | sub_0123456789abcdef0123456789abcdef')).toBe('🇭🇰香港高速01|BGP|CMCU')
    expect(displayName('新加坡 01 0f1e2d3c | sub_demo')).toBe('新加坡 01')
  })
  test('leaves hand-named outbounds alone, including ones containing a pipe', () => {
    expect(displayName('自动选择')).toBe('自动选择')
    expect(displayName('HK | premium')).toBe('HK | premium')
    expect(displayName('node deadbeef')).toBe('node deadbeef')
  })
  test('never returns an empty name', () => {
    expect(displayName(' f45c32ff | sub_1')).toBe(' f45c32ff | sub_1')
  })
})

describe('sortMembers', () => {
  const members = [member('b', 300), member('dead', 0), member('a', 50)]
  test('latency puts untested and failed members last', () => {
    expect(sortMembers(members, 'latency').map((m) => m.name)).toEqual(['a', 'b', 'dead'])
  })
  test('name sorts by display name', () => {
    expect(sortMembers(members, 'name').map((m) => m.name)).toEqual(['a', 'b', 'dead'])
  })
  test('default keeps config order and never mutates the input', () => {
    expect(sortMembers(members, 'default').map((m) => m.name)).toEqual(['b', 'dead', 'a'])
    sortMembers(members, 'latency')
    expect(members.map((m) => m.name)).toEqual(['b', 'dead', 'a'])
  })
})

describe('visibleGroups', () => {
  const groups = [
    group('自动选择', [member('香港 01 aaaaaaaa | sub_1', 40), member('日本 01 bbbbbbbb | sub_1', 80)]),
    group('US', [member('美国 01 cccccccc | sub_2', 160)]),
  ]
  test('an empty query keeps everything', () => {
    expect(visibleGroups(groups, '  ', 'default')).toHaveLength(2)
  })
  test('a group-name match keeps all of its members', () => {
    const result = visibleGroups(groups, '自动', 'default')
    expect(result).toHaveLength(1)
    expect(result[0]!.members).toHaveLength(2)
  })
  test('a member match keeps only that member and drops empty groups', () => {
    const result = visibleGroups(groups, '日本', 'default')
    expect(result).toHaveLength(1)
    expect(result[0]!.members.map((m) => m.name)).toEqual(['日本 01 bbbbbbbb | sub_1'])
  })
  test('the full tag is searchable, so a fingerprint finds its node', () => {
    expect(visibleGroups(groups, 'CCCCCCCC', 'default')[0]!.group.name).toBe('US')
  })
})

describe('withGroupDelays', () => {
  test('members missing from the result are marked failed, and the group follows its current node', () => {
    const before = group('auto', [member('a', 10), member('b', 20)], 'b')
    const after = withGroupDelays(before, { b: 77 })
    expect(after.members.map((m) => m.delay)).toEqual([0, 77])
    expect(after.delay).toBe(77)
    expect(before.members[0]!.delay).toBe(10)
  })
  test('a nested group member keeps its resolved delay', () => {
    const before = { ...group('sel', [member('inner', 90, true), member('a', 10)], 'inner'), delay: 90 }
    const after = withGroupDelays(before, { a: 15 })
    expect(after.members[0]!.delay).toBe(90)
    expect(after.delay).toBe(90)
  })
})

describe('withNodeDelay', () => {
  test('updates the node in every group that holds it, and each group that is using it', () => {
    const groups = [
      group('auto', [member('a', 10), member('b', 20)], 'b'),
      group('sel', [member('b', 20), member('c', 30)], 'c'),
      group('other', [member('d', 40)], 'd'),
    ]
    const after = withNodeDelay(groups, 'b', 77)
    expect(after[0]!.members.map((m) => m.delay)).toEqual([10, 77])
    expect(after[0]!.delay).toBe(77)
    expect(after[1]!.members.map((m) => m.delay)).toEqual([77, 30])
    // `sel` is using c, so its own figure is not b's.
    expect(after[1]!.delay).toBe(0)
    // A group without the node is the same object: nothing to re-render.
    expect(after[2]).toBe(groups[2]!)
    expect(groups[0]!.members[1]!.delay).toBe(20)
  })
  test('a failed test is a result: the old figure is replaced by 0', () => {
    const before = [{ ...group('auto', [member('a', 10)], 'a'), delay: 10 }]
    const after = withNodeDelay(before, 'a', 0)
    expect(after[0]!.members[0]!.delay).toBe(0)
    expect(after[0]!.delay).toBe(0)
  })
  test('a nested group member takes the figure too', () => {
    const before = [{ ...group('sel', [member('inner', 90, true)], 'inner'), delay: 90 }]
    const after = withNodeDelay(before, 'inner', 120)
    expect(after[0]!.members[0]!.delay).toBe(120)
    expect(after[0]!.delay).toBe(120)
  })
})

describe('stripFitsOneLine', () => {
  test('counts blocks and the gaps BETWEEN them, not after the last', () => {
    // 10 blocks: 10 * 11 + 9 * 4 = 146
    expect(stripFitsOneLine(10, 146)).toBe(true)
    expect(stripFitsOneLine(10, 145)).toBe(false)
    expect(stripFitsOneLine(94, 414)).toBe(false)
    expect(stripFitsOneLine(22, 414)).toBe(true)
  })
  test('an unmeasured width counts as fitting, so small groups never flash a bar', () => {
    expect(stripFitsOneLine(94, 0)).toBe(true)
    expect(stripFitsOneLine(0, 100)).toBe(true)
  })
})

describe('tierShares', () => {
  test('counts per tier, best first, with empty tiers left out', () => {
    const members = [member('a', 100), member('b', 900), member('c', 0), member('d', 120), member('e', 0)]
    expect(tierShares(members)).toEqual([
      { tier: 'good', count: 2 },
      { tier: 'poor', count: 1 },
      { tier: 'none', count: 2 },
    ])
  })
  test('an empty group has no shares', () => {
    expect(tierShares([])).toEqual([])
  })
})

describe('columnCount', () => {
  test('one column until two full cards and the gap between them fit', () => {
    expect(columnCount(0)).toBe(1)
    expect(columnCount(440)).toBe(1)
    expect(columnCount(891)).toBe(1)
    expect(columnCount(892)).toBe(2)
    expect(columnCount(1344)).toBe(3)
    expect(columnCount(Number.NaN)).toBe(1)
  })
})

describe('dealIntoColumns', () => {
  test('deals left to right so config order reads across the top', () => {
    expect(dealIntoColumns([1, 2, 3, 4, 5, 6, 7], 3)).toEqual([[1, 4, 7], [2, 5], [3, 6]])
  })
  test('one column keeps the order, and a bad count still yields one column', () => {
    expect(dealIntoColumns([1, 2, 3], 1)).toEqual([[1, 2, 3]])
    expect(dealIntoColumns([1, 2], 0)).toEqual([[1, 2]])
  })
  test('more columns than items leaves the extras empty rather than missing', () => {
    expect(dealIntoColumns([1], 3)).toEqual([[1], [], []])
  })
})
