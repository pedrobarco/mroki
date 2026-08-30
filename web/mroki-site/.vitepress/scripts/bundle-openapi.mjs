// Prebuild step for the mroki docs skeleton.
//
// 1. Bundles the multi-file OpenAPI 3.1 spec (docs/api/openapi/) into a single
//    JSON document that vitepress-openapi can consume. Fails loudly on any
//    unresolvable $ref so a broken spec never ships silently.
// 2. (Re)creates git-ignored copies under docs/ that surface a curated set of
//    pages from the canonical, read-only docs/ tree. Copies (not symlinks) are
//    used deliberately: VitePress resolves a symlink's pageData.relativePath to
//    its realpath outside the site root, which breaks path-keyed sidebar
//    matching for the /docs/ section. Real files inside the site root resolve to
//    docs/<page>.md and match correctly.
import { copyFileSync, existsSync, mkdirSync, rmSync, writeFileSync } from 'node:fs'
import { dirname, relative, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import SwaggerParser from '@apidevtools/swagger-parser'

const currentDir = dirname(fileURLToPath(import.meta.url))
// currentDir = web/mroki-site/.vitepress/scripts
const siteRoot = resolve(currentDir, '../..') // web/mroki-site
const repoRoot = resolve(currentDir, '../../../..') // repo root

const specEntry = resolve(repoRoot, 'docs/api/openapi/openapi.yaml')
const generatedDir = resolve(siteRoot, '.vitepress/generated')
const outFile = resolve(generatedDir, 'openapi.json')

const docsDir = resolve(siteRoot, 'docs')
// Curated skeleton pages surfaced from the canonical docs/ tree.
const docsPages = {
  'overview.md': 'docs/architecture/OVERVIEW.md',
  'full-stack.md': 'docs/getting-started/FULL_STACK.md',
  'development.md': 'docs/development/DEVELOPMENT.md',
}

function copyDocsPages() {
  rmSync(docsDir, { recursive: true, force: true })
  mkdirSync(docsDir, { recursive: true })
  for (const [linkName, target] of Object.entries(docsPages)) {
    const absTarget = resolve(repoRoot, target)
    if (!existsSync(absTarget)) {
      console.error(`[bundle-openapi] missing source doc: ${target}`)
      process.exit(1)
    }
    copyFileSync(absTarget, resolve(docsDir, linkName))
  }
  console.log(`[bundle-openapi] copied ${Object.keys(docsPages).length} docs page(s)`)
}

async function bundleSpec() {
  try {
    const bundled = await SwaggerParser.bundle(specEntry)
    mkdirSync(generatedDir, { recursive: true })
    writeFileSync(outFile, `${JSON.stringify(bundled, null, 2)}\n`)
    console.log(`[bundle-openapi] wrote ${relative(repoRoot, outFile)}`)
  } catch (err) {
    console.error('[bundle-openapi] failed to bundle OpenAPI spec:')
    console.error(err instanceof Error ? err.message : err)
    process.exit(1)
  }
}

copyDocsPages()
await bundleSpec()
