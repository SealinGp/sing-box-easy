# Design System: Sing Box Easy Dashboard

**Project ID:** sing-box-easy-frontend

> This document describes what the code actually does. If you change a token,
> change it here too — a previous version of this file documented pill-shaped
> buttons and a blue brand while the app shipped 10px controls and a violet one,
> and that drift is what made the system feel unpredictable.

---

## 1. The stack

One styling engine, one component library. Nothing else.

| Layer | Choice | Notes |
| --- | --- | --- |
| Engine | **Tailwind CSS v4** | CSS-first config. There is **no `tailwind.config.ts`** — v4 does not auto-load one, so it was dead weight and has been deleted. All configuration lives in `@theme` in `src/style/tokens.css`. |
| Components | **PrimeVue v4 (unstyled) + `src/volt/*`** | `volt` holds hand-written pass-through (PT) themes. Anything with overlay/keyboard behaviour (Select, MultiSelect, Chips, Dialog, Toast) comes from here. |
| App components | **`src/components/*`** | Buttons, inputs, cards, badges, modals — built from tokens, not from a vendor. |

**Deliberately removed** — each provided a second vocabulary for widgets we
already had: `daisyui`, `@headlessui/vue`, `tailwindcss-primeui`, `vue-select`,
`@primeuix/themes`. Do not reintroduce them. If you need a new headless
behaviour, add it to `volt/` on PrimeVue or hand-roll it (see `PopConfirm.vue`).

### Stylesheet layout

`src/style.css` is only an import manifest. Order matters — tokens must load first.

| File | Contains |
| --- | --- |
| `style/tokens.css` | `@theme` tokens + semantic `@utility` definitions. **The source of truth.** |
| `style/base.css` | Document defaults, scrollbars, keyframes |
| `style/glass.css` | `.liquid-app` backdrop, `.liquid-glass`, `.liquid-glass-float` |
| `style/controls.css` | Shared input/textarea/select surface |
| `style/primevue.css` | Teleported volt overlays (panels, toasts) |
| `style/legacy.css` | ⚠️ Deprecated `!important` shims. Shrink, never grow. |

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
which component you reached for.

### Status

| Role | Token | Hex |
| --- | --- | --- |
| Success / running | `--color-success` | `#10b981` |
| Warning / pending | `--color-warning` | `#f59e0b` |
| Danger / stopped | `--color-danger` | `#ef4444` |
| Info / metadata | `--color-info` | `#06b6d4` |

### Surfaces

Dark-first with a functional light mode. Both themes are driven by the same
token names (`--color-bg-*`, `--color-text-*`, `--color-border`), re-declared at
`:root` under `prefers-color-scheme: dark`.

---

## 3. Radius — three tokens, no exceptions

The old system had six competing scales (4px to 24px) and silently redefined
Tailwind's own `--radius-md`/`--radius-lg`, so `rounded-md` rendered at 2.3× the
value its author expected. Now:

| Token | Utility | Value | Use for |
| --- | --- | --- | --- |
| `--radius-control` | `rounded-control` | **10px** | Buttons, inputs, selects, textareas, chips, icon buttons |
| `--radius-surface` | `rounded-surface` | **16px** | Cards, panels, modals, popovers, dropdown overlays |
| `--radius-pill` | `rounded-pill` | `9999px` | Badges, status chips, circular icon-only buttons, avatars, dots |

**Write the semantic utility, not `rounded-lg`.** The numeric t-shirt scale is
still aligned to sane values (`rounded-lg` == control, `rounded-2xl` == surface)
purely so a stray utility degrades gracefully — but new code should not use it.

`--radius-field` is aliased to `--radius-control` because PrimeVue reads that
name for its form controls; without the alias PrimeVue falls back to 4px, which
is what made dropdowns look nearly square next to everything else.

---

## 4. Elevation — two levels

| Token | Utility | Use for |
| --- | --- | --- |
| `--shadow-surface` | `shadow-surface` | Things resting on the page: cards, panels |
| `--shadow-float` | `shadow-float` | Things genuinely above it: dropdowns, modals, toasts, popovers |

`--shadow-focus` is the shared focus ring. The numeric scale (`shadow-sm` …
`shadow-2xl`) is remapped onto these two so a stray utility cannot punch a hard
drop shadow through the glass, but new code should use the semantic names.

Glass surfaces additionally carry `--glass-highlight`, a 1px inset top stroke.

---

## 5. Components

### Buttons

`src/components/Button.vue` is the **only** button. There is no `volt/Button`.

- `variant`: `primary` | `secondary` | `danger` | `success` | `ghost`
- `severity`: PrimeVue-style alias for `variant`, kept for former volt call sites
- `size`: `sm` | `md` | `lg` · plus `loading`, `disabled`, `fullWidth`, `action`, `pill`
- Shape is `rounded-control`. `pill` is for **icon-only** circular buttons, never text.
- `action` drops the resting shadow, for buttons inside toolbars and table rows.

### Inputs & forms

`Input.vue` / `Textarea.vue` own layout, label, hint, and error wiring
(`aria-invalid` + `aria-describedby`); the **visual surface comes from
`style/controls.css`**, so native and PrimeVue controls stay pixel-identical.

Controls use their own `--control-*` fill and border, deliberately tinted *away*
from the card behind them. They must never inherit `--glass-bg-*`: a 0.92-white
fill with a `white/40` border on a white card is why fields were previously
invisible in light mode.

Textareas use `--radius-surface`; every other field uses `--radius-control`.

### Cards, modals & overlays

- `.liquid-glass` — resting surface. `.liquid-glass-float` — floating surface.
- `Modal.vue` is the only modal, backed by `volt/Dialog.vue` (PrimeVue).
  Sizes: `sm` `md` `lg` `wide` `xl` → `max-w-md` `lg` `2xl` `3xl` `4xl`.
- Depth comes from translucency + `backdrop-filter: blur(24px) saturate(1.25)`
  plus a hairline stroke — not from heavy solid cards.

> ⚠️ `.liquid-app div.bg-white` (and `section`/`article`/`aside`/`form`) still
> implicitly captures the glass treatment. This is a **deprecated shim** in
> `glass.css`: wanting a white background should not silently grant a border,
> shadow, and blur. Migrate such elements to `class="liquid-glass"` and delete
> the shim once nothing depends on it.

---

## 6. Typography

- **Family:** system sans stack (`-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, …)
- **Headers:** `font-bold`, `tracking-tight`, high-contrast text
- **Body & labels:** `font-medium` / `font-normal`; secondary copy uses
  `--color-text-secondary`
- **Code & tags:** monospace for identifiers, ports, URLs, and tokens

---

## 7. Layout

- Fixed glass sidebar at `13.75rem` (`w-55`); main content fills the rest.
- Standard gutter: `p-6` / `space-y-6`.
- Responsive from `xs` through `2xl`.

---

## 8. Conventions checklist

Before merging UI work:

- [ ] No `rounded-lg`/`rounded-xl`/`rounded-full` — use `rounded-control` / `rounded-surface` / `rounded-pill`
- [ ] No `shadow-sm`/`shadow-xl` — use `shadow-surface` / `shadow-float`
- [ ] No `violet-*` or hard-coded brand hexes — use `primary-*`
- [ ] No new `daisyui` / `headlessui` / `vue-select` imports
- [ ] Buttons are `<Button>`, not raw `<button>`; fields are `<Input>`/`<Textarea>`
- [ ] Anything added to `style/legacy.css` has a deletion plan
