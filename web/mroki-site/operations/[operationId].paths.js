// Dynamic route loader: emits one page per OpenAPI operation so each renders on
// its own route (/operations/<operationId>) alongside the tag-grouped sidebar.
// Reads the spec produced by the bundle:openapi prebuild step.
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { markdownToPlainText, usePaths } from 'vitepress-openapi'

const currentDir = dirname(fileURLToPath(import.meta.url))
const spec = JSON.parse(
  readFileSync(resolve(currentDir, '../.vitepress/generated/openapi.json'), 'utf8')
)

export default {
  paths() {
    return usePaths({ spec })
      .getPathsByVerbs()
      .map(({ operationId, summary, description }) => ({
        params: {
          operationId,
          pageTitle: summary,
          // Lift the operation's Markdown description into a plain-text meta
          // description via transformPageData (see .vitepress/config.ts).
          description: description
            ? markdownToPlainText(description, { maxLength: 160 })
            : undefined,
        },
      }))
  },
}
