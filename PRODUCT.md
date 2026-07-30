# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Primary users are **backend / platform engineers** who are about to change a
production HTTP service — a refactor, a database migration, or a framework
upgrade — and need to validate that change against real production behavior
*before* rolling it out. They operate in the pre-rollout window: the change is
built and running on a shadow instance, and they need evidence it behaves
identically to live before they trust it with real users.

## Product Purpose

mroki mirrors live HTTP traffic to a shadow service, diffs the JSON responses,
and surfaces the differences — so engineers can validate changes against real
production behavior before rolling them out. Success is catching a behavioral
regression from *real* request patterns (not synthetic tests) while the change
is still shadow-only, and doing so without ever putting production traffic at
risk.

## Positioning

mroki validates a candidate against **real production traffic with zero risk to
the live response**. The proxy forwards each request to both the live and shadow
services, always returns the live response to the client, and treats all
shadow/API work as best-effort — an API outage, a slow shadow, or a diff error
never fails or delays live traffic. A neighboring tool that replays synthetic
traffic, or that sits in the live response path, cannot truthfully make that
combined claim.

## Operating Context

mroki is a four-component system used around a production HTTP service:

- **mroki-proxy** — forwards each request to the live and shadow services in
  parallel, returns the live response to the client, and sends the raw captured
  responses to the API. Deployed as a sidecar or standalone service. Tags shadow
  traffic with `X-Mroki-Mode: shadow`; never modifies live requests.
- **mroki-api** — REST API that manages gates, computes JSON diffs server-side,
  and persists requests, responses, and diffs to PostgreSQL. Stateless.
- **mroki-hub** — the Vue 3 web UI (this surface): browse gates, inspect captured
  requests, and visualize response diffs side-by-side.
- **caddy-mroki** — a Caddy module for standalone diffing embedded in an existing
  Caddy server, no separate proxy binary or API required.

The proxy also runs in a **standalone mode** (hardcoded URLs, diffs printed
locally, no database) for local testing. Server-side diffing means a captured
pair can be re-diffed with different config without replaying traffic.

## Capabilities and Constraints

Confirmed capabilities (hub surface and the system it drives):

- **Gates** — a gate is a live/shadow service-URL pair with per-gate diff config
  and field redaction. Create, list, view, configure (gate settings), delete.
- **Requests** — browse HTTP requests captured for a gate, filter and sort them,
  and open a single request's detail.
- **Diffs** — visualize the JSON diff between the live and shadow responses
  side-by-side, with syntax highlighting; diffs are RFC 6902 JSON Patch.
- **Diff rate** — a value-driven signal (per gate and global) of how often live
  and shadow responses disagree.

Constraints and terminology:

- **JSON-only diffing** — only JSON responses are diffed; other content types are
  skipped.
- **Best-effort, never fails live** — shadow/API/diff failures are logged, never
  surfaced to the client in the live path.
- **Server-side diffing** — diff computation lives in mroki-api (the proxy stays
  a thin forwarder); the Caddy/standalone path diffs locally.
- **Vocabulary** — the interface standardizes on **gate** (live/shadow pair),
  **live** vs **shadow**, and a single **diff** noun (see the hub's `clarify`
  pass). Relations cascade on delete (deleting a gate removes its requests,
  responses, and diffs).
- **Auth** — the API is protected by an API key (`MROKI_APP_API_KEY`, min 16
  chars); the hub talks to the API over REST/JSON.

## Brand Commitments

The following are **binding** and future work must preserve them exactly:

- **Name:** `mroki`, always lowercase.
- **Tagline:** "Safe shadow traffic testing for production systems."
- **Logo & marks:** the existing assets in `docs/assets/brand/` — the
  light/dark logo banners (`mroki-logo-banner-{light,dark}.png`), the logo icons
  (`mroki-logo-icon-{light,dark}.png`), the master `mroki-logo.png`, and the
  light/dark favicons (`favicon-{light,dark}-*.png` / `.ico`). The hub renders
  dark-only and deliberately uses the dark logo icon and favicon; the light
  variants remain for other surfaces (docs, README, banners) and must be honored
  there per color scheme. Keep the full asset inventory.

## Evidence on Hand

- **Brand assets:** `docs/assets/brand/` (logos, banners, favicons — light + dark).
- **Product screenshots:** `docs/assets/screenshots/` and `docs/SCREENSHOTS.md`
  (e.g. the hub request-detail split view used in the README).
- **Documentation:** `README.md`, `docs/architecture/OVERVIEW.md` (data flow,
  data model, design decisions), `docs/api/` (REFERENCE, WALKTHROUGH),
  `docs/getting-started/`, and `docs/production/` (CONFIGURATION, SECURITY,
  MONITORING).
- **License:** MIT, © 2025 Pedro Barco — mroki is open source.
- No testimonials, customer names, benchmarks, or pricing exist; future work
  must not fabricate any.

## Product Principles

1. **Never fail live traffic.** Everything mroki does around the request is
   best-effort; the client's live response is sacrosanct.
2. **Truth comes from real traffic.** Value is validating against real
   production request patterns, not synthetic fixtures.
3. **The proxy stays thin; intelligence is centralized.** Diffing and config
   live in the API so behavior is consistent and re-runnable without replay.
4. **Meet teams where they run.** Sidecar proxy, standalone binary, or embedded
   Caddy module — adopt mroki without rewriting the app.
5. **Make the difference obvious.** The hub's job is to surface *what changed*
   clearly, so an engineer can decide to ship or hold in seconds.

## Accessibility & Inclusion

The hub has an established accessibility baseline that future work must not
regress: keyboard-operable cards/rows/back-links, correct heading order,
`prefers-reduced-motion` support, semantic ARIA roles/labels, and
touch-target sizing. Color is treated as emphasis, not the sole signal
(e.g. the diff rate always carries its numeric value alongside its color).

---

Co-authored by [Augment Code](https://www.augmentcode.com/?utm_source=git&utm_medium=commit&utm_campaign=hub_design)
