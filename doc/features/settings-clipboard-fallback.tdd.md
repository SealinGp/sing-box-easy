# Settings clipboard fallback — TDD evidence

## Source and journey

No plan file was supplied. The journey was derived from the reported Settings
error: as an operator opening the dashboard over a LAN HTTP address, I want the
prepared update command to copy without the browser's secure-context Clipboard
API being a hard dependency.

The failing path is rendered by `AppUpdateCard.vue` inside Settings. It called
`navigator.clipboard.writeText` directly, so a missing API or a permission
rejection immediately produced the “无法写入剪贴板” notification.

## Task report

A shared clipboard helper now tries the modern Clipboard API first and falls
back to a temporary selectable textarea plus `document.execCommand('copy')`
when the API is missing or rejects. Focus is restored after the fallback. The
Settings update command, reusable copy icon, and GitHub device-code action use
the same behavior. The existing error notification remains as recovery guidance
when both methods fail.

## Test specification

| What is guaranteed | Test or command | Type | Result |
| --- | --- | --- | --- |
| A successful Clipboard API write does not invoke the fallback | `bun test src/utils/clipboard.test.ts src/components/AppUpdateCard.test.ts` | Unit | RED before helper existed, then PASS |
| Missing Clipboard API uses the LAN-HTTP fallback | Same focused command | Unit | RED before helper existed, then PASS |
| A rejected Clipboard API write also uses the fallback | Same focused command | Regression | RED before helper existed, then PASS |
| Failure is reported only when both methods fail | Same focused command | Unit | RED before helper existed, then PASS |
| Settings delegates update-command copying to the shared helper | Same focused command | Source contract | RED on the direct API call, then PASS |
| Existing frontend behavior remains intact | `bun test --coverage` | Unit suite | 357 passed, 0 failed |
| Vue and browser types check and the production bundle builds | `bun run build` | Production build | PASS |

Coverage was 90.22% of functions and 91.39% of lines overall.

## Checkpoints and known gaps

- RED: `fbc974c` (`test: reproduce clipboard fallback failure`)
- GREEN: `24d644f` (`fix: fall back when clipboard access is denied`)

No deployed-router browser session was available in this run. The denied-API
condition was reproduced deterministically through the shared clipboard seam;
the focused regression, full suite, type-check, and production bundle passed.
