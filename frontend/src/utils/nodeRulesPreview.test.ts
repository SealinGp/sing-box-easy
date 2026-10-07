import { describe, expect, test } from 'bun:test'
import { normalizePreview } from './nodeRulesPreview'
import type { PreviewResult } from '../types/noderules'

const filter = (over: Record<string, unknown>) =>
  ({ id: 'f', name: 'F', outbound_type: 'urltest', is_fallback: false, member_count: 0, members: [], ...over })

describe('normalizePreview', () => {
  test('the payload from issue #14: a fallback filter with null members', () => {
    const wire = {
      endpoints: 3,
      filters: [
        filter({ id: 'filter_jp', name: 'JP', members: ['a', 'b', 'c'], member_count: 3 }),
        filter({ id: 'filter_other', name: 'Other Nodes', is_fallback: true, members: null }),
      ],
      unmatched: null,
    } as unknown as PreviewResult

    const preview = normalizePreview(wire)!
    expect(preview.filters[1]!.members).toEqual([])
    expect(preview.filters[1]!.member_count).toBe(0)
    expect(preview.unmatched).toEqual([])
    expect(preview.optional).toEqual([])
    // What the page does on every render must not throw.
    const seen: string[] = []
    for (const pf of preview.filters) for (const tag of pf.members) seen.push(tag)
    expect(seen).toEqual(['a', 'b', 'c'])
  })

  test('a field missing altogether is treated the same as null', () => {
    const preview = normalizePreview({ endpoints: 0, filters: [filter({ members: undefined })] } as unknown as PreviewResult)!
    expect(preview.filters[0]!.members).toEqual([])
    expect(preview.unmatched).toEqual([])
  })

  test('a well-formed preview passes through with its values intact', () => {
    const wire: PreviewResult = {
      endpoints: 2,
      filters: [filter({ members: ['a'], member_count: 1 }) as PreviewResult['filters'][number]],
      unmatched: ['b'],
      optional: ['direct'],
    }
    expect(normalizePreview(wire)).toEqual(wire)
    expect(normalizePreview(wire)).not.toBe(wire)
  })

  test('no preview stays no preview', () => {
    expect(normalizePreview(null)).toBeNull()
    expect(normalizePreview(undefined)).toBeNull()
  })

  test('filters that are not a list become an empty list', () => {
    expect(normalizePreview({ endpoints: 0, filters: null, unmatched: [] } as unknown as PreviewResult)!.filters).toEqual([])
  })
})
