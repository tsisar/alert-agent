---
version: 1
slug: "internal-web-templates-scenarios-templ"
primary_target: "internal/web/templates/scenarios.templ"
related_targets: ["internal/web/templates/prompts.templ","internal/web/templates/layout.templ","internal/web/static/app.css"]
---

## Direction contract

THESIS: A scenario is a signal path, not a table row. The surface owns one idea — you can
see which alert is caught by which rule, in what order, and where its report is sent,
without opening anything. It refuses the category default (dark SaaS console: sidebar,
rounded cards, Inter, blue accent) and its soft opposite (pastel admin panel).

OWN-WORLD: Technical drawing on vellum, silkscreened instrument panel. Achromatic ground
(paper/graphite in light, ink/slate in dark), 1px hairline rules that behave like schematic
lines, engraved uppercase micro-labels in Overpass Mono (signage lineage: Highway Gothic),
Overpass for text. Colour is restrained and lives on edges only: one signal hue for the
active path, a fixed engraved mark set for critical/high/normal/catch-all. No shadows as
decoration, no rounded card stacks, no gradient fills.

AMENDED 2026-09-18 — the signal hue is instrument teal (`--signal`), not the amber this
contract first named. Amber is spent on the HIGH classification mark, and an amber signal
would have made "this path is live" and "this path is high priority" the same colour on the
same row. Operate mode gives clarity the conflict; the contract follows the build.

STORY: An SRE opens the panel, scans the field top to bottom in evaluation order, and knows
which path an incoming alert takes. They test a label set against the field, see the exact
path that catches it, open that path, and change one segment with the consequence visible.

FIRST VIEWPORT: Full-width patch field. Left: a rank ladder column carrying evaluation order
(position is the order; the number is a consequence, not an input). Each row is one scenario
drawn as three segments across the width — MATCH (label chips) → INVESTIGATION (tools +
timeout budget) → DESTINATION (channels), joined by hairline routes. Priority reads as an
engraved mark on the row's left edge; catch-all reads as an open-ended route. Above the
field: a status strip with the count, a match-probe input ("paste labels, see who catches
it"), and the single primary action, New Scenario, at the right end of the strip.

FORM: Signal routing panel; candidate 7 of the grounded list; seed key 3ad57621.

RAISE — MATERIAL STATE: every control states itself as material change (seated, engraved,
flattened when disabled), not colour alone. From the declined CD-ROM console challenger.
RAISE — ENGRAVED MARKS: classification is a formal mark system, never a coloured pill. From
the declined TDR sleeve challenger.
RAISE — MARGIN RAIL: the editor carries a fixed margin rail of numbered steps with the
current position always visible and recoverable. From the declined orizuru challenger.
RAISE — RANK LADDER: evaluation rank is legible without reading a number. From the declined
cracktro challenger.
RAISE — KEYED ADDRESS: every scenario is reachable by typing; the field filters from the
keyboard without touching the mouse. From the competitive teletext challenger.
RAISE — EDGE COLOUR: the text field stays achromatic; colour is confined to hairline edges
and marks. From the competitive iridescent-edge challenger.

FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review,
the verdict, DESIGN.md, and every shipping raster carrying its provenance.
