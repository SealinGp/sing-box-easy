import type { RouteTopology } from './routeTopology'
import type { FlowOverlay } from '../utils/flowOverlay'

export interface RouteFlowDiagramProps {
  topology: RouteTopology
  /**
   * The live overlay, when the card is watching real traffic. Null keeps the
   * drawing static. The picture never changes shape between the two — the
   * overlay only lights, widens and animates what is already there, so an
   * operator toggling Live compares actual against expected on ONE diagram.
   */
  live?: FlowOverlay | null
  /**
   * Fill the container's HEIGHT, keeping the aspect ratio, instead of
   * rendering at the canvas's natural size. Used full-window, where the goal is
   * the whole ladder on screen at once; letterboxing on a wide screen is fine,
   * a scrollbar is not.
   */
  fit?: boolean
  /**
   * An explicit zoom factor, overriding both default sizings.
   *
   * `null`/undefined keeps them: fit-to-width docked, fit-to-height with `fit`.
   * Both of those SHRINK a 1290px canvas to whatever room there is, which is
   * the right first look and unreadable once the question is what one rule
   * says. A number here renders the same `viewBox` at `scale` and lets the
   * wrapper scroll — see `composables/useDiagramZoom.ts`.
   */
  scale?: number | null
  /**
   * Rules to fold into collapsed bands instead of drawing — the card's "busy
   * only" mode. A band is clickable and emits `expand-band`, so the fold is
   * never a dead end.
   */
  hiddenRuleIndices?: ReadonlySet<number>
}
