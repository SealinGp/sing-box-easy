import { expect, test } from 'bun:test'
import { inspectorMembers } from './inspectorMembers'
import type { RuntimeMember } from '../types/runtime'
const members: RuntimeMember[] = Array.from({ length: 300 }, (_, i) => ({ name: `Node ${i}`, type: 'VLESS', delay: i + 1, group: false }))
test('active member leads large groups, preserving all other ordering and source data', () => {
  const ordered = inspectorMembers(members, 'Node 299', '')
  expect(ordered[0]!.name).toBe('Node 299')
  expect(ordered.slice(1)).toEqual(members.slice(0, -1))
  expect(members[0]!.name).toBe('Node 0')
})
test('filter applies before stable active-first ordering and missing selection adds no phantom node', () => {
  expect(inspectorMembers(members, 'Node 299', 'NODE 29').map(x => x.name)).toEqual(['Node 299', 'Node 29', ...Array.from({length: 9}, (_, i) => `Node ${290+i}`)])
  expect(inspectorMembers(members, 'gone', 'missing')).toEqual([])
  expect(inspectorMembers(members, 'gone', '').length).toBe(300)
})
