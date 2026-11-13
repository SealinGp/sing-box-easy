# Frontend interaction guidelines

## Create missing dependencies where they are selected

When operation A depends on an entity B, the selector for B must provide a path
to create B. A user should not need to abandon their current work or discover a
separate management page to satisfy the dependency.

- Place a clearly named action such as **Create DNS group** beside the selector,
  including when search returns no results or there are no available entities.
- Anticipate the next step after an unsuccessful search: searchable dependency
  dropdowns must offer **Create …** as the final action below the options. Keep
  it visible with matches, without matches, and when the list is empty. Present
  it as an action, not a selectable data value.
- Support search filtering by meaningful names (and relevant identifiers).
  Carry the search text into the creation form as an editable suggested name;
  never create an entity just because a search produced no results.
- Dismiss the dropdown before opening the creation popover, anchored to the
  persistent selector. Keep the action keyboard-accessible and restore focus
  on cancellation. Successful creation selects the new item without losing A.
- Prefer a contextual inline editor or anchored panel. A link to the management
  page is appropriate for a complex workflow, provided it preserves the draft
  and offers a clear return to the originating operation.
- Preserve all of A's values, validation state and editing position. Creating B
  must not submit, clear, or silently save A.
- After successful creation, refresh B's choices, select the newly created
  entity, return to A and focus its selector. Do not require a manual refresh.
- Cancelling creation returns to A with its previous selection and draft intact.
- Keep errors beside the relevant fields; retain entered values so users can
  correct them. Prevent duplicate submissions during persistence.
- If permission prevents creation, explain why and offer the available access
  or management path. Do not expose a nonfunctional create button.
- If the user opens the management page rather than quick creation, provide a
  return path and preserve the originating draft there too.
- Honor reduced motion and keyboard focus. Loading feedback must preserve layout.

Example: **Add Traffic Policy → Resolve DNS with → Create DNS group → Save →
return to the traffic policy with the new group selected**. Cancelling must keep
the original conditions, outbound and DNS choice.

Review this round trip on desktop and mobile, including an empty selector,
validation failure, cancellation, successful creation, and keyboard navigation.
Lab prototypes must identify sample data and must not call persistence APIs.

## Make configuration changes inspectable

Configuration editors should offer both a plain-language outcome and a technical
view at the point of editing. Let users switch between them without leaving the
form or losing their draft.

- Update the technical preview from the same draft used for submission; do not
  maintain a separate illustrative JSON example that can drift from the form.
- Show concrete additions and, when supported, changes/removals with before/after
  values. Identify affected sections, ordering, generated references, fallback
  behavior, and reuse decisions. Provide copyable output.
- Distinguish proposed, validated, and applied changes. A generated fragment is
  not a complete config or proof that existing rules will permit the outcome.
  State when live-config merge, ordering, or core validation has not occurred.
- Hide invalid or stale generated output and explain what needs correction.
- Preserve the submitted configuration snapshot with the result when useful for
  review. Avoid exposing unrelated credentials or private configuration fields.
- Keep the technical view optional, accessible by keyboard, and usable on narrow
  screens. Trust comes from accurate, inspectable behavior rather than assurances.
