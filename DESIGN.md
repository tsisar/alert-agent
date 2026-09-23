---
name: Alert Agent
description: A plain, familiar operations tool for alert scenarios. Neutral grey ground, white cards, one blue accent, semantic priority pills.
colors:
  bg: "#f5f6f8"
  surface: "#ffffff"
  surface-2: "#f6f7f9"
  surface-hover: "#f0f2f5"
  border: "#dcdfe4"
  border-strong: "#c3c8cf"
  text: "#1a1d21"
  text-2: "#4a5260"
  text-3: "#5f6875"
  accent: "#2360d8"
  accent-hover: "#1b4fb8"
  accent-text: "#1f56c4"
  accent-soft: "#e9f0fd"
  accent-ring: "rgb(35 96 216 / 0.35)"
  on-accent: "#ffffff"
  danger: "#c42b1c"
  danger-hover: "#a82417"
  danger-soft: "#fdeceb"
  warn: "#8f5500"
  warn-soft: "#fff3dd"
  warn-border: "#f0c66b"
  ok: "#19703a"
  ok-soft: "#e5f4ea"
  p-critical: "#c42b1c"
  p-critical-soft: "#fdeceb"
  p-high: "#9a5800"
  p-high-soft: "#fff1d6"
  p-normal: "#4a5260"
  p-normal-soft: "#eef0f3"
typography:
  headline:
    fontFamily: "-apple-system, BlinkMacSystemFont, Segoe UI, system-ui, Roboto, Helvetica Neue, Noto Sans, Liberation Sans, Arial, sans-serif"
    fontSize: "1.5rem"
    fontWeight: 650
    lineHeight: 1.25
    letterSpacing: "-0.01em"
  title:
    fontFamily: "-apple-system, BlinkMacSystemFont, Segoe UI, system-ui, Roboto, Helvetica Neue, Noto Sans, Liberation Sans, Arial, sans-serif"
    fontSize: "1rem"
    fontWeight: 650
    lineHeight: 1.35
  title-sm:
    fontFamily: "-apple-system, BlinkMacSystemFont, Segoe UI, system-ui, Roboto, Helvetica Neue, Noto Sans, Liberation Sans, Arial, sans-serif"
    fontSize: "0.9375rem"
    fontWeight: 600
    lineHeight: 1.5
  body:
    fontFamily: "-apple-system, BlinkMacSystemFont, Segoe UI, system-ui, Roboto, Helvetica Neue, Noto Sans, Liberation Sans, Arial, sans-serif"
    fontSize: "0.875rem"
    fontWeight: 400
    lineHeight: 1.5
    fontFeature: "tnum"
  body-sm:
    fontFamily: "-apple-system, BlinkMacSystemFont, Segoe UI, system-ui, Roboto, Helvetica Neue, Noto Sans, Liberation Sans, Arial, sans-serif"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.5
  label:
    fontFamily: "-apple-system, BlinkMacSystemFont, Segoe UI, system-ui, Roboto, Helvetica Neue, Noto Sans, Liberation Sans, Arial, sans-serif"
    fontSize: "0.75rem"
    fontWeight: 600
    lineHeight: 1.5
  mono:
    fontFamily: "ui-monospace, SFMono-Regular, SF Mono, Menlo, Cascadia Mono, Consolas, Liberation Mono, DejaVu Sans Mono, monospace"
    fontSize: "0.8125rem"
    fontWeight: 400
    lineHeight: 1.5
rounded:
  xs: "4px"
  sm: "6px"
  md: "8px"
  lg: "12px"
  pill: "999px"
spacing:
  xs: "4px"
  sm: "8px"
  md: "12px"
  lg: "16px"
  xl: "20px"
  2xl: "24px"
  3xl: "32px"
components:
  button:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text}"
    typography: "{typography.body}"
    rounded: "{rounded.sm}"
    padding: "0 0.85rem"
    height: "34px"
  button-hover:
    backgroundColor: "{colors.surface-hover}"
  button-primary:
    backgroundColor: "{colors.accent}"
    textColor: "{colors.on-accent}"
    rounded: "{rounded.sm}"
    padding: "0 0.85rem"
    height: "34px"
  button-primary-hover:
    backgroundColor: "{colors.accent-hover}"
  button-danger:
    backgroundColor: "{colors.danger}"
    textColor: "{colors.on-accent}"
    rounded: "{rounded.sm}"
    height: "34px"
  button-danger-hover:
    backgroundColor: "{colors.danger-hover}"
  button-ghost:
    backgroundColor: "transparent"
    textColor: "{colors.text-2}"
    rounded: "{rounded.sm}"
    height: "34px"
  button-sm:
    typography: "{typography.body-sm}"
    padding: "0 0.65rem"
    height: "30px"
  input:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text}"
    typography: "{typography.body}"
    rounded: "{rounded.sm}"
    padding: "0 0.7rem"
    height: "34px"
  card:
    backgroundColor: "{colors.surface}"
    rounded: "{rounded.md}"
  card-foot:
    backgroundColor: "{colors.surface-2}"
    padding: "0.85rem 1.25rem"
  badge-critical:
    backgroundColor: "{colors.p-critical-soft}"
    textColor: "{colors.p-critical}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    padding: "0 0.5rem"
    height: "20px"
  badge-high:
    backgroundColor: "{colors.p-high-soft}"
    textColor: "{colors.p-high}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    height: "20px"
  badge-normal:
    backgroundColor: "{colors.p-normal-soft}"
    textColor: "{colors.p-normal}"
    typography: "{typography.label}"
    rounded: "{rounded.pill}"
    height: "20px"
  chip:
    backgroundColor: "{colors.surface-2}"
    textColor: "{colors.text}"
    rounded: "{rounded.xs}"
    padding: "0 0.45rem"
    height: "22px"
  table-head:
    backgroundColor: "{colors.surface-2}"
    textColor: "{colors.text-2}"
    typography: "{typography.label}"
    padding: "0.55rem 1rem"
  table-row-live:
    backgroundColor: "{colors.accent-soft}"
  dialog:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.text}"
    rounded: "{rounded.lg}"
    width: "min(640px, calc(100vw - 2rem))"
---

# Design System: Alert Agent

## Overview

**Creative North Star: "The Well-Kept Settings Page"**

Alert Agent looks like the tools its SREs already live in: Grafana, Linear, GitHub settings. Nothing about the surface asks to be learned. A neutral grey ground holds white cards; each card is a task (test an alert, the scenario table, a form section) with a plain title and the controls for that task inside it. One blue accent marks the thing you can do next and the thing that is selected; red, amber and grey carry priority and state and nothing else.

Density is desktop-tool density: a 14px body, 34px controls, 16px stroked icons beside text labels, tabular numerals everywhere. Every action is visible at rest and labelled; nothing hides behind hover. The system is deliberately unauthored at the surface level so that the content (scenario names, label matchers, prompts) is what the eye reads.

Light and dark are both first-class. Every colour token has a dark counterpart under `:root[data-theme="dark"]`; the theme follows the OS preference until the user toggles it.

**Key Characteristics:**
- Neutral grey ground, white 8px-radius cards with a soft two-layer shadow.
- System UI font stack; weights 400 / 500 / 550 / 600 / 650.
- One blue accent for primary buttons, current nav and tab, selection, focus and the matched row.
- Semantic pills for priority: critical red, high amber, normal grey.
- 16px stroked SVG icons (1.75 stroke, round caps) paired with text labels.
- Labels written in sentence case, never uppercased or letter-spaced.

## Colors

A cool-neutral grey scale with a single saturated blue and three semantic hues; colour means action, selection or state, never decoration.

### Primary
- **Action Blue** (accent): primary buttons, the current top-nav and tab underline, switch "on", focus outlines, the brand mark and the text caret. Hover deepens to **Pressed Blue** (accent-hover).
- **Link Blue** (accent-text): inline links and the icon in a "matched" test result; a slightly darker step than Action Blue so text sits above 4.5:1.
- **Selection Wash** (accent-soft): the background of a selected pick row, text selection, and the scenario row that a test alert matched (with a 45% accent inset ring).
- **Focus Halo** (accent-ring): the 3px ring around a focused input.

### Secondary (semantic)
- **Stop Red** (danger / p-critical): destructive buttons, error notices, invalid inputs, the critical priority pill and dot. **Blush** (danger-soft) is its background and the row tint while a delete is being confirmed.
- **Caution Amber** (warn / p-high): unsaved-changes state, stale tools, warning notices, the high priority pill. **Cream** (warn-soft) and **Straw** (warn-border) are its fill and edge.
- **Go Green** (ok, ok-soft): the "saved" state pill only.

### Neutral
- **Desk Grey** (bg): the page ground behind cards, and the fade behind the sticky save bar.
- **Paper** (surface): cards, inputs, default buttons, the top bar, dialogs.
- **Shelf Grey** (surface-2): table header, group header rows, card and dialog footers, chips, the segmented control track.
- **Hover Grey** (surface-hover): row, ghost-button and tab hover.
- **Hairline** (border): card edges, table rules, dividers. **Firm Line** (border-strong): input and button edges, empty-state dashes, switch "off".
- **Ink** (text), **Slate** (text-2, also p-normal), **Pewter** (text-3): primary text, secondary text and labels, help/meta/placeholder text.

### Named Rules
**The One Blue Rule.** Blue means "you can act here" or "this is selected". It never decorates a heading, a divider or an empty area.

**The Meaning-Only Hue Rule.** Red, amber and green appear only to report priority or state, always as a soft fill with a same-hue text colour, never as a solid decorative block.

## Typography

**Body Font:** the platform UI stack (-apple-system, Segoe UI, system-ui, Roboto, ... Arial, sans-serif)
**Label/Mono Font:** the platform monospace stack (ui-monospace, SF Mono, Menlo, Cascadia Mono, Consolas, ...)

**Character:** The operating system's own voice. It is the register of the tools SREs already use, so the UI reads as a utility, not a product. Mono is reserved for machine values: label matchers, tool names, timeouts, channel IDs, prompt text.

### Hierarchy
- **Headline** (650, 1.5rem, 1.25, -0.01em): the one page title per screen.
- **Title** (650, 1rem, 1.35): card titles, dialog titles, empty-state titles.
- **Title Small** (600, 0.9375rem): scenario names in the table, the brand name.
- **Body** (400, 0.875rem, 1.5, tabular numerals): everything by default. Ledes and help text cap at 72ch.
- **Body Small** (400, 0.8125rem): help text, prompt previews, table facts, counts, group titles.
- **Label** (600, 0.75rem): table column headers, priority pills, stacked-row field labels on narrow screens.
- **Mono** (400, 0.8125rem; 0.75rem inside chips): labels, tool names, durations, channels, editable prompt text.

### Named Rules
**The Sentence Case Rule.** Every heading, label and column header is sentence case at normal tracking. Hierarchy comes from weight (550 to 650) and size, never from uppercase or letter-spacing.

**The Mono Means Machine Rule.** A value the machine will read (a label, a tool, a duration, a channel) is set in mono; prose about it is not.

## Layout

A single centred column capped at 1280px, with 24px side padding (16px under 720px). A sticky 56px top bar carries the brand, two tab-style nav links and the theme toggle. Each page opens with a page head: title and a one-sentence lede on the left, page actions right-aligned and bottom-aligned; under 720px it stacks and the actions stretch.

Content is a vertical stack of cards separated by 16px. Inside cards, padding is 16 to 20px (1rem 1.25rem); form sections use a 20px grid gap and two-column rows that collapse to one under 860px. The editor column caps at 880px. The spacing rhythm is a 4px base: 4, 8, 12, 16, 20, 24, 32.

Tables are the primary list form. Under 860px the scenario table becomes a stack of two-column grid rows (order number in the left column, fields with small labels on the right). The "Test an alert" card goes from a two-column text/control split to a single column under 1080px.

The primary save action for long forms lives in a sticky bottom save bar (a raised card over a fade of the page ground), so it is always on screen.

## Elevation & Depth

A light, layered system: surfaces are separated by hairline borders first and by soft, cool-tinted shadows second. Depth has three steps and each step has one job.

### Shadow Vocabulary
- **Rest** (`0 1px 2px rgb(16 24 40 / 0.06)`): default buttons, so they read as pressable against white.
- **Card** (`0 1px 3px rgb(16 24 40 / 0.08), 0 1px 2px rgb(16 24 40 / 0.04)`): cards and the selected segment of a segmented control.
- **Floating** (`0 16px 40px rgb(16 24 40 / 0.18), 0 4px 12px rgb(16 24 40 / 0.08)`): only things that float over content, the dialog and the sticky save bar.

Dark theme swaps these for black-based equivalents (0.3 / 0.35 / 0.55 alpha). Dialog backdrop is `rgb(10 14 20 / 0.45)`.

### Named Rules
**The Float Earns Floating Rule.** The large shadow is reserved for surfaces that sit above scrolling content. A card in the page flow never gets it.

## Shapes

Gently rounded, never pill-shaped except for status. Controls (buttons, inputs, segmented track, pick lists) use 6px; cards and notices 8px; dialogs 12px; small in-line objects (chips, kbd keys, extra-small icon buttons, segments) 4px. Only priority/state pills and the switch are fully round. Every surface has a 1px border; dashed borders mean "empty, add something here" (empty pairs, the open "any tool" chip).

## Components

### Buttons
Quiet and solid: white by default, blue for the one primary action per area.
- **Shape:** 6px corners, 34px tall (30px small, 24px extra-small icon), 1px Firm Line border, Rest shadow, 550 weight.
- **Primary:** Action Blue fill, white text; hover to Pressed Blue. One per region (New scenario, Save).
- **Default:** Paper fill, Ink text; hover to Hover Grey. Used for every secondary action (Export, Import, Edit, Run test, presets).
- **Ghost:** transparent, no border or shadow, Slate text; for icon-only chrome (theme toggle, dialog close, reorder arrows).
- **Danger:** Stop Red fill, white text, only as the confirm step of a delete. The at-rest delete is a default icon button that turns red-on-blush on hover.
- **States:** 120ms colour transitions, 1px press-down on active, 0.45 opacity when disabled, 2px Action Blue focus outline at 2px offset. Busy buttons swap their label for a spinner plus a verb ("Saving").
- **Icons:** 16px stroked SVG with a text label; icon-only buttons are square and carry an aria-label and title.

### Chips and Badges
- **Priority/state pill:** 20px, fully round, Label type, soft fill plus same-hue text (critical, high, normal, dirty, saved). A matching 8px dot is used where a pill would be too loud (segmented priority picker).
- **Label chip:** 22px, 4px corners, Shelf Grey fill, Hairline border, mono 0.75rem, key in Pewter followed by "=" and the value in Ink. Stale tools turn amber; the "any" chip is dashed and transparent.

### Cards / Containers
- **Corner Style:** 8px.
- **Background:** Paper, with Shelf Grey head/foot strips where a card has actions.
- **Shadow Strategy:** Card shadow (see Elevation).
- **Border:** 1px Hairline.
- **Internal Padding:** 16px vertical, 20px horizontal; footers 0.85rem by 1.25rem and right-aligned.

### Inputs / Fields
- **Style:** 34px, Paper fill, 1px Firm Line border, 6px corners; textareas start at 96px, prompt editors at 320px. Blue caret.
- **Hover / Focus:** border darkens to Pewter on hover; on focus the border turns Action Blue with a 3px Focus Halo.
- **Error:** Stop Red border. Placeholders are Pewter at full opacity.
- **Field:** a 600-weight label above, Body Small help text below in Pewter.
- **Segmented control and switch:** a Shelf Grey track with a raised white selected segment; a 36 by 20 switch that fills Action Blue when on.

### Navigation
- **Top nav:** 500-weight Slate links, full top-bar height, 2px underline that is Firm Line on hover and Action Blue (with Ink, 600 text) on the current page.
- **Tabs:** the same underline grammar at content level (prompt stages); hover adds a Hover Grey fill with 6px top corners; an amber 7px dot marks a tab with unsaved changes.
- **Brand mark:** a white bell with a small four-point spark on a 7px-rounded Action Blue (#2360d8) tile: an alert, and the agent that investigates it. It is one file, `static/favicon.svg`, used both as the favicon and as the 24px mark in the top bar, so the tab and the page never disagree. The tile colour is fixed; it does not follow the theme.
- **Mobile:** the brand collapses to its mark; nav stays inline.

### Scenario Table (signature)
Grey Label-type header strip, 16px cell padding, Hairline row rules, Hover Grey on hover. The order column shows a 26px numbered tile beside stacked up/down ghost arrows. The row that a test alert matched gets the Selection Wash with a blue inset ring. Delete confirmation happens in the row: the row tints Blush and the actions swap for "Delete this scenario?" with Delete and Cancel. Group headers (Fallback) are Shelf Grey rows with a 650 Body Small title and a note.

### Sticky Save Bar (signature)
A raised card (Floating shadow, 8px) pinned to the bottom of long forms: a save-state dot and text on the left (grey when clean, amber when dirty, green once saved), the secondary action and Save on the right. The scenario editor uses Cancel and Save; each prompt tab has its own bar with Discard changes and Save. Forms and the list page share the page width, and the page always reserves the scrollbar's gutter, so the layout does not shift between pages.

### Dialogs
Up to 640px wide and a fixed height (at most 680px), 12px corners, Floating shadow, head with title and ghost close button, Shelf Grey footer with help text left and actions right. The search field stays put and the pick list is the only thing that scrolls; the page behind is locked while a dialog is open. Pick lists inside are bordered 6px lists whose checked rows take the Selection Wash.

## Do's and Don'ts

### Do:
- **Do** pair every action icon with a visible text label, except square icon-only buttons that carry an aria-label and title.
- **Do** keep every row and card action visible at rest; hover may tint, never reveal.
- **Do** use the soft-fill plus same-hue-text pattern for any priority or state indicator.
- **Do** set machine values (labels, tools, durations, channels) in the mono stack.
- **Do** define every new colour in both `:root` and `:root[data-theme="dark"]`.
- **Do** keep the primary action for a long form in the sticky save bar.

### Don't:
- **Don't** use more than one blue primary button in a single region.
- **Don't** uppercase or letter-space headings, labels or column headers.
- **Don't** use the Floating shadow on anything that scrolls with the page.
- **Don't** use red, amber or green for anything other than priority or state.
- **Don't** introduce a webfont; the system stack is the voice.
- **Don't** use text glyphs as icons; icons are 16px stroked SVG.
