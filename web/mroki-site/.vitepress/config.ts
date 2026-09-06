import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'
import { useSidebar } from 'vitepress-openapi'

const currentDir = dirname(fileURLToPath(import.meta.url))

// The bundled spec is produced by the bundle:openapi prebuild step, which both
// `dev` and `build` run before VitePress. Read it here to generate a native,
// SSR-rendered API sidebar grouped by tag (the OASpec/OAOperation components
// themselves are client-only, so the sidebar cannot be derived from their DOM).
const spec = JSON.parse(readFileSync(resolve(currentDir, 'generated/openapi.json'), 'utf8'))
const apiSidebar = [
  { text: 'Reference', items: [{ text: 'Overview', link: '/api' }] },
  ...useSidebar({ spec }).generateSidebarGroups(),
]

// The documentation groups mirror the README documentation table (the IA source
// of truth). Copied doc pages are served under /docs/, preserving the canonical
// docs/ directory structure; the API Reference points at the live /api renderer,
// not a Markdown page. The same groups drive both the nav dropdowns and the
// /docs/ sidebar.
const docGroups = [
  {
    text: 'Getting Started',
    items: [
      { text: 'Full Stack', link: '/docs/getting-started/FULL_STACK' },
      { text: 'Standalone Proxy', link: '/docs/getting-started/STANDALONE_PROXY' },
      { text: 'Caddy Module', link: '/docs/getting-started/CADDY_MODULE' },
    ],
  },
  {
    text: 'Production',
    items: [
      { text: 'Docker Compose', link: '/docs/production/DOCKER_COMPOSE' },
      { text: 'Kubernetes', link: '/docs/production/KUBERNETES' },
      { text: 'Configuration', link: '/docs/production/CONFIGURATION' },
      { text: 'Security', link: '/docs/production/SECURITY' },
      { text: 'Monitoring', link: '/docs/production/MONITORING' },
    ],
  },
  {
    text: 'API',
    items: [
      { text: 'Walkthrough', link: '/docs/api/WALKTHROUGH' },
      { text: 'Reference', link: '/api' },
    ],
  },
  {
    text: 'Reference',
    items: [
      { text: 'Architecture', link: '/docs/architecture/OVERVIEW' },
      { text: 'Diff Pipeline', link: '/docs/architecture/DIFF_ANALYSIS' },
      { text: 'Troubleshooting', link: '/docs/TROUBLESHOOTING' },
      { text: 'Roadmap', link: 'https://github.com/pedrobarco/mroki/issues' },
    ],
  },
]

// https://vitepress.dev/reference/site-config
const config = withMermaid(
  defineConfig({
    title: 'mroki',
    description: 'Documentation for the mroki traffic-testing service',
    // Map the per-operation params emitted by operations/[operationId].paths.js
    // onto each dynamic page's <title> and <meta name="description">.
    transformPageData(pageData) {
      if (pageData.params?.pageTitle) {
        pageData.title = pageData.params.pageTitle
      }
      if (pageData.params?.description) {
        pageData.description = pageData.params.description
      }
    },
    // Enforce link integrity, except for the localhost dev-server URLs that the
    // canonical docs reference as content (hub, Grafana, Prometheus). VitePress
    // cannot reach those at build time; every other link is still validated.
    ignoreDeadLinks: [/^https?:\/\/localhost(:\d+)?/],
    themeConfig: {
      // Top-level nav mirrors the four README documentation groups as dropdowns;
      // the API group carries the live /api Reference entry.
      nav: docGroups,
      // Path-keyed sidebars: the copied doc pages share the four-group IA, while
      // the API Overview (/api) and the per-operation pages (/operations/*)
      // share the tag-grouped sidebar generated from the OpenAPI spec.
      sidebar: {
        '/docs/': docGroups,
        '/api': apiSidebar,
        '/operations/': apiSidebar,
      },
    },
  })
)

// withMermaid() injects mermaid's transitive deps (dayjs, cytoscape, ...) into
// optimizeDeps.include by name. Under pnpm's strict node_modules layout those
// are not resolvable from the project root, so Vite skips them and the dev
// server renders diagram pages blank. Pre-bundling `mermaid` itself lets esbuild
// resolve those deps from mermaid's own directory instead (emersonbottero/
// vitepress-plugin-mermaid#83).
config.vite ??= {}
config.vite.optimizeDeps ??= {}
config.vite.optimizeDeps.include = ['mermaid']

export default config
