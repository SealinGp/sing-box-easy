# Shared subscription quality — TDD evidence

## Source and journey

No plan file was supplied. The journeys were derived from the requested
dashboard interactions:

- As an operator, I want the probe summary itself to open the quality trend.
- As an operator, I want the same quality component on both subscription
  screens, with a detailed default size and the overview's original compact
  size.
- As an operator, I want provider-defined plan details to occupy one line while
  their full contents remain available on hover or keyboard focus.

## Task report

`SubscriptionQualityCell.vue` owns the quality rendering, trend trigger, and
dialog lifecycle. Its default size preserves the stacked Subscriptions-table
layout; `size="small"` recreates the overview's original inline label,
segmented bar, reachable count, and latency. Both are semantic buttons with
the same hover/focus behavior, and neither uses a separate chart icon.

`SubscriptionsOverviewCard.vue` opts into the compact size. Provider-defined
plan extras are flattened into one ellipsized line, and a teleported tooltip
renders every complete entry on hover or focus without being clipped by the
card's scrollport. Provider CR/LF line breaks become visible ellipses.

## Test specification

| What is guaranteed | Test or command | Type | Result |
| --- | --- | --- | --- |
| The shared quality cell owns the trend dialog and its open action | `bun test src/components/SubscriptionsOverviewCard.test.ts` | Template regression | RED before consolidation, then PASS |
| No separate `ChartBarIcon` action remains | `bun test src/components/SubscriptionsOverviewCard.test.ts` | Template regression | RED before removal, then PASS |
| The default and compact quality sizes both exist, and the overview selects compact | `bun test src/components/SubscriptionsOverviewCard.test.ts` | Template regression | RED before size support, then PASS |
| Plan extras stay on one truncated line and expose a full hover/focus tooltip | `bun test src/components/SubscriptionsOverviewCard.test.ts` | Template regression | RED before tooltip support, then PASS |
| Every provider entry is retained in order and multiline text is normalized with ellipses | `bun test src/utils/subscriptionInfo.test.ts` | Unit | RED before `formatPlanExtras`, then 2 PASS |
| Existing frontend behavior remains intact | `bun test --coverage` | Unit suite | 348 passed, 0 failed |
| The Vue template type-checks and bundles | `bun run build` | Production build | PASS |

Coverage was 91.26% of functions and 92.80% of lines overall.

## Checkpoints and known gaps

- RED: `47503b1` (`test: define probe summary trend action`)
- GREEN: `b088512` (`feat: open subscription trend from probe summary`)
- RED: `063de31` (`test: define shared subscription quality interface`)
- GREEN: `13379fc` (`refactor: centralize subscription quality interaction`)
- RED: `c6eb0e2` (`test: define compact probe and plan extras tooltip`)
- GREEN: `fa269d3` (`feat: add compact subscription quality details`)

No browser screenshot run was available in this session. The template contract,
full unit suite, type-check, and production bundle all passed. Vite reported
only its existing large-chunk advisory.
