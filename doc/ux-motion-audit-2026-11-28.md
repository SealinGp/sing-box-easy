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

## In-place proxy inspection and collection density (v1.18.0)

- Connections and topology exits share a contextual proxy inspector with the existing proxy-group card, search, latency testing, selection feedback and nested-group navigation. Automatic groups retain their non-manual behavior. Leaf exits show available address, subscription and quality information; phone topology has touch-sized exit shortcuts.
- Keep the selected connection detail after it leaves the bounded history list. Preserve nested overlay ownership and reposition an already-open inspector when another exit is selected.
- Outbound rows identify their owning subscription using the existing ownership suffix and one shared subscription-list request. Unknown subscription names fall back to their ID.
- Reduce list-cell padding. Card mode offers per-page persisted compact/comfortable density (300/560 px minimum card width), including the subscription collection. Wide cards retain wrapped field values. Toolbar placement excludes itself on reload.
- Extract topology and outbound controllers/legend into cohesive modules. Cancel delayed form cleanup on reopen; preserve shared options on type changes; guard duplicate submissions and keep busy forms from closing.
- Browser checks covered successful/failed mock selection, duplicate clicks, automatic groups, nested history/settings overlays, connection termination beyond a 500-item history cap, source labels, card density/selection persistence and reload, mobile dark/reduced-motion popup bounds, topology anchor switching and rapid draft reopening. The mock type change retained server, port and bind interface. No real proxy selection was changed for testing.
- Final checks: 606 frontend tests, production typecheck/build, diff whitespace check and the 500-line limit passed. Independent GPT source review reported no remaining release blockers.

## Logical DNS rule editor (v1.18.1)

- Replace the flat-only condition form with a recursive editor for existing logical DNS rules: AND/OR, inversion, nested conditions/groups and undo removal. Render the actual logical tree in the preview instead of suggesting a catch-all rule.
- Deep-copy drafts and recursively normalize listable fields/query types. Preserve unknown fields, sibling conditions and the root action. Reject empty logical groups and empty ordinary child conditions next to their controls.
- Browser verification: edit the third local logical rule with real picker interaction; remove AAAA, capture a mocked update and verify HTTPS plus the complete sibling rule set and predefined/NOERROR action; simulate failure and retain the draft. Remove a rule-set chip, add an empty nested group and block save, remove/undo the group, cancel and confirm the loaded configuration is unchanged. Check 390px dark/reduced-motion popup bounds and no document overflow.
- 611 frontend tests pass, including five recursive draft regression tests; production build, diff checks and modified-file line limits pass. Independent GPT review found the empty-child edge case and verified its correction. Production DNS policy is not modified by verification.
