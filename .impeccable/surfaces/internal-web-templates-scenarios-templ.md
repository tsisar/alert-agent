---
version: 1
slug: "internal-web-templates-scenarios-templ"
primary_target: "internal/web/templates/scenarios.templ"
related_targets: ["internal/web/templates/prompts.templ","internal/web/templates/layout.templ","internal/web/static/app.css"]
---

## Scope

Operate mode. Scenario list, scenario editor, prompt templates. Audience: SREs at a desktop.

## Direction contract

THESIS: The user should never have to hunt for an action. Every task (test an alert, reorder, edit, save) has a visible, labelled control where you would expect it. This replaces the earlier "signal routing panel" world, which the user rejected as too hard to scan.
OWN-WORLD: The category standard, done straight, in the register of Grafana, Linear and GitHub settings: neutral grey ground, white cards with 8px radius and a soft shadow, a system font stack, one blue accent for primary buttons, selection and focus, semantic pills for priority (critical red, high amber, normal grey), and consistent 16px stroked icons on actions.
STORY: The SRE opens Scenarios and reads the table top to bottom in evaluation order, with a separate Fallback group. They type labels into "Test an alert" and see the matching row highlighted. They reorder with the arrows, click Edit, change a field, and press Save in the sticky bar (or Ctrl+S).
FIRST VIEWPORT: The page title and a one-sentence rule on the left; Export, Import and a primary "New scenario" on the right. Below that, a "Test an alert" card. Then the scenarios card: a filter field (press /), a table with Order (number plus up/down arrows), Scenario (name, priority pill, prompt preview), Matches labels, Investigation, Sends to, and actions (Edit, Duplicate, Delete) that are always visible.
FORM: Canon (the standing exit), chosen by the user in the structured question round. No concept-seed roll ran because the user picked the familiar path outright.
FINISH: unreviewed and undocumented is unfinished; this build ends with the finish review, the verdict, DESIGN.md, and every shipping raster carrying its provenance
