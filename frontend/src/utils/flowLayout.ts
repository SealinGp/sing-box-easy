/**
 * Places the route topology on a canvas: three columns and the ribbons between.
 *
 * Deterministic and pure — no DOM measurement, no layout solver, no graph
 * library. The diagram is a fixed tripartite layout, so positions follow from
 * counts alone, and computing them here rather than in the component means the
 * geometry is unit-testable and the component is only markup.
 *
 * WHY NOT MEASURE THE DOM
 * ───────────────────────
 * The obvious alternative — HTML boxes in a grid, ribbons drawn from measured
 * positions — needs a ResizeObserver and repaints on every reflow, and gets the
 * first frame wrong. A fixed `viewBox` scales the whole picture to whatever
 * width the card has, which is what responsiveness means here. The cost is that
 * text has to be fitted by estimate rather than measured, which `displayWidth`
 * does well enough for tags: the labels are proxy names, and the full value is
 * always in a `<title>`.
 */
import type { RouteTopology, TopologyExit, TopologyRule } from '../types/routeTopology'

/* ── Geometry. One place, so the component never invents a number. ────────── */

const PAD = 10
const COL_INBOUND_W = 170
/**
 * The rule column is deliberately the widest thing on the canvas. A rule's
 * conditions are the only part of the diagram that is free text, and clipping
 * them to an ellipsis is what turns a readable ladder into a list of `rule_set:
 * sea-rul…`. The other two columns hold tags, which are short.
 */
const COL_RULE_W = 640
const COL_EXIT_W = 260
/** Horizontal room for the ribbons to curve in. */
const GUTTER = 100

const INBOUND_H = 34
const INBOUND_GAP = 10
const RULE_H = 26
const RULE_GAP = 4
const EXIT_H = 38
const EXIT_GAP = 10
/** Vertical room between the last rule and the fall-through row. */
const FALLTHROUGH_GAP = 16

const COL_RULE_X = PAD + COL_INBOUND_W + GUTTER
const COL_EXIT_X = COL_RULE_X + COL_RULE_W + GUTTER
const CANVAS_W = COL_EXIT_X + COL_EXIT_W + PAD

/** How horizontally lazy the bezier is. 0 = straight line, 0.5 = very round. */
const CURVE = 0.45

/**
 * Font sizes, shared with the component so text fitting and rendering cannot
 * disagree — a label fitted at one size and drawn at another overflows silently.
 */
export const FONT = { node: 12, meta: 9, rule: 11 } as const

/** Inner padding inside a node box, per side. */
export const NODE_PAD = 10

/* ── Text fitting ─────────────────────────────────────────────────────────── */

/**
 * Approximate rendered width in "half-em" units.
 *
 * Tags in this panel are routinely CJK or emoji (`🤖 AI`, `➡️ 直连`, `香港01`),
 * and counting those as one character each overflows the node by roughly double.
 * Anything outside Latin/punctuation is treated as full-width, which is right
 * for CJK and close enough for emoji.
 */
export function displayWidth(text: string): number {
  let width = 0
  for (const char of text) {
    const code = char.codePointAt(0) ?? 0
    width += code < 0x0300 ? 1 : 2
  }
  return width
}

/**
 * Average advance width of one half-em unit, as a fraction of the font size.
 *
 * Deliberately a slight over-estimate: erring high clips a character early,
 * erring low overflows the node, and only one of those is recoverable by the
 * reader (the full string is in the `<title>`).
 */
const UNIT_EM = 0.55

/** Fits text to a pixel width at a given font size. See `UNIT_EM`. */
export function fitToBox(text: string, availablePx: number, fontSize: number): string {
  return truncateToWidth(text, Math.floor(availablePx / (fontSize * UNIT_EM)))
}

/** Clips to a width budget, reserving one unit for the ellipsis. */
export function truncateToWidth(text: string, budget: number): string {
  if (displayWidth(text) <= budget) return text
  let width = 0
  let out = ''
  for (const char of text) {
    const next = width + displayWidth(char)
    // Reserve a unit for the '…' itself, so the result never exceeds budget.
    if (next > budget - 1) break
    out += char
    width = next
  }
  return `${out}…`
}

/* ── Output shapes ────────────────────────────────────────────────────────── */

export interface Box {
  x: number
  y: number
  width: number
  height: number
}

export interface InboundBox extends Box {
  tag: string
  type: string
  label: string
  listenPort?: number
}

export interface RuleBox extends Box {
  index: number
  rule: TopologyRule
}

export interface ExitBox extends Box {
  id: string
  exit: TopologyExit
  label: string
}

export interface FallthroughBox extends Box {
  exitId: string
  source: RouteTopology['fallthrough']['source']
}

export type RibbonKind = 'inbound' | 'rule' | 'final'

export interface Ribbon {
  /** Unique within a layout — usable as a `:key`. */
  id: string
  kind: RibbonKind
  d: string
  fromX: number
  fromY: number
  toX: number
  toY: number
  /** Set for `rule` ribbons. */
  ruleIndex?: number
  /** Set for `rule` and `final` ribbons. */
  exitId?: string
  /** Set for `inbound` ribbons. */
  inboundTag?: string
}

export interface FlowLayout {
  width: number
  height: number
  inbounds: InboundBox[]
  rules: RuleBox[]
  exits: ExitBox[]
  fallthrough: FallthroughBox
  ribbons: Ribbon[]
  /** Where every inbound ribbon lands: the top of the rule ladder. */
  entry: { x: number; y: number }
}

/* ── Layout ───────────────────────────────────────────────────────────────── */

function curve(fromX: number, fromY: number, toX: number, toY: number): string {
  const dx = Math.max((toX - fromX) * CURVE, 12)
  const round = (n: number) => Math.round(n * 100) / 100
  return `M ${round(fromX)} ${round(fromY)} C ${round(fromX + dx)} ${round(fromY)}, ${round(
    toX - dx,
  )} ${round(toY)}, ${round(toX)} ${round(toY)}`
}

/**
 * Orders exits by the mean index of the rules pointing at them.
 *
 * A one-pass barycenter heuristic — the cheap half of what a real graph layout
 * does about edge crossings. First-reference order (what the model produces,
 * and the right order for a list) puts a fall-through-only exit at the top,
 * where its ribbon crosses every other one on the way down.
 */
function orderExits(exits: readonly TopologyExit[]): TopologyExit[] {
  const barycenter = (exit: TopologyExit): number =>
    exit.ruleIndices.length === 0
      ? Number.POSITIVE_INFINITY
      : exit.ruleIndices.reduce((sum, index) => sum + index, 0) / exit.ruleIndices.length

  return [...exits]
    .map((exit, position) => ({ exit, position, key: barycenter(exit) }))
    .sort((a, b) => a.key - b.key || a.position - b.position)
    .map((entry) => entry.exit)
}

/** Keeps a coordinate inside a range. */
function clamp(value: number, low: number, high: number): number {
  return Math.min(Math.max(value, low), high)
}

/** Stacks a column and centres it against the canvas's content height. */
function stack(count: number, itemHeight: number, gap: number, contentHeight: number): number[] {
  const total = count * itemHeight + Math.max(count - 1, 0) * gap
  const top = PAD + Math.max((contentHeight - total) / 2, 0)
  return Array.from({ length: count }, (_, i) => top + i * (itemHeight + gap))
}

export function layoutRouteFlow(topology: RouteTopology): FlowLayout {
  const { inbounds, rules, fallthrough } = topology
  const exits = orderExits(topology.exits)

  // The rule ladder sets the canvas height; the other two columns centre
  // against it. The fall-through row hangs below the last rule.
  const ladderH = rules.length * RULE_H + Math.max(rules.length - 1, 0) * RULE_GAP
  const withFallthrough = ladderH + FALLTHROUGH_GAP + RULE_H
  const inboundsH = inbounds.length * INBOUND_H + Math.max(inbounds.length - 1, 0) * INBOUND_GAP
  const exitsH = exits.length * EXIT_H + Math.max(exits.length - 1, 0) * EXIT_GAP
  const contentH = Math.max(withFallthrough, inboundsH, exitsH)
  const height = contentH + PAD * 2

  const exitTops = stack(exits.length, EXIT_H, EXIT_GAP, contentH)
  const ladderTop = PAD + Math.max((contentH - withFallthrough) / 2, 0)

  /**
   * Traffic enters at the TOP of the ladder, so that is where the inbound
   * ribbons land — at rule #0's left edge, not at the canvas's midpoint.
   * Centring the inbound column against the full height instead (which is what
   * a symmetric three-column layout wants to do) parks it beside rule #14 and
   * sends every ribbon on a long diagonal that reads as "these inbounds enter
   * halfway down", which is the one thing the ladder must not say.
   */
  const entryY = ladderTop + RULE_H / 2
  const inboundsTop = clamp(entryY - inboundsH / 2, PAD, PAD + Math.max(contentH - inboundsH, 0))
  const inboundTops = Array.from(
    { length: inbounds.length },
    (_, i) => inboundsTop + i * (INBOUND_H + INBOUND_GAP),
  )

  const inboundBoxes: InboundBox[] = inbounds.map((inbound, i) => ({
    x: PAD,
    y: inboundTops[i]!,
    width: COL_INBOUND_W,
    height: INBOUND_H,
    tag: inbound.tag,
    type: inbound.type,
    listenPort: inbound.listenPort,
    label: fitToBox(inbound.tag || inbound.type, COL_INBOUND_W - NODE_PAD * 2, FONT.node),
  }))

  const ruleBoxes: RuleBox[] = rules.map((rule, i) => ({
    x: COL_RULE_X,
    y: ladderTop + i * (RULE_H + RULE_GAP),
    width: COL_RULE_W,
    height: RULE_H,
    index: rule.index,
    rule,
  }))

  const exitBoxes: ExitBox[] = exits.map((exit, i) => ({
    x: COL_EXIT_X,
    y: exitTops[i]!,
    width: COL_EXIT_W,
    height: EXIT_H,
    id: exit.id,
    exit,
    label: fitToBox(exit.label, COL_EXIT_W - NODE_PAD * 2, FONT.node),
  }))

  const fallthroughBox: FallthroughBox = {
    x: COL_RULE_X,
    y: ladderTop + ladderH + FALLTHROUGH_GAP,
    width: COL_RULE_W,
    height: RULE_H,
    exitId: fallthrough.exitId,
    source: fallthrough.source,
  }

  const entry = { x: COL_RULE_X, y: entryY }

  const ribbons: Ribbon[] = []

  for (const box of inboundBoxes) {
    const fromX = box.x + box.width
    const fromY = box.y + box.height / 2
    ribbons.push({
      id: `in:${box.tag || box.type}:${box.y}`,
      kind: 'inbound',
      inboundTag: box.tag,
      d: curve(fromX, fromY, entry.x, entry.y),
      fromX,
      fromY,
      toX: entry.x,
      toY: entry.y,
    })
  }

  /**
   * Ribbons landing on one exit fan across its left edge rather than stacking
   * on a single point — several rules reaching one outbound is the fact the
   * card exists to show, and a single attach point renders it as one line.
   */
  const landed = new Map<string, number>()
  const incoming = new Map<string, number>()
  for (const box of exitBoxes) {
    incoming.set(box.id, box.exit.ruleIndices.length + (box.exit.isFinal ? 1 : 0))
  }

  const attach = (box: ExitBox): number => {
    const total = incoming.get(box.id) ?? 1
    const nth = (landed.get(box.id) ?? 0) + 1
    landed.set(box.id, nth)
    return box.y + (box.height * nth) / (total + 1)
  }

  const exitById = new Map(exitBoxes.map((box) => [box.id, box]))

  for (const box of ruleBoxes) {
    const exitId = box.rule.exitId
    if (!exitId) continue
    const target = exitById.get(exitId)
    if (!target) continue
    const fromX = box.x + box.width
    const fromY = box.y + box.height / 2
    const toY = attach(target)
    ribbons.push({
      id: `rule:${box.index}`,
      kind: 'rule',
      ruleIndex: box.index,
      exitId,
      d: curve(fromX, fromY, target.x, toY),
      fromX,
      fromY,
      toX: target.x,
      toY,
    })
  }

  const finalTarget = exitById.get(fallthrough.exitId)
  if (finalTarget) {
    const fromX = fallthroughBox.x + fallthroughBox.width
    const fromY = fallthroughBox.y + fallthroughBox.height / 2
    const toY = attach(finalTarget)
    ribbons.push({
      id: 'final',
      kind: 'final',
      exitId: fallthrough.exitId,
      d: curve(fromX, fromY, finalTarget.x, toY),
      fromX,
      fromY,
      toX: finalTarget.x,
      toY,
    })
  }

  return {
    width: CANVAS_W,
    height,
    inbounds: inboundBoxes,
    rules: ruleBoxes,
    exits: exitBoxes,
    fallthrough: fallthroughBox,
    ribbons,
    entry,
  }
}
