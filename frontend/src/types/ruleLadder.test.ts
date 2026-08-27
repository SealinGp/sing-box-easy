import { describe, expect, test } from 'bun:test'
import { markUnreached, type LadderRung } from './ruleLadder'

const rung = (index: number, state: LadderRung['state']): LadderRung => ({
  index,
  state,
  summary: `rule ${index}`,
  outcome: 'route(proxy)',
  deciding: false,
})

describe('markUnreached', () => {
  /**
   * The reason this function exists.
   *
   * `dnsprobe.Attribute` keeps evaluating after it has decided (match.go:94-115
   * — it sets `decided` and continues the loop), so the payload carries a real
   * verdict for rules the query never reached. Rendering "no match" for one
   * asserts a comparison that never happened, which is the specific lie the
   * lamps exist to stop telling.
   */
  test('re-labels verdicts below the deciding rung', () => {
    const rungs = [
      rung(0, 'not_matched'),
      rung(1, 'matched'),
      rung(2, 'not_matched'),
      rung(3, 'matched'),
    ]

    const out = markUnreached(rungs, 1)

    expect(out.map((r) => r.state)).toEqual(['not_matched', 'matched', 'skipped', 'skipped'])
  })

  test('leaves everything at or above the match untouched', () => {
    const rungs = [rung(0, 'unevaluated'), rung(1, 'not_matched'), rung(2, 'matched')]

    expect(markUnreached(rungs, 2).map((r) => r.state)).toEqual([
      'unevaluated',
      'not_matched',
      'matched',
    ])
  })

  /**
   * A fallthrough means every rule WAS consulted and none matched. Blanking
   * them would hide the fact that the whole ladder ran.
   */
  test('changes nothing when the query falls through to final', () => {
    const rungs = [rung(0, 'not_matched'), rung(1, 'not_matched')]

    expect(markUnreached(rungs, -1)).toEqual(rungs)
  })

  test('does not mutate its input', () => {
    const rungs = [rung(0, 'matched'), rung(1, 'not_matched')]
    const before = JSON.stringify(rungs)

    markUnreached(rungs, 0)

    expect(JSON.stringify(rungs)).toBe(before)
  })
})
