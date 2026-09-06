import type { Theme } from 'vitepress'
import DefaultTheme from 'vitepress/theme'
import { theme, useOpenapi } from 'vitepress-openapi/client'
import 'vitepress-openapi/dist/style.css'
import spec from '../generated/openapi.json'

export default {
  extends: DefaultTheme,
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
