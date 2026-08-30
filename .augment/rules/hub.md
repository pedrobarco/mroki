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

## Data fetching (TanStack Query)

Read server state through TanStack Query — never call raw `@/api` fetch functions from components or
build ad-hoc reactive caches.

- **Query keys**: use the hierarchical `queryKeys` factory in `src/api/query-keys.ts`
  (`all → lists() → list(params) → details() → detail(id)`); never hand-write a key array.
- **Reads**: consume the `queryOptions` factories in `src/api/queries.ts` (`gatesQuery`, `gateQuery`,
  `requestsQuery`, `requestQuery`, `globalStatsQuery`, `configQuery`) via `useQuery(...)`.
- **Writes**: use the `useMutation` composables in `src/api/mutations.ts`, which invalidate the
  matching keys `onSuccess`.
- **Every list/detail view renders all three states**: loading (`isPending`), error (`isError` +
  `error.message` with a Retry that calls `refetch()`), and empty. Lists page/sort server-side via
  headless `@tanstack/vue-table` with `placeholderData: keepPreviousData`.
- **Tests** mock the underlying `@/api/*` module and mount with a fresh, retry-free `QueryClient`
  per test; e2e forces the states with `page.route()` (`e2e/query-states.spec.ts`).
