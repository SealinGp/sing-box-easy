import { describe, expect, test } from 'bun:test'
import { HEAT_FILL, HEAT_PULSE, HEAT_STROKE, HEAT_SWATCH, HEAT_TEXT, HEAT_TIERS } from './flowHeat'

// One class per tier, in every map — a missing entry renders an unstyled
// element with no error, which is exactly the kind of gap a colour scale
// cannot afford (the hottest tier is the one that has to be seen).
describe('heat class maps', () => {
  test('cover every tier', () => {
    for (const map of [HEAT_FILL, HEAT_STROKE, HEAT_TEXT, HEAT_PULSE, HEAT_SWATCH]) {
      expect(map.length).toBe(HEAT_TIERS)
      for (const cls of map) expect(cls.trim().length).toBeGreaterThan(0)
    }
  })

  test('name a dark variant for every light one', () => {
    for (const map of [HEAT_FILL, HEAT_STROKE, HEAT_TEXT, HEAT_PULSE, HEAT_SWATCH]) {
      for (const cls of map) expect(cls).toMatch(/dark:/)
    }
  })
})
