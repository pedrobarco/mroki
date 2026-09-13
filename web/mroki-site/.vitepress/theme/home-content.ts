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

export type DiffKind = 'context' | 'removed' | 'added'

export interface DiffLine {
  kind: DiffKind
  text: string
}

export interface HeroDiff {
  /** Short label for the panel header (e.g. "live vs shadow"). */
  label: string
  /** Badge summarizing the change (e.g. "1 field changed"). */
  badge: string
  /** Illustrative unified-diff lines — an example, not live data. */
  lines: DiffLine[]
  /** Accessible one-line summary read in place of the raw lines. */
  caption: string
}

export interface Showcase {
  title: string
  lede: string
  /** Path under public/ (base-prepended at render time). */
  image: string
  alt: string
  action: HomeAction
}

export interface Closing {
  title: string
  lede: string
  actions: HomeAction[]
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
// (live vs shadow, field by field) rendered with the signal palette. The values
// are an example, not live data — the caption says so for assistive tech.
export const heroDiff: HeroDiff = {
  label: 'live vs shadow',
  badge: '2 fields changed',
  lines: [
    { kind: 'context', text: '{' },
    { kind: 'context', text: '  "order": "9f2c1b",' },
    { kind: 'context', text: '  "status": "confirmed",' },
    { kind: 'removed', text: '  "total": 4200,' },
    { kind: 'added', text: '  "total": 4180,' },
    { kind: 'context', text: '  "currency": "USD",' },
    { kind: 'removed', text: '  "items": 3' },
    { kind: 'added', text: '  "items": 2' },
    { kind: 'context', text: '}' },
  ],
  caption:
    'Example response diff: the shadow service returned total 4180 and items 2 where the live service returned 4200 and 3.',
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
  {
    icon: iconDiff,
    title: 'Diffing lives in the API',
    detail:
      'mroki-api diffs the two responses and stores each change as an RFC 6902 patch. Re-diff later with new rules — no replay needed.',
  },
]

// Product showcase: a real hub screenshot (the dark-only Control Room) framed
// in window chrome so the diff itself does the selling. Anchored by the
// side-by-side "Response Comparison" split view.
export const showcase: Showcase = {
  title: 'Live vs shadow, field by field.',
  lede: 'Every mirrored request becomes a side-by-side comparison, with added, removed, and changed fields highlighted inline.',
  image: '/screenshots/hub-request-detail-split.png',
  alt: 'Hub request detail: a live and shadow JSON response side by side, with changed fields highlighted.',
  action: { text: 'How diffing works', link: '/docs/architecture/DIFF_ANALYSIS', theme: 'alt' },
}

// Closing call-to-action band before the footer: one clear next step (spin up
// the full stack) plus a low-commitment secondary (read the source).
export const closing: Closing = {
  title: 'See how your change behaves on real traffic.',
  lede: 'Bring up the full stack with one Docker Compose file, point a gate at your live and shadow services, and start comparing.',
  actions: [
    { text: 'Get Started', link: '/docs/getting-started/FULL_STACK', theme: 'brand' },
    {
      text: 'View on GitHub',
      link: 'https://github.com/pedrobarco/mroki',
      theme: 'alt',
      external: true,
    },
  ],
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
