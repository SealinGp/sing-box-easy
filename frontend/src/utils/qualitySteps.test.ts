import { describe, expect, it } from 'bun:test'
import { GREEN_STEP, RED_STEP, qualityStepColors } from './qualitySteps'

const count = (colors: string[], want: string) => colors.filter((c) => c === want).length

describe('qualityStepColors', () => {
  it('is all green when every node answered', () => {
    const colors = qualityStepColors(37, 37, 5)
    expect(colors).toHaveLength(5)
    expect(count(colors, GREEN_STEP)).toBe(5)
  })

  it('is all red when nothing answered', () => {
    expect(count(qualityStepColors(0, 37, 5), RED_STEP)).toBe(5)
  })

  it('never rounds a single dead node up to all green', () => {
    // 36/37 is 97.3%. Rounded onto 5 blocks that is 5 green — a feed with a
    // dead node drawn as flawless. The whole point of the gauge is that the
    // failure is visible at a glance, so the green count floors instead.
    const colors = qualityStepColors(36, 37, 5)
    expect(count(colors, RED_STEP)).toBeGreaterThanOrEqual(1)
    expect(count(colors, GREEN_STEP)).toBe(4)
  })

  it('never hides a single working node either', () => {
    // The mirror case: 1/37 up floors to 0 green, which would draw a feed that
    // still has a usable exit as totally dead.
    const colors = qualityStepColors(1, 37, 5)
    expect(count(colors, GREEN_STEP)).toBe(1)
    expect(count(colors, RED_STEP)).toBe(4)
  })

  it('orders green before red so the bar fills from the left', () => {
    const colors = qualityStepColors(3, 5, 5)
    expect(colors).toEqual([GREEN_STEP, GREEN_STEP, GREEN_STEP, RED_STEP, RED_STEP])
  })

  it('scales to any step count', () => {
    const colors = qualityStepColors(5, 10, 10)
    expect(colors).toHaveLength(10)
    expect(count(colors, GREEN_STEP)).toBe(5)
  })

  it('returns nothing to colour when there is nothing measured', () => {
    // Not "all red": a subscription with no tested nodes has not failed, and
    // painting it as a total outage is the one thing the backend refuses to
    // record. The caller renders the trail instead.
    expect(qualityStepColors(0, 0, 5)).toEqual([])
  })

  it('survives nonsense input rather than rendering NaN blocks', () => {
    expect(qualityStepColors(-1, 5, 5)).toHaveLength(5)
    expect(qualityStepColors(99, 5, 5)).toHaveLength(5)
    expect(qualityStepColors(3, 5, 0)).toEqual([])
  })
})
