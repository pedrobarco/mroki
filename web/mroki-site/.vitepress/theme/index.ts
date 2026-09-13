import { h, watch } from 'vue'
import type { Theme } from 'vitepress'
import { useRoute } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import { theme, useOpenapi } from 'vitepress-openapi/client'
import 'vitepress-openapi/dist/style.css'
import spec from '../generated/openapi.json'
import MrokiHome from './components/MrokiHome.vue'
import './custom.css'

// Toggles `is-search-route` on <html> per route so custom.css can scope the
// search box to where it's useful: the home page and /docs/. The API renderer
// (/api, /operations/*) opts out of the index, so its box stays hidden.
// Guarded for SSR.
const RouteAwareLayout = {
  setup() {
    if (typeof document !== 'undefined') {
      const route = useRoute()
      watch(
        () => route.path,
        (path) => {
          const showSearch = path === '/' || path.startsWith('/docs/')
          document.documentElement.classList.toggle('is-search-route', showSearch)
        },
        { immediate: true }
      )
    }
    return () => h(DefaultTheme.Layout)
  },
}

export default {
  extends: DefaultTheme,
  Layout: RouteAwareLayout,
  async enhanceApp({ app }) {
    useOpenapi({
      spec,
      config: {
        server: { allowCustomServer: true },
      },
    })
    theme.enhanceApp({ app })
    // Custom landing layout, named by index.md's `layout: MrokiHome`
    // frontmatter (rendered by VPContent while keeping nav/footer chrome).
    app.component('MrokiHome', MrokiHome)
  },
} satisfies Theme
