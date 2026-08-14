# Design System: Sing Box Easy Dashboard

**Project ID:** sing-box-easy-frontend

> This document describes what the code actually does. If you change a token,
> change it here too — a previous version of this file documented pill-shaped
> buttons and a blue brand while the app shipped 10px controls and a violet one,
> and that drift is what made the system feel unpredictable.
>
> §11 records the drift that exists *right now*, with counts. Prefer fixing a
> row there over adding a new rule above it.

---

## 1. The stack

One styling engine, one component library. Nothing else.

| Layer | Choice | Notes |
| --- | --- | --- |
| Engine | **Tailwind CSS v4** | CSS-first config. There is **no `tailwind.config.ts`** — v4 does not auto-load one, so it was dead weight and has been deleted. All configuration lives in `@theme` in `src/style/tokens.css`. |
| Primitives | **PrimeVue v4 (unstyled) + `src/volt/*`** | `plugins/primevue.ts` registers PrimeVue with `{ unstyled: true }` app-wide, so **PrimeVue ships no CSS at all**. Every pixel comes from the hand-written pass-through (PT) themes in `volt/`. Anything with overlay or keyboard behaviour lives here. |
| App components | **`src/components/*`** | Buttons, inputs, cards, badges, modals, popovers — built from tokens, not from a vendor. |

**Deliberately removed** — each provided a second vocabulary for widgets we
already had: `daisyui`, `@headlessui/vue`, `tailwindcss-primeui`, `vue-select`,
`@primeuix/themes`. Do not reintroduce them. If you need a new headless
behaviour, add it to `volt/` on PrimeVue or hand-roll it (see `PopConfirm.vue`,
which is ~60 lines of toggle + click-outside + Escape and did not justify a
dependency).

### Stylesheet layout

`src/style.css` is only an import manifest. Order matters — tokens must load first.

| File | Lines | Contains |
| --- | --- | --- |
| `style/tokens.css` | 222 | `@theme` tokens + semantic `@utility` definitions. **The source of truth.** |
| `style/base.css` | 106 | Document defaults, scrollbars, keyframes, reduced-motion |
| `style/glass.css` | 119 | `.liquid-app` backdrop, `.liquid-glass`, `.liquid-glass-float` |
| `style/controls.css` | 147 | Shared input / textarea / select / `.volt-select` surface |
| `style/primevue.css` | 122 | Teleported volt overlays (panels, toasts) |
| `style/legacy.css` | 57 | ⚠️ Deprecated `!important` shims. Shrink, never grow. |

> **Tailwind v4 gotcha, load-bearing:** `@theme` blocks are hoisted out of any
> surrounding at-rule and emitted at `:root`. A
> `@media (prefers-color-scheme: dark) { @theme { … } }` block therefore does
> **not** scope to dark mode — it overwrites the light values for everyone. Dark
> overrides must be a plain `:root` block inside the media query. See the header
> of `tokens.css`.

### ⚠️ The teleport rule

**Never scope a rule under `.liquid-app` unless you have checked it is
unreachable from a teleported overlay.**

PrimeVue renders `Dialog` and every overlay panel with `appendTo: 'body'`, so
they mount **outside** `.liquid-app`. Any ancestor-scoped rule silently stops
applying the moment its element appears inside a modal or dropdown. This has
bitten the codebase twice: once for the select panel (documented in
`primevue.css`) and once for the entire control layer, which left every input,
textarea, and native `<select>` inside every dialog with no border, fill, or
radius at all.

Layers are scoped as follows, deliberately:

| Layer | Scoped? | Why |
| --- | --- | --- |
| `controls.css` | **No** | Must reach controls inside dialogs |
| `primevue.css` | **No** | Overlays are teleported to `<body>` |
| `legacy.css` | **No** | Dark-mode text fixes must reach dialog content |
| `glass.css` `.liquid-glass` | No | Explicit opt-in class |
| `glass.css` `bg-white` shim | **Yes — required** | De-scoping would give PrimeVue's internal `bg-white` `<div>`s (MultiSelect's list, header, empty message) a full panel border/shadow/blur *inside* a dropdown |

Because `controls.css` is unscoped, it must exclude DOM we do not own:
`[data-pc-section]` (PrimeVue's own filter/chip inputs, themed by their PT
options) and Monaco's `.inputarea`, which also gets an explicit reset.

---

## 2. Color

### Brand

**Liquid Blue `#1575ff`**, published as a full ramp so shade and opacity
modifiers work: `--color-primary-50` … `--color-primary-950`, with
`--color-primary` aliasing the 600 step.

```
bg-primary-600   text-primary-400   ring-primary-500/20
```

Use the ramp. Do **not** hard-code a Tailwind palette (`violet-*`, `blue-*`) for
brand purposes — the app previously had 450 hard-coded `violet-*` usages against
22 token usages, so the same button rendered two different colors depending on
which component you reached for. That migration is now complete: zero `violet-*`
usages remain outside the explanatory comment in `tokens.css`.

### Status

| Role | Token | Hex |
| --- | --- | --- |
| Success / running | `--color-success` | `#10b981` |
| Warning / pending | `--color-warning` | `#f59e0b` |
| Danger / stopped | `--color-danger` | `#ef4444` |
| Info / metadata | `--color-info` | `#06b6d4` |

**The status tint recipe** — used by `Alert`, `Badge`, and (as `color-mix`) by
toast severities, so the three read as one family:

```
bg-<hue>-500/15   border-<hue>-500/25   text-<hue>-800 dark:text-<hue>-100
```

Alert and Badge spell this with Tailwind's `emerald / amber / red / sky` scales
rather than the four tokens above. That is deliberate for now — the tokens have
no 500-step ramp to take a `/15` opacity modifier — but it means a status colour
has two spellings. See §11.

### Surfaces

Dark-first with a functional light mode. Both themes are driven by the same
token names (`--color-bg-*`, `--color-text-*`, `--color-border`), re-declared at
`:root` under `prefers-color-scheme: dark`. There is no manual theme toggle;
the OS setting is the only input.

---

## 3. Radius — three tokens, no exceptions

The old system had six competing scales (4px to 24px) and silently redefined
Tailwind's own `--radius-md`/`--radius-lg`, so `rounded-md` rendered at 2.3× the
value its author expected. Now:

| Token | Utility | Value | Use for |
| --- | --- | --- | --- |
| `--radius-control` | `rounded-control` | **10px** | Buttons, inputs, selects, textareas*, chips, icon buttons, dropdown *items* |
| `--radius-surface` | `rounded-surface` | **16px** | Cards, panels, modals, popovers, dropdown *panels*, textareas |
| `--radius-pill` | `rounded-pill` | `9999px` | Badges, status chips, nav rows, circular icon-only buttons, avatars, dots |

\* Textareas are the one control on `--radius-surface`: they hold multi-line
data and read as a panel, not a field. `controls.css` overrides them explicitly.

**Write the semantic utility, not `rounded-lg`.** The numeric t-shirt scale is
still aligned to sane values (`rounded-lg` == control, `rounded-2xl` == surface)
purely so a stray utility degrades gracefully — but new code should not use it.

`--radius-field` is aliased to `--radius-control` because PrimeVue reads that
name for its form controls; without the alias PrimeVue falls back to 4px, which
is what made dropdowns look nearly square next to everything else.

Note the one asymmetry worth memorising: **nav rows are pills, buttons are not.**
A sidebar or top-bar entry uses `rounded-pill`; a `<Button>` uses
`rounded-control` unless it is icon-only (`pill` prop).

---

## 4. Elevation — two levels

| Token | Utility | Value (light) | Use for |
| --- | --- | --- | --- |
| `--shadow-surface` | `shadow-surface` | `0 8px 22px rgba(15,23,42,.045)` | Things resting on the page: cards, panels, **the sidebar and top bar** |
| `--shadow-float` | `shadow-float` | `0 12px 28px rgba(15,23,42,.1)` | Things genuinely above it: dropdowns, modals, toasts, popovers |

`--shadow-surface-hover` and `--shadow-float-lg` are the hover/modal-mask steps.
`--shadow-focus` (`0 0 0 3px rgba(21,117,255,.18)`) is the shared focus ring.
The numeric scale (`shadow-sm` … `shadow-2xl`) is remapped onto these two so a
stray utility cannot punch a hard drop shadow through the glass, but new code
should use the semantic names.

The sidebar and top bar sit on the **surface** tier on purpose. They previously
carried a bespoke `0 24px 70px / 0.16`, which is above even `--shadow-float-lg`
(the modal tier) and made the chrome read as a banner hovering over the app. The
dropdowns those bars open keep `shadow-float` — that difference is what earns
the contrast between "chrome" and "thing on top of chrome".

Glass surfaces additionally carry `--glass-highlight`, a 1px inset top stroke,
and `--glass-blur` (`blur(24px) saturate(1.25)`).

---

## 5. App components (`src/components/`)

`components/index.ts` is the barrel and exports exactly nine:
`Button`, `Input`, `Textarea`, `Card`, `Modal`, `Alert`, `Badge`, `Loading`,
`NodeList`. Everything else (`TabNav`, `PopConfirm`, `ConfirmDialog`,
`ChipsField`, `Sidebar`, `Topbar`, …) is imported by path. The barrel's old
`Select` export is gone: every call site now uses the PrimeVue-backed `Select`
from `src/volt`.

### Buttons

`Button.vue` is the **only** button. There is no `volt/Button`.

- `variant`: `primary` | `secondary` | `danger` | `success` | `ghost`
- `severity`: PrimeVue-style alias for `variant`, kept for former volt call sites
- `size`: `sm` (`px-3 py-1.5`) | `md` (`px-4 py-2`) | `lg` (`px-6 py-2.5`)
- plus `loading` (renders a spinner, sets `aria-busy`), `disabled`, `fullWidth`,
  `action`, `pill`, `label`
- Shape is `rounded-control`. `pill` is for **icon-only** circular buttons, never text.
- `action` drops the resting shadow, for buttons inside toolbars and table rows.
  `ghost` drops it too, automatically.

### Inputs & forms

`Input.vue` / `Textarea.vue` own layout, label, hint, and error wiring
(`useId()` → `aria-invalid` + `aria-describedby`); the **visual surface comes
from `style/controls.css`**, so native, volt, and app-level controls stay
pixel-identical. Both default to `fullWidth: true`.

Controls use their own `--control-*` fill and border, deliberately tinted *away*
from the card behind them. They must never inherit `--glass-bg-*`: a 0.92-white
fill with a `white/40` border on a white card is why fields were previously
invisible in light mode.

The invalid state is driven by `[aria-invalid='true']`, not a presentational
class, so the styling follows the same signal a screen reader does.

`ChipsField.vue` is the multi-value field (domains, suffixes, keywords, geosite
codes): a label, the volt `Chips` control, and a hint. The hint is not optional
decoration — a bare chips box gives no clue that Enter commits an entry, which
reads as "the input is broken". It also offers a `removable` + `@remove` pair so
a form can let an operator take a field away again, and call sites only enable
that while the field is empty (removing a filled field would discard data
silently).

### Cards, modals & overlays

- `.liquid-glass` — resting surface. `.liquid-glass-float` — floating surface.
- `Card.vue`: `liquid-glass rounded-surface`, `padding` of `none|sm|md|lg`
  (`p-0` / `p-4` / `p-6` / `p-8`), optional `hoverable`.
- `Modal.vue` is the only modal, backed by `volt/Dialog.vue` (PrimeVue).
  Sizes: `sm` `md` `lg` `wide` `xl` → `max-w-md` `lg` `2xl` `3xl` `4xl`.
  `wide` exists solely because the outbound form was authored between `lg` and
  `xl`. The size is passed as **`class`, not `pt`** — `volt/Dialog.vue` already
  binds `:pt` internally, and `ptViewMerge`'s tailwind-merge lets the caller's
  `max-w-*` beat the theme's default.
- `ConfirmDialog.vue` replaces `window.confirm()` app-wide. Mount it **exactly
  once**, in `App.vue`; it renders the shared state owned by `useConfirm()` and
  resolves that composable's in-flight promise.
- `PopConfirm.vue` is the anchored alternative: a `liquid-glass-float` popover
  that names the row it belongs to. A centre-screen dialog detaches the question
  from its subject — "Delete this rule?" gives you no way to check *which* rule.
  It attaches its document listeners only while open, returns focus to the
  trigger on Escape, and moves focus into the confirm button on open.
- Depth comes from translucency + `backdrop-filter: blur(24px) saturate(1.25)`
  plus a hairline stroke — not from heavy solid cards.

> ⚠️ `.liquid-app div.bg-white` (and `section`/`article`/`aside`/`form`) still
> implicitly captures the glass treatment. This is a **deprecated shim** in
> `glass.css`: wanting a white background should not silently grant a border,
> shadow, and blur. Migrate such elements to `class="liquid-glass"` and delete
> the shim once nothing depends on it. 28 `.vue` files still do; 3 opt in
> explicitly.

---

## 6. The volt layer (`src/volt/`)

Six wrappers, all the same shape: `<PrimeVueComponent unstyled :pt="theme"
:ptOptions="{ mergeProps: ptViewMerge }">` plus a generic slot forwarder. The
`theme` is a plain `const`, not a `ref` — these never change at runtime.

| Wrapper | Wraps | Surface comes from |
| --- | --- | --- |
| `Select` | `primevue/select` | Field: `.volt-select` in `controls.css`. Panel: `.volt-select-panel` in `primevue.css` |
| `MultiSelect` | `primevue/multiselect` | Field: `.volt-select` in `controls.css`. Panel: `.volt-multiselect-panel` in `primevue.css` |
| `Chips` | `primevue/inputchips` | Its own PT classes |
| `Dialog` | `primevue/dialog` | `.liquid-glass-float` |
| `Toast` | `primevue/toast` | `.volt-toast*` in `primevue.css`, severity via `data-p~=` |
| `Timeline` | `primevue/timeline` | PT classes only; the marker is deliberately left to the caller's `#marker` slot, because a step's colour carries meaning |

`ptViewMerge` (in `utils.ts`) is what makes all of this composable: it pulls
`class` out of both the global and component-local PT props, reconciles them
with `twMerge`, and hands the rest to Vue's `mergeProps`. That is why a caller
can pass `class="max-w-md"` to `Modal` and win against the theme.

### Three traps this layer has already sprung

1. **PT section keys are not guessable.** `Chips` themes its entry field through
   `inputItem` / `inputItemField`. An earlier revision used
   `inputToken` / `inputTokenField` — keys from a different PrimeVue line — which
   matched nothing, so the browser painted its own default focus ring *inside*
   our already-focused shell. If a volt control ever looks unstyled, check the
   key names against `node_modules/primevue/<component>/<Component>.vue` before
   anything else.
2. **An empty model value renders the placeholder, not the option label.**
   PrimeVue `Select` treats `''`/`undefined` as "nothing selected". Any dropdown
   whose default entry has `value: ''` must repeat that entry's label in
   `:placeholder`, or the control shows an empty box. `AppUpdateCard`,
   `DialerOptions`, `FinalPolicy` and the Subscriptions form all do this.
3. **Bare `bg-white` inside an overlay is a landmine.** `Select`'s `dropdown` is
   a `<div>`, so a `bg-white` on it matched the `.liquid-app div.bg-white` card
   shim and inherited a full panel shadow on a 32px icon box. `label` is a
   `<span>`, so only one half looked detached. No volt theme uses the bare token.

### Chips

`Chips` sanitises centrally on every update — trim, drop blanks, dedupe, always
emit a **new** array. Pasting `" a.com, b.com"` would otherwise store a chip with
a leading space that silently never matches anything in sing-box. It also
defaults `separator: ','` and `addOnBlur: true`; without the latter, typing an
entry and clicking Save discards it.

Chip pills are deliberately small (`px-2 py-0.5 text-xs rounded-pill`): a DNS
rule can hold a dozen domains, and full-size chips turned the field into a wall
of boxes.

---

## 7. Navigation shell

Two layouts share **one** `menuItems` tree, built in `views/Dashboard.vue`:

- `Sidebar.vue` — everywhere. `w-55` (13.75rem), `m-3 mr-0 rounded-surface`,
  `.liquid-sidebar` glass, collapsible submenus.
- `Topbar.vue` — OpenWrt only. LuCI already owns the left edge of the screen
  there, so a second vertical menu reads as two apps fighting; top nav also gives
  config forms the full width on a small router display. Parents become
  dropdowns instead of collapsible sections.

`Dashboard.vue` picks between them via `useDeployment()`'s `isOpenWrt`, which the
router guard resolves before the view mounts so the correct layout renders on
first paint. Shared chrome — version + update badge, live service dot, signed-in
user, logout — lives in `useNavChrome()`.

### The sliding active pill

The sidebar's active row does **not** paint its own background. Two
absolutely-positioned pills (`.nav-pill-primary`, `.nav-pill-secondary`) sit
behind the list and are moved by `useNavIndicator()`:

- A `linear-gradient()` is **not an interpolatable CSS value**. A per-row
  `background: linear-gradient(...)` → `none` cannot transition; it snaps. One
  persistent element animating `transform` sidesteps that entirely.
- `transform`/`opacity` are compositor-driven, so the slide keeps running while
  the main thread mounts the incoming route — which is exactly when a
  background/size transition would stutter.
- Positions are written straight to the DOM, not through a reactive style
  binding: the `ResizeObserver` fires every frame of a submenu expand, and
  routing that through Vue's render cycle reintroduces the jank.

Rows opt in with `data-nav-pill="primary" | "secondary"`. Only the
route-matching entry gets the gradient — a single pill cannot be in two places,
and painting a merely-expanded parent as "active" was misleading.

`Topbar.vue` has no sliding pill; it paints `.topbar-item-active` on the row
directly, since its items do not move.

Shared easing across the shell: `cubic-bezier(0.32, 0.72, 0, 1)` at **320ms**
for the pill, **300ms** for submenu expansion (a `grid-template-rows: 0fr → 1fr`
animation, because the old `max-height: 0 → 24rem` finished in the first ~40ms
on ~5rem of content and read as a stutter), and **220ms** for the select panel.

---

## 8. Motion

| Where | Duration | Easing |
| --- | --- | --- |
| Nav pill slide | 320ms | `cubic-bezier(.32,.72,0,1)` |
| Submenu expand | 300ms | `cubic-bezier(.32,.72,0,1)` |
| Select panel in / out | 220ms / 150ms | same / `ease-in` |
| MultiSelect panel | none | Appears instantly — see below |
| Dialog in / out | 200ms / 150ms | `ease-out` / `ease-in` |
| Toast in / out | 500ms | height-collapse on leave |
| Topbar dropdown | 150ms / 100ms | `ease-out` / `ease-in` |
| Colour & border transitions | 150–200ms | `ease-out` |

Anything under ~150ms stops reading as motion and just looks like a flicker —
the select panel was 100ms/75ms and was raised for exactly that reason.
`MultiSelect` had the same 100/75 pair and was given **no** transition instead:
its panel is a dense list, and a scale transform visibly reflowed the labels on
the way in. Either fix is valid; what is not valid is leaving a sub-150ms
scale in place. If it ever gets motion back, it needs Select's 220ms.

`prefers-reduced-motion: reduce` is honoured by `base.css` (the four keyframe
utilities), the sidebar (pill + submenu drop to a 120ms opacity fade), and the
topbar dropdown. It is **not** honoured by the volt PT transitions — see §11.

---

## 9. Typography

- **Family:** system sans stack (`-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, …)
- **Headers:** `font-bold`, `tracking-tight`, high-contrast text
- **Body & labels:** `font-medium` / `font-normal`; secondary copy uses
  `--color-text-secondary`
- **Code & tags:** monospace for identifiers, ports, URLs, and tokens

---

## 10. Layout

- Sidebar: `w-55` (13.75rem), inset by `m-3 mr-0`, `rounded-surface`. Main
  content fills the rest. On OpenWrt, `Topbar` takes the top edge with `m-3 mb-0`
  and `relative z-30` (load-bearing: `backdrop-filter` makes the header a
  stacking context, and unpositioned it painted below later page cards, which
  covered its dropdowns; `30` keeps it under the app's `z-50` modals).
- Standard gutter: `p-6` / `space-y-6` / `gap-6`.
- Card grids use **container queries**, not viewport breakpoints, where the
  container is what actually constrains them —
  `@container` + `@3xl:grid-cols-2 @6xl:grid-cols-3` in `Settings.vue`. The
  sidebar/topbar swap changes available width without changing viewport width,
  so viewport breakpoints would lay the same page out wrongly on one of them.
- Responsive from `xs` through `2xl` otherwise.

---

## 11. Known drift

Measured against `src/`. Fix a row here before adding a new rule anywhere above.

| Drift | Count | Notes |
| --- | --- | --- |
| Files relying on the `bg-white` glass shim | 28 `.vue` | vs 3 using `.liquid-glass` explicitly. Blocks deleting the shim in `glass.css`. |
| Files depending on `legacy.css` dark-mode shims | 67 `.vue` | Every file still using raw `text-gray-N00` without a `dark:` variant. |
| Raw `<button>` elements | 99 in 28 files | vs 124 `<Button>`. Many are legitimate bespoke affordances (nav rows, chip "add" buttons, icon toggles); a Button variant for them does not exist yet. |
| Status colours with two spellings | — | Tokens (`--color-success`) vs Tailwind scales (`emerald-500/15`) in Alert/Badge. |
| Volt transitions ignoring reduced-motion | 4 wrappers | PT `transition` classes are not media-guarded. (`MultiSelect` and `Timeline` have no transition at all.) |
| Stray numeric shadow | 1 site | `DnsRuleFlow.vue:155` uses `shadow-sm`. |
| `Loading.vue` has no dark mode | 1 file | `fullScreen` uses `bg-white opacity-90`; text is `text-gray-600`. |

Clean, and worth keeping clean: **zero** native `<select>` elements, **zero**
`violet-*` usages, **zero** stray numeric radius utilities (350 semantic radius
usages).

---

## 12. Conventions checklist

Before merging UI work:

- [ ] No `rounded-lg`/`rounded-xl`/`rounded-full` — use `rounded-control` / `rounded-surface` / `rounded-pill`
- [ ] No `shadow-sm`/`shadow-xl` — use `shadow-surface` / `shadow-float`
- [ ] No `violet-*` or hard-coded brand hexes — use `primary-*`
- [ ] No new `daisyui` / `headlessui` / `vue-select` / native `<select>`
- [ ] Buttons are `<Button>`, fields are `<Input>`/`<Textarea>`, multi-value fields are `<ChipsField>`
- [ ] Dropdowns come from `volt/Select`, and any `value: ''` default entry also sets `:placeholder`
- [ ] New CSS is checked against the teleport rule (§1) before it is scoped
- [ ] Chrome sits on `shadow-surface`; only genuinely floating things use `shadow-float`
- [ ] Anything added to `style/legacy.css` has a deletion plan
