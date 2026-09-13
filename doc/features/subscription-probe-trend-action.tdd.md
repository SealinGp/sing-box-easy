# Subscription probe trend action — TDD evidence

## Source and journey

No plan file was supplied. The journey was derived from the requested overview
interaction: as an operator, I want to open a subscription's quality trend by
selecting the probe summary itself, without targeting a separate chart icon.

## Task report

`SubscriptionsOverviewCard.vue` now renders the probe summary as a semantic
button. Its full visible area opens the existing quality dialog and provides
hover and focus feedback. The separate `ChartBarIcon` import and button were
removed. Rows without a probe sample still render no trend trigger.

## Test specification

| What is guaranteed | Test or command | Type | Result |
| --- | --- | --- | --- |
| The probe summary itself invokes `openQuality(row.id)` | `bun test src/components/SubscriptionsOverviewCard.test.ts` | Template regression | RED while it was a paragraph, then PASS |
| No separate `ChartBarIcon` action remains | `bun test src/components/SubscriptionsOverviewCard.test.ts` | Template regression | RED before removal, then PASS |
| Existing frontend behavior remains intact | `bun test --coverage` | Unit suite | 339 passed, 0 failed |
| The Vue template type-checks and bundles | `bun run build` | Production build | PASS |

Coverage was 92.78% of functions and 94.67% of lines overall.

## Checkpoints and known gaps

- RED: `47503b1` (`test: define probe summary trend action`)
- GREEN: `b088512` (`feat: open subscription trend from probe summary`)

No browser screenshot run was available in this session. The template contract,
full unit suite, type-check, and production bundle all passed.
