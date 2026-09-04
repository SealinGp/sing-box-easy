/**
 * The heat palette: one class string per `Heat` tier, indexed by tier.
 *
 * Kept as literal strings in a plain TS file (not built from a hue variable)
 * because Tailwind only emits classes it can SEE in source; a class assembled
 * at runtime is a class that is never generated.
 *
 * The scale is blue → blue → amber → orange, deliberately not one hue at four
 * lightnesses: tiers 1–2 are "traffic" and stay in the brand ramp, tiers 3–4
 * are "a lot of traffic" and leave it, so the extreme is a different COLOUR
 * rather than a slightly darker one that has to be compared against its
 * neighbour. Amber and orange are the warm side of the app's status palette
 * (DESIGN.md §2); red is not used, because red is what a missing outbound is.
 */
import type { Heat } from './flowOverlay'

/** Number of tiers — the length every map below must have. */
export const HEAT_TIERS = 5

type HeatMap = readonly [string, string, string, string, string]

/** Ribbon stroke by tier. Tier 0 is the idle stroke a quiet ribbon already has. */
export const HEAT_STROKE: HeatMap = [
  'stroke-primary-400 dark:stroke-primary-600',
  'stroke-primary-400 dark:stroke-primary-600',
  'stroke-primary-600 dark:stroke-primary-400',
  'stroke-amber-500 dark:stroke-amber-400',
  'stroke-orange-600 dark:stroke-orange-400',
]

/** SVG text and rank-mark fill by tier. */
export const HEAT_FILL: HeatMap = [
  'fill-gray-400 dark:fill-gray-500',
  'fill-primary-600 dark:fill-primary-400',
  'fill-primary-700 dark:fill-primary-300',
  'fill-amber-600 dark:fill-amber-400',
  'fill-orange-600 dark:fill-orange-400',
]

/** The pulse's bright head — one step brighter than its ribbon. */
export const HEAT_PULSE: HeatMap = [
  'stroke-primary-600 dark:stroke-primary-300',
  'stroke-primary-600 dark:stroke-primary-300',
  'stroke-primary-700 dark:stroke-primary-200',
  'stroke-amber-600 dark:stroke-amber-300',
  'stroke-orange-600 dark:stroke-orange-300',
]

/** HTML text (the live strip's totals) by tier. */
export const HEAT_TEXT: HeatMap = [
  'text-gray-500 dark:text-gray-400',
  'text-primary-700 dark:text-primary-300',
  'text-primary-700 dark:text-primary-300',
  'text-amber-600 dark:text-amber-400',
  'text-orange-600 dark:text-orange-400',
]

/** Legend swatch background by tier. */
export const HEAT_SWATCH: HeatMap = [
  'bg-gray-300 dark:bg-slate-600',
  'bg-primary-400 dark:bg-primary-600',
  'bg-primary-600 dark:bg-primary-400',
  'bg-amber-500 dark:bg-amber-400',
  'bg-orange-600 dark:bg-orange-400',
]

export const heatClass = (map: HeatMap, heat: Heat): string => map[heat]
