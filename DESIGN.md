---
name: mroki-hub
description: Safe shadow traffic testing for production systems.
colors:
  instrument-zinc-bg: "#ffffff"
  instrument-zinc-fg: "#09090b"
  instrument-zinc-muted: "#71717a"
  instrument-zinc-border: "#e4e4e7"
  instrument-zinc-accent: "#f4f4f5"
  primary: "#18181b"
  primary-foreground: "#fafafa"
  signal-green: "#22c55e"
  signal-blue: "#3b82f6"
  signal-amber: "#f59e0b"
  signal-red: "#ef4444"
  syntax-key: "#0369a1"
  syntax-string: "#6d28d9"
  syntax-number: "#047857"
  syntax-boolean: "#c2410c"
  syntax-null: "#be185d"
  syntax-punctuation: "#71717a"
typography:
  body:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, Roboto, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 500
    letterSpacing: "0.1em"
  mono:
    fontFamily: "SF Mono, Menlo, Monaco, Consolas, monospace"
    fontSize: "0.75rem"
    fontWeight: 400
    lineHeight: 1.5
rounded:
  sm: "0.375rem"
  md: "0.5rem"
  lg: "0.625rem"
  xl: "0.75rem"
  full: "9999px"
spacing:
  xs: "0.5rem"
  sm: "0.75rem"
  md: "1.25rem"
  lg: "2rem"
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.primary-foreground}"
    rounded: "{rounded.md}"
    padding: "0.5rem 1rem"
  button-ghost:
    backgroundColor: "{colors.instrument-zinc-accent}"
    textColor: "{colors.instrument-zinc-fg}"
    rounded: "{rounded.md}"
  card:
    backgroundColor: "{colors.instrument-zinc-bg}"
    textColor: "{colors.instrument-zinc-fg}"
    rounded: "{rounded.xl}"
    padding: "1.25rem"
  input:
    backgroundColor: "{colors.instrument-zinc-bg}"
    textColor: "{colors.instrument-zinc-fg}"
    rounded: "{rounded.md}"
    height: "2.25rem"
---

# Design System: mroki-hub

## Overview

**Creative North Star: "The Control Room"**

mroki-hub is a calm, dark-capable operations dashboard for engineers deciding
whether a shadow service behaves like production. It is dense but legible:
every pixel earns its place by carrying signal, and decoration is treated as
noise. The interface is built almost entirely from a near-monochrome zinc
foundation so that the moment a semantic color appears — a red diff rate, a
tinted method badge, a highlighted JSON key — it *means* something. Color is a
readout, not a mood.

The system is quiet and restrained by design. Chrome recedes; data (URLs,
paths, status codes, JSON) is set in monospace and given the foreground. It is
unmistakably a developer tool — keyboard-operable, information-first — but a
composed one, closer to instrumentation than to a terminal. Depth is nearly
flat: surfaces are separated by hairline borders and subtle tonal fills rather
than shadow.

**Key Characteristics:**

- Near-monochrome zinc base; semantic color reserved for actual signal.
- Flat surfaces, hairline borders, tonal layering instead of elevation.
- Monospace for all machine data; system sans for chrome and prose.
- Dark-only: the hub ships a single dark palette (the `.dark` values are the
  canonical, and only reachable, theme); no light mode or theme toggle.
- Color never carries meaning alone; it accompanies a number or label.

## Colors

A near-monochrome zinc field punctuated by a small, disciplined family of
semantic signals.

### Primary
- **Ink** (`#fafafa` in the dark palette): the primary action color — primary
  buttons and high-emphasis marks. A light seed value (`#18181b`) is retained in
  the token but is not reachable while the hub renders dark-only.

### Secondary
- **Signal Green** (`#22c55e`): success and "no diff" / identical states, and the
  healthy end of the diff-rate scale.
- **Signal Blue** (`#3b82f6`): informational and "live" affordances (e.g. the live
  side of a gate, GET method badges).
- **Signal Amber** (`#f59e0b`): warning and the mid band of the diff-rate scale
  (1–10%).
- **Signal Red** (`#ef4444`): danger, destructive actions, the presence of a diff,
  and the high end of the diff-rate scale (≥10%).

### Neutral
- **Instrument Zinc** (`#fafafa` → `#71717a` → `#09090b`): the calm grey the whole
  control room is built from — backgrounds, cards, foreground text, muted/dim
  secondary text, borders, and dividers. Drives nearly every surface.

### Syntax
- **JSON palette** (key `#38bdf8`, string `#a78bfa`, number `#34d399`, boolean
  `#fb923c`, null `#f472b6`, punctuation zinc): a dedicated palette for the diff
  viewer only, composed for legibility against the dark surface. Light seed
  values (key `#0369a1`, string `#6d28d9`, number `#047857`, boolean `#c2410c`,
  null `#be185d`) are retained in the tokens but are not currently reachable.

### Named Rules
**The Signal-Sparingly Rule.** The zinc foundation carries the interface;
semantic accents mark only attention — status, method, diff. If a color is not
reporting a fact, it does not appear.

**The Color-Is-Never-Alone Rule.** Color is emphasis, never the sole signal. The
diff rate always shows its number; status always shows its code. Remove the
color and the meaning must survive.

## Typography

**Body / Chrome Font:** system-ui stack (`-apple-system`, `Segoe UI`, `Roboto`, sans-serif)
**Data / Mono Font:** `SF Mono` (with `Menlo`, `Monaco`, `Consolas` fallbacks)

**Character:** Two registers, cleanly separated. The native system sans handles
all interface chrome — it disappears into the OS and keeps the focus on content.
Monospace handles everything the machine produced — URLs, paths, methods, status
codes, latencies, JSON — so data reads as data.

### Hierarchy
- **Title** (600, `1rem`–`1.125rem`): page and card headings; sparse.
- **Body** (400, `0.875rem`, 1.5): default interface text and prose.
- **Label** (500, `0.75rem`, `0.1em` tracking, often uppercase): metadata captions
  and section eyebrows (e.g. the `LIVE` / `SHADOW` gate captions).
- **Mono** (400, `0.75rem`): all machine data — the workhorse of the diff viewer,
  request rows, and gate URLs.

### Named Rules
**The Monospace-For-Data Rule.** If a value came from the wire — a URL, path,
method, status, latency, or JSON — it is set in monospace. Prose and chrome are
never monospace; data never is not.

## Layout

A single-column, max-width content shell centered on the page, driven by a
consistent spacing rhythm (`0.5 / 0.75 / 1.25 / 2rem`). Lists (gates, requests)
are full-width stacked rows separated by borders rather than gaps. Density is
high but banded: related metadata is grouped tightly, distinct groups separated
generously. Responsive behavior is mobile-first with a primary `sm` (640px)
breakpoint — rows that read as a single aligned line on desktop stack into
primary-content-first cards on phones (method + path first, metadata reflowing
below), so the scan target is never crushed by fixed-width metadata.

## Elevation & Depth

Nearly flat. Depth is conveyed by hairline borders (`instrument-zinc-border`) and
subtle tonal background shifts (`bg-card`, `bg-accent`, `bg-background/60`), not by
ambient shadow. The only shadow in the system is a minimal functional `shadow-xs`
on interactive controls — inputs and outline buttons — to seat them as
manipulable objects. Cards, lists, and panels never cast shadow at rest.

### Named Rules
**The Flat-By-Default Rule.** Surfaces are flat. A border or a tonal fill
separates layers; a shadow appears only on an interactive control (input, outline
button) to signal it can be manipulated — never for decoration or to lift a card.

## Shapes

Softly rounded, consistent, never pill-by-default. The radius scale derives from a
`0.625rem` base: `sm 0.375rem` (inputs, small controls, code chips), `md 0.5rem`
(buttons, inputs), `lg 0.625rem` (icon tiles), `xl 0.75rem` (cards). Fully round
(`9999px`) is reserved for status badges and the small state dots. Borders are
uniformly 1px in a zinc tone; there are no heavy outlines or double borders.

## Components

### Buttons
- **Shape:** rounded (`md`, `0.5rem`); default height `2.25rem`, `0.5rem 1rem` padding.
- **Primary:** Ink background with inverse foreground; `hover` darkens to 90%.
- **Ghost / Secondary:** transparent or zinc-accent fill; `hover` shifts to accent.
- **Focus:** a `3px` ring in `ring` with a border shift — always visible, keyboard-first.

### Badges / Chips
- **Shape:** fully round (`full`), `0.75rem` text, `0.5rem` horizontal padding.
- **Status:** tinted at low alpha over a semantic signal (e.g. `bg-danger/10 text-danger`
  for a present diff, `bg-success/10 text-success` for none), paired with a dot.
- **Method (signature):** HTTP verb tinted from a fixed map — GET = Signal Blue,
  POST = Signal Green, others fall back to muted — in a `w-14` monospace pill.

### Cards / Containers
- **Corner:** `xl` (`0.75rem`). **Background:** `bg-card`. **Border:** 1px zinc.
- **Shadow:** none at rest (see The Flat-By-Default Rule); `hover` shifts border and
  background rather than lifting. **Padding:** `1.25rem` (`md`).

### Inputs / Fields
- **Style:** transparent fill, 1px zinc border, `md` radius, `shadow-xs`, `2.25rem` tall.
- **Focus:** border shifts to `ring` with a `3px` ring; no glow.
- **Invalid:** `aria-invalid` drives a destructive ring and border.

### Diff Viewer (signature)
The defining surface: a monospace, side-by-side JSON diff. Removed lines carry a
`line-through` in a semantic tone; added/changed lines are tinted; the six-token
dark syntax palette colors keys, strings, numbers, booleans, nulls, and
punctuation. The header is deliberately spare — format badge plus the three real
controls (Wrap, View mode, Config); no redundant legend or counts.

### Diff Rate (signature)
A value-driven readout that colors itself green below 1%, amber from 1–10%, and
red at or above 10% — but always alongside the numeric value (The
Color-Is-Never-Alone Rule).

## Do's and Don'ts

### Do:
- **Do** build from Instrument Zinc first and add a semantic signal only when it
  reports a fact (status, method, diff).
- **Do** set every wire-derived value (URL, path, method, status, latency, JSON) in
  monospace.
- **Do** keep surfaces flat — separate layers with a 1px zinc border or a tonal
  fill, and reserve `shadow-xs` for interactive controls.
- **Do** pair every color signal with a number or label so meaning survives without color.
- **Do** provide a visible `3px` focus ring and full keyboard operability on every control.
- **Do** define new tokens in the dark palette (`.dark`); a light seed value is
  optional, since the hub renders dark-only.

### Don't:
- **Don't** use a semantic accent for decoration, emphasis, or "brand color" fills.
- **Don't** add ambient shadows to cards, lists, or panels to create depth.
- **Don't** set interface chrome or prose in monospace, or body data in the sans stack.
- **Don't** introduce a raw Tailwind palette swatch (`text-sky-400`) — use the
  semantic or syntax tokens so both themes stay correct.
- **Don't** convey state with color alone (no bare red dot without its value).

---

Co-authored by [Augment Code](https://www.augmentcode.com/?utm_source=git&utm_medium=commit&utm_campaign=hub_design)
