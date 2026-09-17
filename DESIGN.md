---
name: Alert Agent
description: A signal-routing panel for alert scenarios — achromatic drafting ground, hairline rules, engraved labels, one instrument-teal signal.
colors:
  ground: "#eef1f0"
  panel: "#f7f9f8"
  panel-sunk: "#e7ebea"
  ink: "#141a19"
  ink-muted: "#414b49"
  ink-faint: "#56625f"
  rule: "#c5cfcc"
  rule-strong: "#9fada9"
  rule-faint: "#dae1df"
  signal: "#0d6f78"
  signal-strong: "#0a565d"
  signal-edge: "#12939e"
  signal-wash: "#d9e9ea"
  mark-critical: "#a5271c"
  mark-high: "#8a5a09"
  mark-normal: "#55625f"
  mark-ok: "#1c6b43"
  danger: "#a5271c"
  danger-wash: "#f6e4e1"
typography:
  display:
    fontFamily: "Overpass, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "1.75rem"
    fontWeight: 600
    lineHeight: 1.15
    letterSpacing: "-0.01em"
  title:
    fontFamily: "Overpass, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "1.0625rem"
    fontWeight: 600
    lineHeight: 1.3
    letterSpacing: "-0.01em"
  body:
    fontFamily: "Overpass, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "normal"
    fontVariation: "tabular-nums"
  small:
    fontFamily: "Overpass, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.5
    letterSpacing: "normal"
  label:
    fontFamily: "Overpass Mono, ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas, monospace"
    fontSize: "0.6875rem"
    fontWeight: 600
    lineHeight: 1.2
    letterSpacing: "0.14em"
  mono:
    fontFamily: "Overpass Mono, ui-monospace, SFMono-Regular, SF Mono, Menlo, Consolas, monospace"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.55
    letterSpacing: "normal"
rounded:
  hairline: "2px"
  pill: "999px"
spacing:
  s-1: "0.25rem"
  s-2: "0.5rem"
  s-3: "0.75rem"
  s-4: "1rem"
  s-5: "1.5rem"
  s-6: "2rem"
  s-7: "3rem"
components:
  button:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    typography: "{typography.small}"
    rounded: "{rounded.hairline}"
    padding: "0.4rem 0.85rem"
    height: "34px"
  button-hover:
    backgroundColor: "{colors.panel-sunk}"
    textColor: "{colors.ink}"
  button-primary:
    backgroundColor: "{colors.signal}"
    textColor: "{colors.panel}"
    rounded: "{rounded.hairline}"
    padding: "0.4rem 0.85rem"
    height: "34px"
  button-primary-hover:
    backgroundColor: "{colors.signal-strong}"
    textColor: "{colors.panel}"
  button-danger:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.danger}"
    rounded: "{rounded.hairline}"
  button-danger-hover:
    backgroundColor: "{colors.danger-wash}"
    textColor: "{colors.danger}"
  button-sm:
    typography: "{typography.label}"
    padding: "0.2rem 0.6rem"
    height: "28px"
  icon-button:
    backgroundColor: "transparent"
    textColor: "{colors.ink-muted}"
    rounded: "{rounded.hairline}"
    width: "32px"
    height: "32px"
  input:
    backgroundColor: "{colors.panel-sunk}"
    textColor: "{colors.ink}"
    typography: "{typography.small}"
    rounded: "{rounded.hairline}"
    padding: "0.4rem 0.6rem"
  chip:
    backgroundColor: "{colors.panel-sunk}"
    textColor: "{colors.ink-muted}"
    typography: "{typography.label}"
    rounded: "{rounded.hairline}"
    padding: "1px 6px"
  chip-open:
    backgroundColor: "transparent"
    textColor: "{colors.ink-faint}"
  chip-stale:
    backgroundColor: "{colors.panel-sunk}"
    textColor: "{colors.mark-high}"
  mark:
    backgroundColor: "transparent"
    textColor: "{colors.mark-normal}"
    typography: "{typography.label}"
  engraved:
    backgroundColor: "transparent"
    textColor: "{colors.ink-faint}"
    typography: "{typography.label}"
  panel-surface:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    rounded: "{rounded.hairline}"
    padding: "1rem"
  panel-head:
    backgroundColor: "{colors.panel-sunk}"
    textColor: "{colors.ink}"
    padding: "0.75rem 1rem"
  rank-node:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink-muted}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    size: "22px"
  dialog:
    backgroundColor: "{colors.panel}"
    textColor: "{colors.ink}"
    rounded: "{rounded.hairline}"
    width: "min(680px, calc(100vw - 2rem))"
---

# Design System: Alert Agent

## Overview

**Creative North Star: "The Silkscreened Instrument Panel"**

Alert Agent's control room is drawn, not decorated. The ground is drafting film — cool
achromatic paper in light, ink-slate in dark — ruled by a faint 88px grid. Everything above
it is built from 1px hairlines that behave like schematic lines: they divide, they route,
they terminate. Content sits in a flat plane; a scenario is not a card in a stack but a
path traced across the full width of the sheet, from what it catches, through how it
investigates, to where it delivers. The mode is Operate: an SRE at a desktop, Grafana in
the next tab, reading configuration as logic rather than as a form.

Colour is rationed. The content layer is achromatic top to bottom; colour appears only on
edges, on the fixed classification marks, and on the single primary action. One signal hue
— instrument teal — means *live*: the current route in the header, the path a probe just
traced, the focused field, the primary button. The classification set (critical / high /
normal / ok) is a separate, fixed vocabulary of engraved marks that is never allowed to
become a fill.

The build refuses the category default it was drawn against: no dark-SaaS sidebar, no
rounded card stacks, no gradient fills, no decorative shadows, no blue accent, no Inter.
Density is expert-grade — 34px controls, 0.9375rem body, tabular numerals everywhere — and
both themes are first-class, stored per browser with an OS-preference fallback.

**Key Characteristics:**

- Achromatic drafting ground with a deliberate hairline grid (88px, 56px under 720px).
- 1px rules as the primary structural device; 2px radius is effectively a square corner.
- Engraved uppercase Overpass Mono micro-labels for every field name and readout.
- One signal hue (instrument teal) on edges, marks and the primary action only.
- Classification as an engraved mark set, never a coloured pill.
- Material state: seated, engraved, flattened — controls state themselves without colour.
- Self-hosted variable Overpass / Overpass Mono, embedded in the binary; no CDN.
- Light and dark both normative; every text token clears WCAG AA in both.

## Colors

A cool achromatic palette with a single teal signal and a fixed four-step classification
set; the frontmatter carries the light theme, which is canonical.

### Primary

- **Instrument Teal** (`signal`): the live-state hue. The primary action, the active nav
  seat, the focus ring, the caret, the traced path's left edge and rank node, the rail's
  active step, the checked switch. `signal-edge` is the brighter variant used on 1px and
  2px edges where the fill hue would read too dark; `signal-strong` is the pressed/hover
  fill; `signal-wash` is the only tinted background in the system, used for the probe glow
  ring, the checked switch track, and `::selection`. In dark the family shifts to
  `#45b8c2` / `#6fd3db` / `#11302f`, keeping the same roles.

### Secondary

- **Classification Marks** (`mark-critical`, `mark-high`, `mark-normal`, `mark-ok`): a
  fixed, closed set. Critical is a filled triangle, High a half-filled square, Normal a
  hollow square, and the set also carries the saved state on prompt stages (`mark-ok`).
  `mark-high` doubles as the system's warning edge: stale MCP tools, an unsaved prompt
  stage, a probe that matched nothing. In dark: `#f0705f`, `#dda43f`, `#8a9793`, `#5bbd85`.

### Tertiary

- **Danger** (`danger`, `danger-wash`): destructive affordances only — the delete confirm
  button, the quiet trash button on hover, invalid fields, error notices. Identical in hue
  to `mark-critical` in both themes but scoped to action, not classification.

### Neutral

- **Drafting Ground** (`ground`): the body field, carrying the grid. Never used on a
  component.
- **Panel** (`panel`): every raised surface — field, form section, strip, dialog, stage.
- **Sunk Panel** (`panel-sunk`): recessed areas — panel heads, dialog heads and feet, the
  rank column, input interiors, hover state of rows and quiet buttons.
- **Ink / Ink Muted / Ink Faint** (`ink`, `ink-muted`, `ink-faint`): body text, secondary
  prose and help text, engraved labels and placeholders respectively.
- **Rule / Rule Strong / Rule Faint** (`rule`, `rule-strong`, `rule-faint`): the three
  hairline weights. `rule` is the default container border; `rule-strong` is a committed
  edge (control borders, input borders, the rank trunk); `rule-faint` is an internal
  division (row separators, segment dividers, the body grid).

### Named Rules

**The Edge Colour Rule.** Colour lives on edges, marks and the primary action. No region
larger than a chip is ever filled with a hue; the text layer stays achromatic. If a state
needs to be visible, give it a 1–3px coloured edge (`inset 3px 0 0 var(--signal-edge)`),
not a background.

**The One Meaning Per Hue Rule.** Teal means *live*. Amber (`mark-high`) means *high
classification, or something needs attention*. They may never be swapped: an amber live
state would make "this path is live" and "this path is high priority" the same colour on
the same row. That collision is why the signal hue is teal.

**The AA Floor Rule.** Every text token clears 4.5:1 against `ground`, `panel` and
`panel-sunk` in both themes; the tightest pair in the shipped set is 4.9:1. A new colour
that does not clear the floor on all three surfaces does not enter the palette.

## Typography

**Display / Body Font:** Overpass (variable, 100–900), self-hosted woff2
**Label / Mono Font:** Overpass Mono (variable, 300–700), self-hosted woff2

**Character:** Overpass descends from Highway Gothic — signage lettering, built to be read
fast and at an angle. That lineage is the point for a routing panel: the sans carries plain
instruction text, and the mono carries every machine value (labels, tool names, durations,
channel IDs, ranks, counts) so that data never dresses as prose.

Both faces are self-hosted from `internal/web/static/fonts` and embedded in the Go binary,
split by `unicode-range` into latin / latin-ext / cyrillic subsets, with the latin subsets
preloaded. This is deliberate and load-bearing: the tool must render correctly in an
air-gapped cluster, so there is no CDN and no Google Fonts link.

### Hierarchy

The ramp is a fixed rem scale at roughly a 1.2 ratio (0.6875 / 0.8125 / 0.9375 / 1.0625 /
1.375 / 1.75rem). It does not fluid-scale: this is a product UI, not brand type.

- **Display** (600, 1.75rem, 1.15): the page title only, one per screen.
- **Title** (600, 1.0625rem): form-section and dialog headings, empty-state titles, and
  the mono readout values on the strip.
- **Body** (400, 0.9375rem, 1.5): default text and path names, tabular numerals globally.
- **Small** (400, 0.8125rem): controls, inputs, page lede, secondary prose.
- **Label** (mono, 600, 0.6875rem, 0.14em, uppercase): the engraved micro-label — field
  names, readout captions, column heads, classification marks, stage state.

### Named Rules

**The Engraved Label Rule.** Every field name, readout caption and column head is an
engraved micro-label: Overpass Mono, 0.6875rem, 600, 0.14em tracking, uppercase, in
`ink-faint`. Uppercase tracking is allowed at this size and nowhere else.

**The Machine Value Rule.** Anything the system produced or the operator typed as data —
label pairs, tool names, timeouts, channel IDs, ranks, counts — is set in Overpass Mono.
Anything written for a human to read is set in Overpass.

**The Air-Gap Rule.** Fonts ship in the binary. No webfont CDN, no external stylesheet, no
`@import` from a third party — on any surface, ever.

## Layout

A single centred shell at 1400px max width (`--shell-max`), with a 52px sticky header and
`s-6 / s-5 / s-7` main padding. Spacing is a seven-step rem rhythm (0.25 → 3rem); all
component padding is drawn from it.

The scenario field is the signature layout: a five-column grid
(`52px | minmax(200px,1fr) | minmax(220px,1.1fr) | minmax(190px,0.9fr) | auto`) shared
exactly between the column-head strip and every path row, so the heads stay registered
with the rows. The rank column is fixed-width because it draws a continuous trunk; the
three content segments flex.

The editor is a two-column grid (`208px | 1fr`) with a sticky margin rail at `top: 76px`
and matching `scroll-margin-top: 76px` on each section, so a rail jump lands clear of the
sticky header. Form fields inside a section use `repeat(auto-fit, minmax(240px, 1fr))`.

Two breakpoints, both content-driven:

- **≤1080px** — the path row refolds to a three-column areas grid (rank / name+investigation
  / destination / actions stacked); the column-head strip is hidden because the heads no
  longer register; the editor collapses to one column and the rail becomes a horizontal
  tab row with an underline active edge; the rail note is dropped.
- **≤720px** — the path row becomes rank + a single stacked column; shell padding drops to
  `s-3`; the route address and brand rule are hidden; readout cells wrap two-up; page-head
  buttons stretch to fill. The body grid tightens from 88px to 56px so the drafting texture
  stays proportional.

Measure is capped where prose appears: 62ch on empty states, 68ch on the page lede, 72ch
on help text, 104ch on prompt textareas and their footers.

### Named Rules

**The Shared Grid Rule.** Column heads and rows are the same `grid-template-columns`
declaration. If a column changes width, both change, or the heads are hidden.

**The Desktop-First Rule.** The audience is at a desktop with Grafana open next door.
Mobile refolds are honest and complete, but density is never reduced on desktop to make a
phone layout easier.

## Elevation & Depth

This system is flat. Depth is tonal and material, not atmospheric: surfaces separate by a
three-value ladder (`ground` → `panel` → `panel-sunk`) plus a 1px rule, never by a drop
shadow. There are exactly two ambient shadows in the whole build, and both belong to
genuinely floating layers.

State is expressed as *material change*: a pressed button takes an inset seat, a chosen
segment is pressed into the panel, a disabled control loses its fill and flattens, a live
row gains a coloured inset edge. None of these use elevation.

### Shadow Vocabulary

- **Seated press** (`inset 0 1px 2px rgb(0 0 0 / 0.16)`): `.btn:active`. The control takes
  the press as material.
- **Seated segment** (`inset 0 2px 3px rgb(0 0 0 / 0.12)`): the chosen segment of a
  segmented control.
- **Live edge** (`inset 3px 0 0 var(--signal-edge), inset 0 -1px 0 var(--signal-edge)`):
  the path a probe traced.
- **Nav seat** (`inset 0 -2px 0 var(--signal-edge)`): the current page in the header.
- **Probe node ring** (`0 0 0 3px var(--signal-wash)`): the rank node of a live path.
- **Pointer focus ring** (`0 0 0 2px color-mix(in srgb, var(--signal-edge) 60%, transparent)`):
  mouse focus on inputs, distinct from the keyboard outline.
- **Dialog lift** (`0 18px 44px rgb(0 0 0 / 0.22)`): the only true elevation, on `<dialog>`
  over a blurred `rgb(8 12 12 / 0.46)` backdrop.
- **Header veil** (`backdrop-filter: blur(6px)` over a 92% panel mix): the sticky header
  reads as frosted film over the grid, without a shadow.

### Named Rules

**The Flat-Sheet Rule.** Nothing in the content layer casts a shadow. Only a `<dialog>`
floats. Everything else separates with tone and a 1px rule.

**The Material State Rule.** Every control states itself as material change — seated,
engraved, flattened — before it states itself in colour. Colour alone is never the only
signal of a state.

## Shapes

Corners are effectively square: a single 2px radius (`--radius`) softens the laser edge
without reading as a rounded card, and it is applied uniformly to panels, controls, inputs,
chips, dialogs and notices. The two exceptions are fully round: the 22px rank node and the
18px switch track (`999px`), both of which are instruments rather than containers.

Borders carry the form language. Three hairline weights (`rule-faint`, `rule`,
`rule-strong`) do the structural work; a border is never thicker than 1px except as a state
edge (2px rail/nav edge, 3px live inset). Dashed borders are semantic, not decorative: a
dashed rank node means catch-all (the trunk terminates there), a dashed chip means an open
or defaulted value, a dashed box means an empty label set.

The classification marks are pure geometry drawn in CSS at 8–9px: a hollow square (normal),
a diagonal half-fill (high), a filled triangle via `clip-path` (critical). They are
readable as shapes with colour removed.

### Named Rules

**The 2px Corner Rule.** One radius for everything rectangular. A second container radius
is not introduced; if something needs to read as softer, it is the wrong component.

**The Dashed-Means-Open Rule.** A dashed edge always means "unbounded / unset / catches
everything". It is never used as decoration.

## Components

### Buttons

- **Shape:** near-square (2px radius), 34px tall, 0.4rem/0.85rem padding, 0.8125rem/500.
- **Default:** panel fill, `rule-strong` border, ink text. Hover sinks the fill to
  `panel-sunk` and darkens the border to `ink-faint`; active adds the seated inset.
- **Primary:** teal fill, `signal-strong` border, panel-coloured label (near-black in
  dark). One primary action per surface — the strip's New Scenario, the form's submit, the
  stage's Save.
- **Danger:** no fill at rest; `danger` text with a 45%-mixed border, filling with
  `danger-wash` on hover.
- **Quiet:** transparent with no border until the row is hovered or focused within, then a
  `rule` border appears; it turns danger-coloured on hover. Used for the row-level delete.
- **Small:** 28px, 0.6875rem, 0.04em tracking, for in-row and in-strip actions.
- **Working state:** htmx swaps the label for an inline spinner (`.btn-idle` /
  `.btn-working` toggled by `.htmx-request`) — the control states that it is busy rather
  than the page doing it for it.
- **Icon buttons** are 32×32 with the same border logic; every icon is an inline stroked
  SVG at 1.25 stroke weight, 14–18px.

### Chips

- **Style:** mono 0.6875rem, `panel-sunk` fill, `rule` border, 1px/6px padding, 2px radius,
  ellipsised. The key is `ink-faint` and gets a rule-coloured `=` from `::after`, so a
  label pair reads as `key=value` without markup for the separator.
- **Open variant** (`data-open="true"`): dashed border, no fill — "catch-all", "default
  channels".
- **Stale variant** (`data-stale="true"`): amber border and text — a tool no connected MCP
  server offers. Kept, never silently dropped.
- Chips carry *values*. Classification never uses a chip.

### Marks

- **Style:** an 8–9px CSS glyph plus a mono uppercase word at 0.1em tracking, coloured from
  the classification set.
- **Geometry:** hollow square (normal), diagonal half-fill (high), filled triangle
  (critical). In the rank column and in the segmented control the word is suppressed
  (`font-size: 0`) and only the glyph shows, with the full word carried on `title` and in
  the adjacent label.

### Cards / Containers

- **Corner style:** 2px.
- **Background:** `panel`; heads and feet `panel-sunk`.
- **Border:** 1px `rule`; internal divisions 1px `rule-faint`.
- **Shadow:** none (see Elevation & Depth).
- **Internal padding:** `s-4` body, `s-3`/`s-4` head. Form sections use
  `s-4 s-5 s-5`.

### Inputs / Fields

- **Style:** sunk (`panel-sunk`) with a `rule-strong` 1px border, 2px radius,
  0.4rem/0.6rem padding, teal caret. `.code` switches the field to mono for machine values.
- **Hover:** border darkens to `ink-faint`.
- **Focus:** keyboard focus gets the global 2px `signal-edge` outline at 2px offset plus a
  teal border; pointer focus gets a 2px translucent teal ring instead, so the keyboard ring
  stays distinctive.
- **Invalid:** `aria-invalid="true"` turns the border danger-coloured and a mono error line
  appears beneath.
- **Textareas:** vertical resize only, 84px minimum, auto-resizing on input and again after
  fonts settle.
- **Segmented control:** a bordered strip of radio labels with `rule` dividers; the checked
  segment is seated into the panel with an inset shadow and 600 weight.
- **Switch:** a 34×18px appearance-none checkbox; checked turns the track `signal-wash`
  with a teal border and slides a teal 12px knob.

### Navigation

- **Header:** 52px, sticky, frosted (`blur(6px)` over 92% panel), bottom `rule`. Brand mark
  (an inline teal signal-path SVG) + 1px divider + engraved route address
  (`PANEL / SCENARIOS / EDITOR`), nav pushed right, theme toggle at the end.
- **Links:** 0.8125rem/500 `ink-muted`, transparent border; hover reveals a `rule` border.
- **Current:** `panel-sunk` seat, `rule-strong` border, and an `inset 0 -2px 0` teal edge.
- **Mobile (≤720px):** the route address and brand divider are dropped; the two nav links
  and the toggle remain.
- **Theme toggle:** one icon button showing sun or moon by `[data-theme]` CSS, writing
  `localStorage.theme`, with an inline head script applying the stored value — or the OS
  preference when unset — before first paint.

### The Path Row (signature)

The system's defining component: one scenario drawn as a signal path across the sheet.

- **Rank trunk:** a 1px `rule-strong` vertical line drawn down the centre of the rank
  column by `::before`, clipped to start at the first row's node and end at the last —
  a continuous trunk every alert travels down.
- **Rank node:** a 22px round mono node on the trunk carrying the evaluation number
  (position *is* the order; the number is a consequence). Dashed border = catch-all, the
  trunk terminates there.
- **Classification mark:** the engraved mark sits on the row's left edge beside the trunk,
  glyph-only, with the panel-sunk background knocking the trunk out behind it.
- **Three segments:** name + prompt preview → catches (label chips) → investigates &
  delivers (mono facts + destination chips), divided by 1px `rule-faint`, with a matching
  `rule-faint` row separator.
- **Hover:** the whole row sinks to `panel-sunk`.
- **Live:** `data-live="true"` (set by the match probe) draws a 3px teal inset left edge
  and a teal bottom edge, rings the rank node with `signal-wash`, and animates a teal
  tracer 420ms down the trunk to the node.
- **Actions:** Edit / Copy / quiet-delete, right-aligned, revealing their borders on row
  hover or focus-within.

### The Instrument Strip

A single bordered strip above the field holding, left to right: four mono readout cells
(Paths / Critical / High / Catch-all, each an engraved caption over a 1.0625rem mono value,
critical and high tinted by their marks), the match probe (engraved label, mono input
placeholdered `alertname=HighCPU, severity=critical`, Trace button), and the primary New
Scenario action at the right end. Cells are divided by 1px rules and wrap two-up under
720px. The probe result renders below as a notice-style block with a coloured left edge:
teal when a path caught the labels, amber when nothing matched, danger on error.

### The Margin Rail

A sticky 208px column of numbered form steps with a `rule` right border. The active step
takes a 2px teal left edge and 600 weight, driven by an IntersectionObserver on the
sections. A rail note in `ink-faint` micro type explains the model. Under 1080px the rail
lies down as a horizontal row and the active edge moves to the bottom.

### The Label-Pair Editor

A grid of `key | = | value | remove` rows (`1fr 14px 1.2fr 32px`) in mono inputs, with a
mono `=` glyph between them, an Add Label action, and a dashed empty block that states the
consequence ("this path is a catch-all"). Rows serialise into a hidden textarea so the form
posts the same shape the YAML uses.

### The Tool Picker

A `<dialog>` with a search field, a status line, and a 1px-gapped list of rows
(`checkbox | name + 2-line clamped description`) over a `rule-faint` background so the gaps
read as hairlines. Tool names are mono; a tool no connected MCP server offers is rendered
in amber in both the picker and its chip, and is kept rather than dropped. The foot pairs
"Allow all tools" with the primary Done.

### Inline Delete Confirm

No browser dialog. `data-confirming="true"` on the row swaps the default actions for an
inline "Delete? / Yes, delete / Keep" group in danger type; focus moves to the confirm
button and Escape restores it.

### Prompt Stage

A panel per stage with an engraved state readout in the head that moves through In sync →
Unsaved changes (amber) → Saved (green). A dirty stage takes an amber-mixed border on the
whole panel; Save and Revert are disabled until the textarea differs from its original.

### Named Rules

**The Remount Rule.** Every initializer in `panel.js` is idempotent and re-runs on
`htmx:afterSwap`; per-element binding is guarded by a `dataset` flag. htmx replaces whole
page bodies, so any new behaviour must survive being mounted again against fresh DOM.

**The Consequence-Visible Rule.** A control that changes routing states its consequence in
place: the empty label set says it becomes a catch-all, the probe marks the row that would
win, the dirty stage marks itself before you leave.

## Do's and Don'ts

### Do:

- **Do** keep the content layer achromatic and confine colour to edges, marks and the
  single primary action (The Edge Colour Rule).
- **Do** use instrument teal only for *live* and `mark-high` amber only for *high
  classification / needs attention*; never swap them.
- **Do** set every machine value in Overpass Mono and every field name as an engraved
  micro-label (mono, 0.6875rem, 600, 0.14em, uppercase, `ink-faint`).
- **Do** separate surfaces with the `ground → panel → panel-sunk` tonal ladder plus a 1px
  rule.
- **Do** express state as material change first — seated, engraved, flattened — and colour
  second.
- **Do** use one radius (2px) for every rectangular thing; reserve `999px` for the rank
  node and the switch.
- **Do** make a dashed edge mean open/unset/catch-all, and nothing else.
- **Do** keep the body's drafting grid: it is the world's ground, not a decoration to be
  cleaned up.
- **Do** self-host every font in the binary and preload the latin subsets.
- **Do** ship both themes together and check new colours at 4.5:1 against `ground`,
  `panel` and `panel-sunk` in each.
- **Do** write every new initializer to be idempotent and re-runnable on `htmx:afterSwap`.
- **Do** draw icons as inline stroked SVG at 1.25 stroke weight, 14–18px.
- **Do** keep column heads and rows on the same grid declaration, or hide the heads.

### Don't:

- **Don't** fill a region with a hue. No gradient fills, no tinted cards, no coloured
  banners; `signal-wash` is a ring and a selection, not a background for content.
- **Don't** render classification as a coloured pill, badge or chip — it is an engraved
  mark whose geometry reads without colour.
- **Don't** add a drop shadow to anything in the content layer; only `<dialog>` floats.
- **Don't** introduce a second container radius or a rounded-card stack.
- **Don't** load a font, stylesheet or script from a CDN; the tool must run air-gapped.
- **Don't** use uppercase letter-spaced type above the 0.6875rem engraved label size.
- **Don't** use a browser `confirm()`/`alert()` — destructive confirmation is inline on the
  row it affects.
- **Don't** replace the row-level grid with a generic table; the rank trunk, the node and
  the segment rules are the component.
- **Don't** silently drop a stale MCP tool; mark it amber and keep it.
- **Don't** make colour the only carrier of a state — pair it with a mark, an edge, or a
  word.
