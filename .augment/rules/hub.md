---
type: auto
description: Conventions for the Vue 3 + TypeScript hub frontend — apply when editing files under web/mroki-hub.
---

# Hub (web/mroki-hub)

Applies when editing files under `web/mroki-hub`. Use `pnpm` (Node.js 22 LTS, pnpm 10). Full detail
in `AGENTS.md`.

- **TypeScript is required** in every Vue component: `<script setup lang="ts">` (enforced by ESLint
  `vue/block-lang`). Plain `<script setup>` fails lint.
- Composition API with `<script setup>`. Tailwind CSS v4 + shadcn-vue.
- **Use semantic CSS color tokens** (`bg-background`, `text-foreground`, `bg-primary`,
  `text-muted-foreground`, …) — never hardcoded colors like `bg-white` / `text-gray-900`.
- `web/mroki-hub/src/components/ui/` is generated (shadcn-vue) — do not hand-edit.
- Before committing: `pnpm lint` and `pnpm format`. Tests: `pnpm test:unit` (vitest),
  `pnpm test:e2e` (Playwright, spins up the backend stack).
