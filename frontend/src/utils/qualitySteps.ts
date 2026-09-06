/**
 * Maps a subscription's reachable/total node counts onto per-block colours for
 * `SegmentedProgress`.
 *
 * This is composition, not progress: every block is filled, and the colours say
 * how the node population splits between working and not. The component is
 * given `percent="100"` so it renders them all.
 */

/** Tailwind classes, so the blocks follow the app's dark-mode tokens. */
export const GREEN_STEP = 'bg-green-500'
export const RED_STEP = 'bg-red-500'

/**
 * Returns one colour per block, green first.
 *
 * The rounding rule is the whole point, and it is deliberately NOT the
 * arithmetic one. On five blocks, 36 of 37 nodes is 97.3%, which rounds to five
 * green — a subscription with a dead node drawn as flawless. A health gauge
 * that can round a failure away is worse than no gauge, because it is trusted.
 *
 * So the green count FLOORS, and both minorities are clamped into view:
 *
 *   - any failure keeps at least one block red (36/37 → 4 green, 1 red);
 *   - any success keeps at least one block green (1/37 → 1 green, 4 red).
 *
 * The cost is that the blocks overstate a small minority — one dead node in 37
 * paints a fifth of the bar. That is accepted: the exact figure is always
 * rendered next to the blocks ("97% (36/37)"), so the bar is the glance and the
 * text is the truth. Losing a failure entirely has no such backstop.
 *
 * An empty array means "nothing was measured" — not "everything failed". The
 * caller renders the trail, matching the backend, which refuses to record a
 * sample it could not take.
 */
export function qualityStepColors(reachable: number, total: number, steps = 5): string[] {
  if (!Number.isFinite(total) || total <= 0 || !Number.isFinite(steps) || steps <= 0) {
    return []
  }

  const up = Math.min(Math.max(reachable, 0), total)
  if (up >= total) return Array.from({ length: steps }, () => GREEN_STEP)
  if (up <= 0) return Array.from({ length: steps }, () => RED_STEP)

  const green = Math.min(steps - 1, Math.max(1, Math.floor((steps * up) / total)))
  return Array.from({ length: steps }, (_, i) => (i < green ? GREEN_STEP : RED_STEP))
}
