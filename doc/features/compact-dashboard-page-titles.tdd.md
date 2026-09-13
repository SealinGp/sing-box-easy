# Compact dashboard page titles — TDD evidence

## Source and journey

No plan file was supplied. The journey was derived from the requested layout
cleanup: as an operator, I want content and useful controls to begin at the top
of the dashboard so repeated navigation titles do not consume a separate row.

Subscriptions, Outbounds, and Node Rules receive their heading from the shared
`Outbounds.vue` route shell. Inbounds and Settings rendered their own headings.

## Task report

The shared Outbounds heading and the direct Inbounds and Settings headings were
removed. Inbounds keeps its Add action in a smaller, right-aligned toolbar;
Settings starts directly with its cards. Existing child-page actions, subtitles,
section headings, and active navigation labels are unchanged.

## Test specification

| What is guaranteed | Test or command | Type | Result |
| --- | --- | --- | --- |
| Outbounds, Subscriptions, and Node Rules have no shared page-title row | `bun test src/views/dashboard/pageTitleRows.test.ts` | Template regression | RED on the existing `<h1>`, then PASS |
| Inbounds no longer renders its page-title translation | `bun test src/views/dashboard/pageTitleRows.test.ts` | Template regression | RED before removal, then PASS |
| Settings starts without its page-title translation | `bun test src/views/dashboard/pageTitleRows.test.ts` | Template regression | RED before removal, then PASS |
| Existing frontend behavior remains intact | `bun test --coverage` | Unit suite | 337 passed, 0 failed |
| Vue templates and translations type-check and bundle | `bun run build` | Production build | PASS |

Coverage was 92.78% of functions and 94.67% of lines overall.

## Checkpoints and known gaps

- RED: `e92a160` (`test: capture compact dashboard page headers`)
- GREEN: `393ee20` (`style: remove redundant dashboard page titles`)

No browser screenshot run was available in this session. The template contract,
full unit suite, type-check, and production bundle all passed.
