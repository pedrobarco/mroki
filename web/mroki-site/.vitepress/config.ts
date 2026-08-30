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
    // The docs/ pages are git-ignored copies of the canonical docs/ tree,
    // whose relative cross-links do not resolve in this skeleton layout.
    ignoreDeadLinks: true,
    themeConfig: {
      nav: [
        { text: 'Docs', link: '/docs/overview' },
        { text: 'API', link: '/api' },
      ],
      // Path-keyed sidebars: the docs pages keep their own nav, while the API
      // Overview (/api) and the per-operation pages (/operations/*) share the
      // tag-grouped sidebar generated from the OpenAPI spec.
      sidebar: {
        '/docs/': [
          {
            text: 'Docs',
            items: [
              { text: 'Architecture Overview', link: '/docs/overview' },
              { text: 'Full Stack Setup', link: '/docs/full-stack' },
              { text: 'Development', link: '/docs/development' },
            ],
          },
        ],
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
