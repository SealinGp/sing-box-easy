import { describe, expect, it } from 'bun:test'
import { contrastRatio, labPalettes } from './palettes'

describe('lab palette accessibility', () => {
  for (const palette of labPalettes) {
    it(`${palette.name} keeps normal text and feedback readable`, () => {
      for (const [foreground, background] of [
        [palette.ink, palette.surface], [palette.muted, palette.surface], [palette.muted, palette.canvas],
        [palette.onPrimary, palette.primary], [palette.onPrimary, palette.primaryHover],
        [palette.success, palette.surface], [palette.warning, palette.surface], [palette.danger, palette.surface],
        [palette.navMuted, palette.nav], [palette.navInk, palette.navActive],
      ] as const) expect(contrastRatio(foreground, background)).toBeGreaterThanOrEqual(4.5)
    })
    it(`${palette.name} gives field boundaries enough contrast`, () => {
      expect(contrastRatio(palette.fieldBorder, palette.field)).toBeGreaterThanOrEqual(3)
    })
  }
})
