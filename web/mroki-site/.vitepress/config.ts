import { createRequire } from 'node:module'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { defineConfig } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'

const require = createRequire(import.meta.url)
const currentDir = dirname(fileURLToPath(import.meta.url))
// currentDir = web/mroki-site/.vitepress -> repo root is three levels up.
const repoRoot = resolve(currentDir, '../../..')

// https://vitepress.dev/reference/site-config
const config = withMermaid(
  defineConfig({
    title: 'mroki',
    description: 'Documentation for the mroki traffic-testing service',
    // The guide/ pages are git-ignored symlinks into the canonical docs/ tree,
    // whose relative cross-links do not resolve in this skeleton layout.
    ignoreDeadLinks: true,
    themeConfig: {
      nav: [
        { text: 'Guide', link: '/guide/overview' },
        { text: 'API', link: '/api' },
      ],
      sidebar: [
        {
          text: 'Guide',
          items: [
            { text: 'Architecture Overview', link: '/guide/overview' },
            { text: 'Full Stack Setup', link: '/guide/full-stack' },
            { text: 'Development', link: '/guide/development' },
          ],
        },
        {
          text: 'API',
          items: [{ text: 'API Reference', link: '/api' }],
        },
      ],
    },
    vite: {
      server: {
        // guide/ pages are symlinks that resolve outside the project root.
        fs: {
          allow: [repoRoot],
        },
      },
      // Workaround for vuejs/vitepress#4612: markdown sourced from outside the
      // project root needs vue resolvable from this package.
      resolve: {
        alias: {
          'vue/server-renderer': require.resolve('vue/server-renderer'),
          vue: require.resolve('vue'),
        },
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
