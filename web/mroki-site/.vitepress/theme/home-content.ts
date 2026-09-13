// Typed content for the custom mroki landing page (MrokiHome.vue). Single
// source of truth for the hero and its sections — kept here (not in index.md
// frontmatter or CSS vars) so the whole page is edited in one typed place.
// Copy is grounded in PRODUCT.md (mechanism, positioning, brand commitments).
//
// NB: not named `*.data.ts` on purpose — VitePress treats that suffix as a
// build-time data loader that must export a `load()`/object, which this is not.

export interface HomeAction {
  text: string
  link: string
  theme: 'brand' | 'alt'
  /** External links open in a new tab and skip base-prepending. */
  external?: boolean
}

export interface Feature {
  /** Inline SVG (viewBox 0 0 24 24, currentColor, 1.5 stroke). */
  icon: string
  title: string
  detail: string
}

export interface Hero {
  name: string
  tagline: string
  lede: string
  actions: HomeAction[]
}

/** A titled section with a supporting lede (e.g. the features heading). */
export interface Section {
  title: string
  lede: string
}

export type DiffKind = 'context' | 'removed' | 'added' | 'changed'

export interface DiffLine {
  kind: DiffKind
  text: string
}

export interface HeroDiff {
  /** Illustrative unified-diff lines — an example, not live data. */
  lines: DiffLine[]
  /** Accessible one-line summary read in place of the raw lines. */
  caption: string
}

/** One selectable diff view in the showcase, mirroring the hub's viewer. */
export interface ShowcaseView {
  /** Stable id used as the tab value and for wiring ARIA attributes. */
  id: string
  /** Segmented-control label, matching the hub UI verbatim. */
  label: string
  /** One-line explanation of what this view shows, rendered under the frame. */
  caption: string
}

/** How a single field differs between the live and shadow responses. */
export type ShowcaseChange = 'context' | 'changed' | 'added' | 'removed'

/**
 * One field of the illustrative example, rendered three ways by the switcher.
 * Values are pre-formatted display strings (quotes included for JSON strings)
 * so the renderer stays type-agnostic. This is an example, not live data.
 */
export interface ShowcaseField {
  /** Property name as it appears in the JSON. */
  key: string
  /** RFC 6902 JSON Pointer, shown in the patch view. */
  path: string
  /** How the field differs between the two responses. */
  change: ShowcaseChange
  /** Live value — set for context, changed, and removed fields. */
  live?: string
  /** Shadow value — set for context, changed, and added fields. */
  shadow?: string
}

/** A signal tone shared by the diff views, used by the showcase legend. */
export type DiffTone = 'added' | 'removed' | 'replace'

/** One entry in the showcase color legend, mapping a signal tone to a label. */
export interface LegendItem {
  tone: DiffTone
  label: string
}

export interface Showcase {
  title: string
  lede: string
  /** Selectable diff views, in the same order the hub presents them. */
  views: ShowcaseView[]
  /** Id of the view the showcase opens on. */
  defaultView: string
  /** The one example request, rendered three ways by the switcher. */
  example: ShowcaseField[]
  /** Color key rendered under the frame (recognition over recall). */
  legend: LegendItem[]
  action: HomeAction
}

/** A single line of the closing terminal illustration. */
export type TerminalLineKind = 'prompt' | 'output' | 'success'

export interface TerminalLine {
  kind: TerminalLineKind
  text: string
}

/** An illustrative terminal transcript shown in the closing band. */
export interface Terminal {
  /** Label shown in the terminal's title bar. */
  title: string
  /** Accessible one-line summary read in place of the illustrative lines. */
  caption: string
  lines: TerminalLine[]
}

export interface Closing {
  title: string
  lede: string
  actions: HomeAction[]
  /** Illustrative "bring up the stack" transcript beside the actions. */
  terminal: Terminal
}

export interface FooterLink {
  text: string
  link: string
  /** External links open in a new tab and skip base-prepending. */
  external?: boolean
}

export interface Footer {
  links: FooterLink[]
  copyright: string
}

// Line icons: single 1.5px stroke, currentColor, no fill (Control Room parity).
const iconShield = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z"/></svg>`
const iconDiff = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="18" cy="18" r="3"/><circle cx="6" cy="6" r="3"/><path d="M13 6h3a2 2 0 0 1 2 2v7"/><path d="M11 18H8a2 2 0 0 1-2-2V9"/></svg>`
const iconEye = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7Z"/><circle cx="12" cy="12" r="3"/></svg>`
const iconServer = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="8" x="2" y="2" rx="2" ry="2"/><rect width="20" height="8" x="2" y="14" rx="2" ry="2"/><line x1="6" x2="6.01" y1="6" y2="6"/><line x1="6" x2="6.01" y1="18" y2="18"/></svg>`
// Up-right arrow: marks a CTA/link that opens in a new tab.
export const iconExternal = `<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><path d="M7 7h10v10"/><path d="M7 17 17 7"/></svg>`

export const hero: Hero = {
  name: 'mroki',
  tagline: 'Safe shadow traffic testing for production systems.',
  lede: 'Mirror real production traffic to a shadow copy of your service. mroki diffs the JSON responses and shows every field that drifted, before a single user is affected.',
  actions: [
    { text: 'Get Started', link: '/docs/getting-started/FULL_STACK', theme: 'brand' },
    { text: 'API Reference', link: '/api', theme: 'alt' },
  ],
}

// Illustrative hero diff motif: a compact unified diff showing mroki's core job
// (live vs shadow, field by field) rendered with the signal palette. It mirrors
// the showcase example verbatim — the same order narrative across all three ops
// (changed `total`, removed `coupon`, added `tax`) — so the hero and the
// showcase's Unified view render the identical block. The values are an example,
// not live data — the caption says so for assistive tech.
export const heroDiff: HeroDiff = {
  lines: [
    { kind: 'context', text: '{' },
    { kind: 'context', text: '  "order": "9f2c1b",' },
    { kind: 'changed', text: '  "total": 4200,' },
    { kind: 'changed', text: '  "total": 4180,' },
    { kind: 'removed', text: '  "coupon": "SAVE10",' },
    { kind: 'added', text: '  "tax": 334' },
    { kind: 'context', text: '}' },
  ],
  caption:
    'Example response diff: compared with the live service, the shadow service changed total from 4200 to 4180, dropped the coupon field, and added a tax field.',
}

// Heading for the features grid, so the section is scannable and takes a proper
// place in the heading outline (an h2 above the h3 feature cards).
export const featuresIntro: Section = {
  title: 'How it works',
  lede: 'What happens to every request mroki mirrors.',
}

export const features: Feature[] = [
  {
    icon: iconShield,
    title: 'Live traffic comes first',
    detail:
      'Your users always get the live response. Shadow mirroring runs to the side and never delays or breaks production — every diff, API call, and error stays best-effort.',
  },
  {
    icon: iconDiff,
    title: 'Diffing lives in the API',
    detail:
      'mroki-api diffs the two responses and stores each change as an RFC 6902 patch. Re-diff later with new rules — no replay needed.',
  },
  {
    icon: iconEye,
    title: 'See what changed',
    detail:
      'Read live and shadow responses side by side in the hub, with changed fields highlighted and a diff rate showing how often they disagree.',
  },
  {
    icon: iconServer,
    title: 'Self-hosted & open source',
    detail:
      'Everything runs on your own infrastructure. No traffic leaves your network, no account to create, MIT-licensed.',
  },
]

// Product showcase: one illustrative live-vs-shadow diff, rendered three ways by
// a segmented control that mirrors the hub's own viewer — Unified, Split, and
// Patch. Built from theme-native markup (not screenshots) so it stays crisp on
// both site themes and speaks the same visual language as the hero diff. The
// example extends the hero's order narrative to cover all three RFC 6902 ops
// (replace, remove, add). Views are listed in the hub's order; the showcase
// opens on Split to pay off the section headline.
export const showcase: Showcase = {
  title: 'Live vs shadow, field by field.',
  lede: 'Every mirrored request becomes a diff you can read three ways — side by side, inline, or as a list of exact changes.',
  defaultView: 'split',
  views: [
    {
      id: 'unified',
      label: 'Unified',
      caption: 'One column, git-style — added, removed, and changed lines inline.',
    },
    {
      id: 'split',
      label: 'Split',
      caption: 'Live and shadow side by side, with changed fields highlighted.',
    },
    {
      id: 'patch',
      label: 'Patch',
      caption: 'Every change as an RFC 6902 add, remove, or replace operation.',
    },
  ],
  example: [
    { key: 'order', path: '/order', change: 'context', live: '"9f2c1b"', shadow: '"9f2c1b"' },
    { key: 'total', path: '/total', change: 'changed', live: '4200', shadow: '4180' },
    { key: 'coupon', path: '/coupon', change: 'removed', live: '"SAVE10"' },
    { key: 'tax', path: '/tax', change: 'added', shadow: '334' },
  ],
  legend: [
    { tone: 'added', label: 'Added' },
    { tone: 'removed', label: 'Removed' },
    { tone: 'replace', label: 'Replaced' },
  ],
  action: { text: 'How diffing works', link: '/docs/architecture/DIFF_ANALYSIS', theme: 'alt' },
}

// Closing call-to-action band before the footer: one clear next step (spin up
// the full stack) plus a low-commitment secondary (read the source).
export const closing: Closing = {
  title: 'See how your change behaves on real traffic.',
  lede: 'Bring up the full stack with one Docker Compose file, point a gate at your live and shadow services, and start comparing.',
  actions: [
    { text: 'Spin up the stack', link: '/docs/getting-started/FULL_STACK', theme: 'brand' },
    {
      text: 'View on GitHub',
      link: 'https://github.com/pedrobarco/mroki',
      theme: 'alt',
      external: true,
    },
  ],
  // Illustrative transcript (not live output): one Docker Compose file brings the
  // database, API, proxy, and hub online, ending on the happy-path "ready" line.
  terminal: {
    title: 'bash',
    caption:
      'Illustration: docker compose up brings the mroki database, API, proxy, and hub online, with the hub ready on localhost.',
    lines: [
      { kind: 'prompt', text: 'docker compose up' },
      { kind: 'output', text: '✔ Container mroki-db      Healthy' },
      { kind: 'output', text: '✔ Container mroki-api     Started' },
      { kind: 'output', text: '✔ Container mroki-proxy   Started' },
      { kind: 'output', text: '✔ Container mroki-hub     Started' },
      { kind: 'success', text: 'hub ready → http://localhost:5173' },
    ],
  },
}

// VitePress's default footer (themeConfig.footer) is hidden on any page with a
// sidebar and never mounts under the custom home layout, so the landing footer
// lives here in the component instead. Copyright is attributed to the project;
// the year is rendered dynamically so it never goes stale.
export const footer: Footer = {
  links: [
    { text: 'Docs', link: '/docs/getting-started/FULL_STACK' },
    { text: 'API', link: '/api' },
    { text: 'GitHub', link: 'https://github.com/pedrobarco/mroki', external: true },
  ],
  copyright: `© ${new Date().getFullYear()} mroki · MIT Licensed`,
}
