# Overview card arrangement — TDD evidence

## Source and journey

No plan file was supplied. The journey was derived from the requested iOS-style
interaction: an operator can enter a clear arrangement mode, drag a whole card
while the card's own controls are inactive, use accessible move buttons as an
alternative, and finish or cancel the session without an always-empty toolbar.

## Evidence

| Guarantee | Test or command | Type | Result |
| --- | --- | --- | --- |
| A card surface arms its item for dragging, while overlaid controls do not | `bun test src/composables/useDragReorder.test.ts` | Unit | RED before implementation (`surfaceAttrs is not a function`), then 3 passed |
| Existing live reorder, save, and cancel behavior remains intact | `bun test --coverage` | Unit suite | 331 passed, 0 failed |
| The Vue template, inert card state, icons, and translations type-check and bundle | `bun run build` | Production build | Passed |

Coverage after the change was 92.97% of functions and 94.68% of lines overall;
`useDragReorder.ts` measured 86.84% of functions and 94.16% of lines.

## Checkpoints and known gap

- RED: `f1a2646` (`test: add overview card surface drag reproducer`)
- GREEN: `e66feb1` (`feat: support full-surface overview dragging`)
- Visual refactor: page-level Arrange/Done mode with inert card contents and
  accessible move controls.

The in-app browser had no available connection, so screenshot and pointer-drag
verification remain a manual follow-up; build and interaction-logic checks passed.
