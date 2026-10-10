/**
 * Pure helpers for the runtime Proxies page. Kept out of the `.vue` files so
 * the rules that are easy to get subtly wrong — what counts as "slow", what a
 * tag's readable part is, where an untested node sorts — are tested directly.
 */
import type { RuntimeGroup, RuntimeMember } from '../types/runtime'

export type LatencyTier = 'good' | 'fair' | 'poor' | 'none'

/**
 * zashboard's defaults (its low/medium latency settings), so a node is the same
 * colour here as on the dashboard operators are coming from. Measured on a
 * real feed, 200/500 painted almost every working overseas node amber or red.
 */
export const LATENCY_GOOD_MS = 400
export const LATENCY_FAIR_MS = 800

/**
 * 0 is "no result" — untested, or the last test failed. It is NOT a fast node,
 * and must never share a colour with one.
 */
export function latencyTier(delay: number): LatencyTier {
  if (!Number.isFinite(delay) || delay <= 0) return 'none'
  if (delay < LATENCY_GOOD_MS) return 'good'
  if (delay < LATENCY_FAIR_MS) return 'fair'
  return 'poor'
}

/**
 * A subscription node is tagged `<name> <8-hex fingerprint> | <subscription id>`
 * (see "Subscription outbound tags" in CLAUDE.md). Only the first part is for
 * people. The pattern is anchored and strict on purpose: a hand-named outbound
 * that merely contains a "|" must be shown whole.
 */
const NODE_TAG_SUFFIX = / [0-9a-f]{8} \| \S+$/

export function displayName(tag: string): string {
  const name = tag.replace(NODE_TAG_SUFFIX, '')
  return name.trim() === '' ? tag : name
}

export type MemberSort = 'default' | 'latency' | 'name'

/**
 * Returns a NEW array. Untested/failed members (delay 0) sort LAST under
 * `latency` — sorting them first would put every dead node at the top of a
 * list the operator opened to find the fastest one.
 */
export function sortMembers(members: readonly RuntimeMember[], sort: MemberSort): RuntimeMember[] {
  const copy = [...members]
  if (sort === 'latency') {
    const rank = (member: RuntimeMember) => (member.delay > 0 ? member.delay : Number.POSITIVE_INFINITY)
    return copy.sort((a, b) => rank(a) - rank(b))
  }
  if (sort === 'name') {
    return copy.sort((a, b) => displayName(a.name).localeCompare(displayName(b.name)))
  }
  return copy
}

export interface VisibleGroup {
  group: RuntimeGroup
  members: RuntimeMember[]
}

/**
 * Applies the search box and the sort.
 *
 * A query matching the GROUP's name keeps all of its members; otherwise only
 * the matching members are kept and a group left with none is dropped. The
 * full tag is searched as well as the display name, so pasting a fingerprint
 * out of config.json finds its node.
 */
export function visibleGroups(groups: readonly RuntimeGroup[], query: string, sort: MemberSort): VisibleGroup[] {
  const needle = query.trim().toLowerCase()
  const result: VisibleGroup[] = []
  for (const group of groups) {
    const groupMatches = needle === '' || group.name.toLowerCase().includes(needle)
    const members = groupMatches
      ? group.members
      : group.members.filter((member) => member.name.toLowerCase().includes(needle))
    if (!groupMatches && members.length === 0) continue
    result.push({ group, members: sortMembers(members, sort) })
  }
  return result
}

/** Folds a group test's results into a group, returning a new object. */
export function withGroupDelays(group: RuntimeGroup, delays: Record<string, number>): RuntimeGroup {
  const members = group.members.map((member) =>
    // A member that is itself a group is not in the result map by its own
    // name; its delay is whatever it resolves to, which a refetch supplies.
    member.group ? member : { ...member, delay: delays[member.name] ?? 0 },
  )
  const current = members.find((member) => member.name === group.now)
  return { ...group, members, delay: current && !current.group ? current.delay : group.delay }
}

/**
 * Folds ONE node's test result into every group that lists it, returning a new
 * array. A node is routinely a member of several groups, and a figure that
 * changed on the card that was clicked but not on its neighbour reads as two
 * different nodes. Groups without the node are returned as the same object.
 *
 * Unlike a group test, the node here was dialled by its own name — so a member
 * that is itself a group takes the figure too: it is what sing-box measured
 * through that group's current node.
 */
export function withNodeDelay(groups: readonly RuntimeGroup[], name: string, delay: number): RuntimeGroup[] {
  return groups.map((group) => {
    if (!group.members.some((member) => member.name === name)) return group
    return {
      ...group,
      members: group.members.map((member) => (member.name === name ? { ...member, delay } : member)),
      delay: group.now === name ? delay : group.delay,
    }
  })
}

/** Side of one latency block in the collapsed strip, and the gap between two. */
export const STRIP_BLOCK_PX = 11
export const STRIP_GAP_PX = 4

/**
 * Whether `count` blocks fit on ONE line of `width` pixels.
 *
 * When they do not, the collapsed card draws a single proportional bar instead
 * of wrapping: three rows of ninety-four blocks is a texture, not a reading,
 * and it makes one card several times taller than its neighbours. A width of 0
 * means "not measured yet" and is treated as fitting, so the first paint is the
 * blocks rather than a bar that flips back a frame later for every small group.
 */
export function stripFitsOneLine(count: number, width: number): boolean {
  if (width <= 0 || count <= 0) return true
  return count * STRIP_BLOCK_PX + (count - 1) * STRIP_GAP_PX <= width
}

export interface TierShare {
  tier: LatencyTier
  count: number
}

/** Best first, so the bar reads left to right from healthy to dead. */
const TIER_ORDER: readonly LatencyTier[] = ['good', 'fair', 'poor', 'none']

/**
 * How a group's members split across the latency tiers, in a fixed order and
 * with empty tiers omitted — a zero-width segment would still take a gap.
 */
export function tierShares(members: readonly RuntimeMember[]): TierShare[] {
  const counts: Record<LatencyTier, number> = { good: 0, fair: 0, poor: 0, none: 0 }
  for (const member of members) counts[latencyTier(member.delay)] += 1
  return TIER_ORDER.filter((tier) => counts[tier] > 0).map((tier) => ({ tier, count: counts[tier] }))
}

/** A group card is not drawn narrower than this; it sets how many columns fit. */
export const GROUP_COLUMN_MIN_PX = 440
export const GROUP_COLUMN_GAP_PX = 12

/** How many card columns fit in `width`. Always at least one. */
export function columnCount(width: number, minWidth = GROUP_COLUMN_MIN_PX, gap = GROUP_COLUMN_GAP_PX): number {
  if (!Number.isFinite(width) || width <= 0) return 1
  return Math.max(1, Math.floor((width + gap) / (minWidth + gap)))
}

/**
 * Deals items into `count` columns, left to right: item 0 to column 0, item 1
 * to column 1, and so on round.
 *
 * Round-robin rather than "shortest column first": it keeps config order
 * readable across the top of the page (the first N groups head the N columns)
 * and — the part that matters here — it does not depend on card heights, so
 * expanding a card can never make another card jump to a different column.
 */
export function dealIntoColumns<T>(items: readonly T[], count: number): T[][] {
  const columns: T[][] = Array.from({ length: Math.max(1, count) }, () => [])
  items.forEach((item, index) => {
    columns[index % columns.length]?.push(item)
  })
  return columns
}
