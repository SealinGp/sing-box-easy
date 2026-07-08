# Design System: Sing Box Easy Dashboard
**Project ID:** sing-box-easy-frontend

## 1. Visual Theme & Atmosphere
The dashboard is designed with a modern, clean, and airy visual aesthetic that translates information density into structured clarity. It utilizes a dark-first design language that seamlessly adapts to a light mode. Depth is introduced via semi-transparent frosted-glass overlays (`backdrop-blur-sm`), micro-animations for transitions (`animate-slide-up`, `animate-fade-in`), and subtle color gradients that convey premium visual quality.

## 2. Color Palette & Roles

### Core Palette
* **Vibrant Cobalt Blue** (`#3b82f6`): Primary theme color used for call-to-actions, active navigation highlights, and interactive states.
* **Cobalt Hover Blue** (`#2563eb`): Darker blue variation applied to primary components during cursor hover.
* **Slate Navy Background** (`#0f172a`): Deep slate dark mode primary background, conveying high-end technology and reducing eye strain.
* **Frosted Slate Card** (`#1e293b`): Secondary dark mode container color used for cards, sidebars, and elevated UI blocks.
* **Paper White Background** (`#f9fafb`): Light mode primary body background.
* **Pure White Card** (`#ffffff`): Light mode secondary container color.

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
* **Shape**: Generously rounded corners (`border-radius: 0.75rem` / `rounded-xl` or `border-radius: 9999px` / `rounded-full`).
* **Interactive Behavior**: Smooth 200ms ease-out transitions for scale, hover shadows, and background changes. Disabling states drop opacity and enforce `cursor-not-allowed`.

### Cards & Containers
* **Shape**: Generous rounding (`border-radius: 1rem` / `rounded-2xl`).
* **Borders**: Hairline strokes (`border border-gray-100` / `dark:border-gray-800`).
* **Shadows & Layers**: Light diffused shadows (`var(--shadow-sm)`) lifting to elevated shadows (`var(--shadow-md)`) on mouse hover.

### Inputs & Forms
* **Shape**: Uniform rounding (`border-radius: 0.75rem` / `rounded-xl`).
* **Styling**: Solid background fills (`bg-gray-50` / `dark:bg-slate-800`) with high-contrast text fields, glowing focus rings (`focus:ring-2 focus:ring-violet-500/20`), and clear icon indicators.

---

## 5. Layout Principles
* **Structure**: Fixed side navigation bar (`width: 13.75rem` / `w-55`) on the left, with the main content area spanning the remaining width.
* **Gutter & Spacing**: Standardized margins and padding (`p-6` or `space-y-6`) creating a consistent grid alignment.
* **Responsive Breakpoints**: Adapts cleanly from mobile viewports (`xs`) to ultra-wide displays (`2xl`).
