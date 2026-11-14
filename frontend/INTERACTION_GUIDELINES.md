# Frontend interaction guidelines

## Design interactions from first principles

Start with the user’s goal, the object they are acting on, the information they
need, and their likely next action. Choose controls and layout from those needs
rather than copying a familiar component or exposing the implementation model.

- Put actions on the objects they affect and show the result where the action
  was triggered. Keep validation, request errors, loading, and completion
  feedback beside the submitting control or inside its active panel. A toast
  must not be the only feedback for an operation inside a panel.
- Prefer direct, reversible manipulation. For example, remove an overview card
  with × on the card; turn its vacant slot into an Add card entry point listing
  only cards not already included. Preserve the draft until Save or Cancel.
- Reduce navigation, memory burden, and unnecessary confirmations. Preserve
  spatial context, keyboard focus, and layout while showing progress or errors.
- Validate the complete flow: discovery, action, visible result, failure, retry,
  cancellation, and recovery, including keyboard and narrow-screen use.

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

## Size form fields for their content

- Base columns on the available panel width, not the browser viewport. A wide
  page can still contain a narrow popover or nested editor.
- Group related settings. Keep ports, priorities, durations, and tolerances
  compact; give names and selectors moderate width and URLs, paths, lists,
  and structured editors full rows when needed.
- Keep toggle/interval pairs together. Allow descriptions to wrap independently
  of the input width, without squeezing the input or its adjacent label.
- Use shared Input fieldSize hints and schema field layouts before adding local
  width overrides. Keep existing values, validation, and keyboard order intact.
- Check desktop and phone panel widths. Long labels, selected options, matcher
  values, and helper text must not force horizontal scrolling. Vertical scrolling
  remains appropriate for long forms; keep submit controls reachable.

- Prefer short, valid examples for familiar input formats instead of explaining
  their syntax in prose (for example, `1h, 24h, 7d, 2w (weeks)`). Keep important
  behavioral consequences visible; report constraints beside the field when
  validation fails. Do not repeat information already clear from its label.

- Put additional field explanations behind an information icon beside the label.
  Reveal help on hover, keyboard focus, or tap, and support Escape/outside
  dismissal. Keep errors and essential consequences visible. Shared Input hints
  use this pattern by default; use inline hints only for brief examples that
  need to remain visible.

## Reveal dependent fields when their option is enabled

Show fields only when their controlling choice makes them relevant. For example,
Auto-update reveals Update Interval beside it. Preserve its draft value when
disabled and re-enabled; skip irrelevant validation and do not let hidden values
activate disabled behavior. Keep the controller and dependent fields together,
with natural keyboard order and no automatic focus jump.
