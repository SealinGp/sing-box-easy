# Linked routing and DNS policies

Status: implementation proposal, based on source inspection on 2026-11-13.
No runtime configuration changes have been applied.

## Findings

Source examined: sibling `../sing-box` checkout at `eb57d4f52` (clean working
tree). The locally installed `sing-box` reports 1.14.0; this does not establish
that the installed executable was built from that exact commit. The panel's Go
module dependency is not a substitute for checking installed-core capabilities.

In the inspected `bin/config.json` (indices below are zero-based):

- `route.rules[7]` is a suffix-only route for `a.com` to `♻️ 自动选择`.
  Adding `xx.com` to its suffix list is a structural reuse candidate.
- `dns.rules[11..14]` is an existing domain-suffix race: `GoogleDoT` and
  `dns_google`, with response tags `proxy_domain_dot` and `proxy_domain_google`.
  Both evaluate rules have the same domain list and a **5s** timeout. Both DNS
  servers explicitly use the `♻️ 自动选择` detour. Adding `xx.com` requires
  changing BOTH evaluate lists; the respond rules can remain unchanged.
- `dns.rules[1..6]` is a domestic suffix race across `dns_router`, `aliyun`, and
  `tencent`, with 3s timeouts. These servers do not specify a detour; the UI must
  inspect their effective dial path rather than equate omission with a verified
  geographic guarantee.
- `dns.rules[7..10]` is a separate overseas rule-set race, not the suffix group.
- `dns.rules[21..26]` is an unconditional domestic race and DNS final is `aliyun`.
  Appending an overseas exception after this block can send the query to domestic
  resolvers first. Placement is part of correctness, not just list presentation.
- Reusing the existing 5s group is NOT equivalent to the requested 3s policy.
  Offer to inherit 5s or create a separate 3s group; never silently change other
  domains' timeout.

The source launches evaluate requests asynchronously in
`dns/router.go:launchDNSEvaluate`; matching continues to subsequent evaluates.
Respond rules with `race: true` compete on their referenced responses. A
`NOERROR` match does not guarantee a nonempty address answer or independent
resolver providers (both named servers here use Google). If every response is
unacceptable or fails, subsequent rules/final resolution matter.

## Explicit matching semantics

Do not present `domain_suffix` plus `rule_set` as universally AND or OR.
`route/rule/rule_abstract.go:matchInner` delegates to
`rule_item_rule_set.go:matchWithOuterGroups`. A single default, non-inverted,
non-nested rule-set rule can merge its condition groups with the outer rule:
destination-address alternatives use OR. A multi-rule or logical set is instead
an independent boolean condition combined with outer requirements. Other
constraints, such as ports, still apply.

The rule-set's internal structure can therefore affect matching after a remote
update. The UI should offer explicit "Match any" / "Match all" conditions and
compile mixed suffix/rule-set conditions to logical `or` / `and` subrules. Keep
the action on the parent. For DNS evaluate actions, evaluate remains top-level;
only match conditions belong in logical children. Separate simple terminal
route rules are also suitable when their ordering is intentionally preserved.

The user's sentence describing suffix AND membership already describes an
intersection. If the intention is to add an independent exception, it needs OR.
The UI must make this distinction visible instead of guessing from field names.

## Proposed interaction

Add a shared "Add domain policy" contextual panel to Routing Rules and DNS
Rules, anchored to the triggering button. Keep the existing advanced editors.

1. Enter editable domain text; choose exact domain or suffix. Show that suffix
   includes subdomains. Normalize and validate on submission, without converting
   a single pasted value into a chip before the user can edit it.
2. Choose the traffic outbound.
3. Choose a named DNS profile, such as Overseas or Domestic. A saved outbound
   mapping may preselect a profile, but names/emoji must never infer geography.
4. Show a compact preview: traffic target, DNS servers, transport/detour,
   timeout, failure behavior, reused rules and insertion positions.
5. Save both sections once. Show the resulting policy beside the initiating
   control and link to its route and DNS members.

DNS profiles are panel metadata, not a new sing-box configuration field. Store
stable IDs, a user-defined name, resolver tags, timeout, accepted-response policy,
failure policy, and optional outbound mappings. "Overseas" and "Domestic" are
user classifications, not runtime facts inferred from IPs or names. A selector
may contain mixed paths or change its selected node; expose that uncertainty.

Existing race blocks can be detected and offered for adoption. Detect using the
evaluate/respond dependency graph, complete equal conditions, server/settings
signatures and globally unique response tags, not naming prefixes alone. Do not
silently adopt complex or externally shared response dependencies.

## Safe reuse and failure policy

- Route reuse requires an identical effective action and all other constraints,
  not just the same outbound. Exclude inverted/logical/mixed matcher rules from
  the initial simple suffix merger. Deduplicate normalized suffixes.
- Check priority before suggesting reuse. Earlier rules can shadow the new
  domain; moving an existing rule would affect its existing domains. Preserve
  sniff, DNS interception, LAN exceptions and explicit mode overrides. Report
  unresolved rule-set overlaps; do not claim a complete proof without set data.
- DNS reuse updates every evaluate condition in a recognized group atomically.
  Preserve all action options and response dependencies. Exact timeout, cache,
  response-filter and fallback compatibility are required.
- New groups must precede applicable catch-all blocks. Review earlier DNS rules
  too: a query already dispatched by an earlier evaluate cannot be recalled.
- For a strict overseas policy, default to an explicit domain-scoped failure
  result after the race, before domestic fallback, or use a named overseas-only
  fallback. Let users explicitly choose general fallback when desired and show
  that it can send these queries outside the selected profile. Preserve existing
  domains' failure behavior when reusing a group; a new domain-specific guard
  must not accidentally change the whole group's fallback.
- Check resolver hostname bootstrap (for example `dns.google`), detours and
  dependency cycles. Routing client traffic through a proxy alone does not set
  the DNS transport path. App-managed DNS policy cannot guarantee protection for
  application DNS traffic that bypasses sing-box.

## Implementation boundaries

Existing building blocks:

- `frontend/src/components/RoutingRules.vue` and `DNSRules.vue`: editor entry
  points, contextual panels and installed-core capability checks.
- `frontend/src/utils/parallelResolve.ts`: evaluate/respond generator. Reuse its
  intent, but avoid treating mixed matcher fields as an explicit logical choice.
- `app/pkg/singbox/config/sections/dns.go:AddDNSRulesBatch`: atomic DNS-only
  append; insufficient for ordered insertion plus cross-section reuse.
- `app/pkg/singbox/config/section_document.go:UpdateConfigSections`: existing
  locked multi-section mutation primitive, followed by whole-document validation
  and save. Use this rather than consecutive route and DNS API calls.

Introduce a server-side plan/commit API. Planning reads one config snapshot and
returns proposed edits, conflict/priority findings and a revision hash. Commit
rechecks that revision inside the mutation lock, recomputes the plan, checks
installed-core support for evaluate/respond/race, validates the complete
candidate using the core, and writes both sections together. Preserve unknown
raw fields. Reject stale plans rather than applying old array indices.

Profile metadata and generated-member ownership need explicit recovery semantics:
the config file and panel database do not form one existing atomic transaction.
Use durable operation records/reconciliation or a single authoritative policy
store; never report success with only half of a linked change applied. Detect
manual edits as drift. On policy deletion, remove only owned suffixes/members,
not rules shared with other domains.

## Verification

Existing source tests executed successfully in the sibling checkout:

```sh
go test ./route/rule ./dns -run 'Test(RouteRuleSetMergeDestinationAddressGroup|RuleSetShapeBoundary|DNSRuleSetSemantics|DNSEvaluateParallelFallback|DNS.*Race)' -count=1
```

Implementation tests should cover scalar/array reuse, duplicate domains,
different action options, timeout mismatch, one-vs-multiple-rule set semantics,
shadowed positions, strict fallback with both upstreams failing, stale revisions,
validation failure with no partial writes, profile rename/deletion, drift and
ownership. Browser tests should verify editable paste, both entry points, preview
and cancellation, stable skeleton layout, and a single save updating both views.

References: [route matching](https://sing-box.sagernet.org/configuration/route/rule/)
and [DNS actions](https://sing-box.sagernet.org/configuration/dns/rule_action/).
Source tests verify the inspected checkout; no live DNS traffic or runtime
configuration mutations were performed in this investigation.

## Implemented managed policy workflow

Live pages are `/dashboard/dns/groups` and `/dashboard/route/policies`.
The Lab remains a non-persistent prototype. Both live pages support English and
Chinese, including the label **DNS解析分组**. Creating a group alone changes only
metadata. A policy can use a group, one DNS server, or leave DNS unchanged.

The `trafficpolicy` service owns one SQLite state row, `traffic_policy_state`.
It stores group definitions, policy references, exact generated rule blocks,
and a pending-operation journal. It deliberately does not merge into arbitrary
existing rules: those rules have no proven ownership. Group deletion checks
policy dependencies; replacement rewrites all dependent owned blocks in one
validated config save. Stop managing is a separate metadata-only operation.

Authenticated endpoints (available under both API prefixes):

- `GET /traffic-policies`: metadata, current references, drift and recovery state.
- `POST /traffic-policies/plan`: proposed rule arrays, affected policies, normalized
  command and revision. This does not write configuration or validate with the core.
- `POST /traffic-policies/commit`: recompute under the shared config mutation lock,
  reject stale revision, persist the pending record, validate the complete
  document with the installed core, promote configuration, and finalize metadata.
- `POST /traffic-policies/impact`: review a raw-editor candidate. Its revision is
  sent as `X-Managed-Revision` on `PUT /config`; impact cannot be bypassed by skipping
  the frontend. Nothing is silently regenerated after a manual edit.

Rule positions are explicit zero-based insertion points. Appending is the safe
non-reordering default, but earlier matching rules can shadow a new policy. The
preview includes existing rule order. Policy changes preserve unknown fields in
unrelated sections. Generated DNS blocks launch evaluates first, then responses,
then the optional scoped rejection guard. Single-server responses do not race.

Ownership compares complete canonical contiguous blocks. A duplicated, changed,
partially removed or reordered block cannot be deleted automatically. External
`match_response` references also block owned-block deletion. Compatible adoption
accepts DNS server/timeout edits only when the whole regenerated block matches;
it updates other policies sharing that group's settings in the same plan.
Arbitrary edits can be retained by stopping management. Fully missing policies
can be explicitly restored; modified policies are never blindly overwritten.

Reconciliation is computed against the current file on every list, plan and
commit, so external edits and history rollback are detected without a watcher.
Missing DNS servers, outbounds and rule sets are reported separately. Metadata
is retained when generated rules disappear. A pending operation is recovered by
comparing before/after config hashes; if neither matches, the panel stays
available and offers an explicit metadata-generation choice while preserving the
current file. Unresolved operations block other writes, including rollback.

Configuration saves do not restart sing-box. The UI states that runtime activation
is a separate action in Settings. Process-interruption recovery is tested; no
claim is made here about transactional atomicity across SQLite and a filesystem.

### End-to-end validation (2026-11-13)

Verified against the local development UI and real API/core using temporary
`.invalid` domain policies, without restarting sing-box:

- Search with no matching DNS group, keyboard Tab/Enter into Create, wait for
  dropdown exit, then save the group and retain the policy draft.
- Preview and save a policy; compare persisted members with `config.json`.
- Change only the test policy's DNS timeout through the Monaco editor; review
  the save impact, observe drift, and adopt compatible changes.
- Refuse referenced-group deletion without replacement. Create a replacement
  at that selector, cancel and return without losing context, then create/save
  and replace all dependencies in one operation.
- Remove generated rules manually, keep metadata marked missing, then explicitly
  restore them. Delete only the exact owned rules afterward.
- Select a single DNS server and verify its response has no `race` flag.
- Review the 390px layout and Chinese labels; verify no browser exceptions.

Temporary managed records were removed and the resulting configuration was
compared structurally with its original snapshot. Screenshots are local ignored
artifacts under `bin/managed-policy-review/`. Go tests cover dependency checks,
validation failures, stale revisions, interruption recovery, compatible adoption,
external references, manual drift, dependency removal and metadata-only detach.
