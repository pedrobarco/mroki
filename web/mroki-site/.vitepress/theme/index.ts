import { h, watch } from 'vue'
import type { Theme } from 'vitepress'
import { useRoute } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import { theme, useOpenapi } from 'vitepress-openapi/client'
import 'vitepress-openapi/dist/style.css'
import spec from '../generated/openapi.json'
import './custom.css'

// Toggles `is-docs-route` on <html> per route so custom.css can scope the
// search box to /docs/. Guarded for SSR.
const RouteAwareLayout = {
  setup() {
    if (typeof document !== 'undefined') {
      const route = useRoute()
      watch(
        () => route.path,
        (path) => {
          document.documentElement.classList.toggle('is-docs-route', path.startsWith('/docs/'))
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
  },
} satisfies Theme
