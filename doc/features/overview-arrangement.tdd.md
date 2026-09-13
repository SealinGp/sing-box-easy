# Overview card arrangement — TDD evidence

## Source and journey

No plan file was supplied. The journey was derived from the requested iOS-style
interaction: an operator can press and hold a non-interactive part of any card
to enter a clear arrangement mode, drag the whole card while its own controls
are inactive, use accessible move buttons as an alternative, and finish or
cancel the session without a permanent page header or arrangement button.

## Evidence

| Guarantee | Test or command | Type | Result |
| --- | --- | --- | --- |
| A card surface arms its item for dragging, while overlaid controls do not | `bun test src/composables/useDragReorder.test.ts` | Unit | RED before implementation (`surfaceAttrs is not a function`), then 3 passed |
| A 500 ms card hold enters arrangement mode and arms that card | `bun test src/composables/useDragReorder.test.ts` | Unit | RED before implementation (`activationAttrs is not a function`), then passed |
| Release, pointer movement beyond 8 px, and interactive controls do not enter arrangement mode | `bun test src/composables/useDragReorder.test.ts` | Unit | Passed |
| Existing live reorder, save, and cancel behavior remains intact | `bun test --coverage` | Unit suite | 334 passed, 0 failed |
| The Vue template, inert card state, icons, and translations type-check and bundle | `bun run build` | Production build | Passed |

Coverage after the change was 92.78% of functions and 94.67% of lines overall;
`useDragReorder.ts` measured 79.59% of functions and 93.81% of lines.

## Checkpoints and known gap

- RED: `f1a2646` (`test: add overview card surface drag reproducer`)
- GREEN: `e66feb1` (`feat: support full-surface overview dragging`)
- Hold RED: `9309090` (`test: define hold-to-arrange behavior`)
- Hold GREEN: `1432115` (`feat: enter overview arrangement on hold`)
- Visual refactor: no normal-state title or Arrange row; arrangement mode keeps
  its sticky Done/Cancel/Reset toolbar, inert card contents, and accessible move
  controls.

The in-app browser had no available connection, so screenshot and pointer-drag
verification remain a manual follow-up; build and interaction-logic checks passed.
