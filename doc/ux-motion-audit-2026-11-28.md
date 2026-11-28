# UX and motion audit — 2026-11-28

## Changes

- Separate initial loading from saving. Keep settings and drafts mounted, show local results, prevent duplicate requests, and block saving unread configuration after load failure.
- Reveal dependent cache, DNS, logging and subscription fields beside their controller; retain drafts and validate only enabled behavior.
- Keep subscription drafts through failed requests, Escape, scrolling and resize. Reposition contextual forms when their anchor leaves the viewport.
- Share service status with navigation. Restore persisted final-policy selection after failed autosave.
- Support search-picker keyboard navigation, accessible native input attributes, explicit toggle states and mobile configuration-editor actions.
- Preserve newer editor changes when an earlier save completes; confirm reset. Extract the editor controller and navigation-search styles within the 500-line limit.
- Use shared feedback, panel, icon and disclosure timings. Reverse connection-detail folds without removal timers. Respect reduced motion, including third-party and utility animations.
- Memoize unchanged proxy members, skip offscreen painting and hidden-subtree layout, and keep member size estimates consistent with the border box. Bulk group changes are immediate; individual folds animate.

## Verification

- 606 frontend tests passed (62 files); production typecheck/build passed.
- `go test ./...`, `go vet ./...`, `git diff --check` passed.
- All modified/new production files were checked against the 500-line limit.
- Reviewed 26 dashboard routes at 1440 × 900 in light mode and 390 × 844 in dark mode, including document/main overflow and accessible action names. Corrected the mobile editor toolbar overflow.
- Exercised dependent-field draft retention, failed settings saves, subscription double-submit and Escape during a slow request, keyboard picker selection and dismissal, rapid connection-detail reversals, reduced motion, and long-list scrolling/focus.
- Verified subscription draft remains visible after resize/scroll and after simulated anchors move above/below the viewport; no live subscription was created.
- Independent GPT source review found and rechecked load-failure overwrite, duplicate-save, native/custom control disabling, hidden-field validation and overlay dismissal issues. Final review reported no remaining release blockers.

## Performance observations

Browser measurements use requestAnimationFrame intervals with no CPU throttling; they are local lab observations, not guarantees for every device.

- Dataset: 11 proxy groups, roughly 24,700 DOM nodes; repeated 8 toggles at 120 ms intervals.
- Original bulk-toggle baseline: P95 about 222 ms, maximum about 382 ms.
- Final production bulk toggles: P95 about 55–63 ms, maximum about 83–90 ms in warm runs. A cold run included a 577 ms outlier. Bulk operations still have a measurable one-time layout cost; they do not animate all columns simultaneously.
- Final production individual fold with 504 members: P95 7.3 ms, maximum 14.1 ms, no frame over 33 ms in the sampled run.
- Development-mode bulk results were faster and are deliberately not used as production evidence.

## Limits

Mobile coverage used browser viewport/touch emulation, not physical iOS/Android devices. Real configuration changes and network-node switching were not used as UI test fixtures. The existing large-bundle build warning remains. Large simultaneous bulk changes are substantially faster but are not claimed to meet a universal full-frame-rate target.
