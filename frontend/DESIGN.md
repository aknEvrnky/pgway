# Design System Document

## 1. Overview & Creative North Star
**Creative North Star: The Kinetic Monolith**
This design system rejects the "boxed-in" aesthetic of standard admin templates. Instead, it treats the proxy gateway interface as a living, breathing command center. The vision is one of **high-density precision meets editorial elegance.** We break the generic grid through intentional asymmetry—using heavy-weighted typography offset by vast "breathing zones" of negative space. The goal is to make the developer feel they are not just managing a proxy, but orchestrating a high-performance engine.

---

## 2. Colors: Tonal Architecture
We move away from the "line-based" UI. Depth is created through light and shadow, not strokes.

### The "No-Line" Rule
**Explicit Instruction:** Do not use 1px solid borders to section off the UI.
- Use **Background Shifts:** Place a `surface_container_low` element against a `surface` background.
- Use **Negative Space:** Use the `8` (2rem) or `12` (3rem) spacing tokens to create mental boundaries.

### Color Tokens (Dark Mode Primary)
- **Base Foundation:** `surface` (#0b1326) – The deep, obsidian core.
- **Surface Nesting:**
- `surface_container_lowest`: (#060e20) For background utility areas.
- `surface_container`: (#171f33) The primary "card" layer.
- `surface_container_highest`: (#2d3449) For active/hovered states or elevated modals.
- **Accents:**
- `primary`: (#8ed5ff) Electric Cyan. Use for critical focal points.
- `primary_container`: (#38bdf8) For high-visibility action areas.

### The "Glass & Gradient" Rule
To elevate the "Aura" aesthetic, floating elements (modals, dropdowns) must use a **Glassmorphism** effect:
- **Fill:** `surface_variant` at 60% opacity.
- **Blur:** `backdrop-filter: blur(12px)`.
- **Signature Gradient:** Main CTAs should utilize a subtle linear gradient from `primary` to `primary_container` at a 135-degree angle to provide a "lit from within" tech-glow.

---

## 3. Typography: The Editorial Edge
We use **Inter** not just for legibility, but as a structural element.

- **Display Scale (`display-lg` to `display-sm`):** Reserved for high-level metrics (e.g., total requests/sec). Use tight letter-spacing (-0.02em) to create an authoritative, "instrument-panel" feel.
- **Headline Scale (`headline-lg` to `headline-sm`):** Used for page titles. Pair these with a `label-md` uppercase sub-header to create a "technical manual" hierarchy.
- **Mono-Space Integration:** For all YAML configuration and Log streams, use `JetBrains Mono` or `Inter Tight`. These should always be set against `surface_container_lowest` to simulate a terminal embedded within the glass.

---

## 4. Elevation & Depth: Tonal Layering
Traditional drop shadows are forbidden. We use **Ambient Lifts**.

- **The Layering Principle:**
- Level 0: `surface` (The floor)
- Level 1: `surface_container_low` (Sidebar/Navigation)
- Level 2: `surface_container` (Main Content Cards)
- Level 3: `surface_container_highest` (Interactive Elements)
- **Ambient Shadows:** For floating popovers, use a shadow with a 32px blur, 0% spread, and the color `surface_container_lowest` at 40% opacity. It should feel like a soft glow-void, not a "drop" shadow.
- **Ghost Borders:** If a boundary is required for accessibility, use `outline_variant` at **15% opacity**. It should be felt, not seen.

---

## 5. Components: The Primitive Set

### Buttons
- **Primary:** Gradient-filled (`primary` to `primary_container`). No border. `md` (0.375rem) corner radius.
- **Secondary:** Surface-tinted. Background: `secondary_container` at 20% opacity. Text: `primary`.
- **States:** On hover, increase `surface_bright` exposure by 5%. No heavy transitions; use a crisp 150ms ease-out.

### Data Tables (The "Ghost" Table)
- **Header:** No background color. Use `label-md` in `on_surface_variant` (muted) with a bottom "Ghost Border."
- **Rows:** No horizontal dividers. Use a subtle background shift to `surface_container_low` on hover.
- **Cells:** Use `body-md` for data. Status indicators must use a **Pulse Dot** (a 6px circle with a subtle CSS animation) instead of a large "pill" badge to keep the UI clean.

### Inputs & YAML Editors
- **Fields:** Use `surface_container_low` with a 1px `outline_variant` (10% opacity).
- **Focus State:** Transition the border to 100% opacity `primary` and add a 4px outer "glow" using the `primary` color at 20% opacity.
- **YAML Blocks:** Always wrapped in `xl` (0.75rem) radius containers with `surface_container_lowest` background.

### Navigation (The Kinetic Rail)
- Use a slim sidebar using `surface_container_low`.
- Active states should not use a "box." Instead, use a vertical 2px `primary` line on the far left and shift the icon color to `primary`.

---

## 6. Do's and Don'ts

### Do:
- **Do** embrace vertical whitespace. If it feels like there is "too much" room, you are likely on the right track.
- **Do** use `tertiary` (#ffc176) for "Warning" states to maintain the high-tech, amber-alert aesthetic.
- **Do** use `surface_container_highest` for nested configurations to show a clear "drill-down" hierarchy.

### Don't:
- **Don't** use pure black (#000) or pure white (#fff) for surfaces. It breaks the "Aura" tonal depth.
- **Don't** use standard "Success Green" (#00FF00). Use the system's `active` status tokens which are slightly desaturated to prevent visual vibration.
- **Don't** use dividers between list items. Use the spacing scale `2` (0.5rem) to separate content through proximity.

