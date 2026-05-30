---
type: always
description: Workflow for tackling non-trivial or multi-layer issues in mroki.
---

# Issue workflow (non-trivial / multi-layer changes)

Apply this **only when the task is non-trivial or multi-layer** — a new feature/behavior, a change
spanning more than one `internal/` layer, or one touching multiple files. **Skip it for trivial
changes** (typos, one-line fixes, isolated doc tweaks); handle those directly.

When it applies, follow the "Working on an issue" workflow in `AGENTS.md`, with these Augment
specifics:

1. Restate the goal and list the affected layers.
2. Present a **per-layer summary of the planned changes** and get user approval before implementing
   (use plan mode / the approval prompt). Do not start editing until approved.
3. Implement layer by layer (schema → domain → application → repository/mapper → handler → DTO).
4. Add/update tests and run `make lint` + `make test` (or the scoped targets) until green.
5. **Launch a read-only review sub-agent** (an `explore`/`validate`-style agent that makes no edits)
   to cross-check code ↔ tests ↔ docs and report gaps or inconsistencies. Address its findings
   before handing back.
