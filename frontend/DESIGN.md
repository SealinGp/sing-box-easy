# Design System: Sing Box Easy Dashboard
**Project ID:** sing-box-easy-frontend

## 1. Visual Theme & Atmosphere
The dashboard uses a liquid-glass operational theme: translucent surfaces, blurred layered backdrops, bright blue action states, and compact information density. It is dark-first but keeps a functional light mode. Depth comes from semi-transparent panels, `backdrop-filter` blur, soft highlight strokes, and restrained shadows rather than heavy solid cards.

## 2. Color Palette & Roles

### Core Palette
* **Liquid Blue** (`#1575ff`): Primary theme color used for CTAs, active navigation, and focused controls.
* **Liquid Blue Hover** (`#0f63df`): Darker blue variation for hover and pressed states.
* **Deep Glass Background** (`#06080b`): Dark mode foundation with layered cyan, amber, and blue light fields.
* **Glass Surface** (`rgba(...)` via CSS tokens): Secondary containers use translucent fills with blurred backdrops.
* **Mist Background** (`#eef3f8`): Light mode foundation, still layered with subtle grid and glow effects.
* **Highlight Stroke** (`rgba(255,255,255,...)`): One-pixel borders and inset top highlights define edges.

### Functional & Status Colors
* **Emerald Green** (`#10b981`): Service running, operations succeeded, or system active states.
* **Amber Yellow** (`#f59e0b`): Warning messages, intermediate processing states, and alerts.
* **Ruby Red** (`#ef4444`): Destructive actions, system stopped states, and error alerts.
* **Cyan Info** (`#06b6d4`): Info tags and metadata badges.

---

## 3. Typography Rules
* **Font Family**: Default system sans-serif fallback stack (`-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, `Roboto`, `Helvetica Neue`, `Arial`, `sans-serif`) for native, high-performance rendering.
* **Headers**: Heavy weights (`font-bold`, `tracking-tight`) for major headers and section titles, using high contrast text.
* **Body & Labels**: Medium and normal weights (`font-medium`, `font-normal`) with soft secondary colors (`#6b7280` in light, `#cbd5e1` in dark) for secondary copy to maintain visual hierarchy.
* **Code & Tags**: Monospace typography for identifiers, ports, and token representations to indicate system parameters.

---

## 4. Component Stylings

### Buttons
* **Shape**: Pill-shaped command buttons (`rounded-full`) for primary actions and compact toolbar controls.
* **Interactive Behavior**: Smooth 200ms ease-out transitions for hover background, blue glow, and disabled opacity.

### Cards & Containers
* **Shape**: Moderately rounded panels (`1.125rem` to `1.5rem`) to keep dashboards polished but not toy-like.
* **Borders**: Translucent hairline strokes (`--glass-border`, `--glass-border-muted`) with an inset top highlight.
* **Shadows & Layers**: Diffused dark shadows plus `backdrop-filter: blur(24px) saturate(1.25)`.

### Inputs & Forms
* **Shape**: Pill inputs for single-line fields; rounded rectangle textareas for multiline data.
* **Styling**: Glass fills with clear contrast, blue focus rings, and stable heights for dense forms.

---

## 5. Layout Principles
* **Structure**: Fixed glass side navigation bar (`width: 13.75rem` / `w-55`) on the left, with the main content area spanning the remaining width.
* **Gutter & Spacing**: Standardized margins and padding (`p-6` or `space-y-6`) creating a consistent grid alignment.
* **Responsive Breakpoints**: Adapts cleanly from mobile viewports (`xs`) to ultra-wide displays (`2xl`).
