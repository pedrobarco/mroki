// Prebuild step for the mroki docs site.
//
// 1. Bundles the multi-file OpenAPI 3.1 spec (docs/api/openapi/) into a single
//    JSON document that vitepress-openapi can consume. Fails loudly on any
//    unresolvable $ref so a broken spec never ships silently.
// 2. (Re)creates a git-ignored, structure-preserving copy of the canonical,
//    read-only docs/ tree under this site's docs/. Copies (not symlinks) are
//    used deliberately: VitePress resolves a symlink's pageData.relativePath to
//    its realpath outside the site root, which breaks path-keyed sidebar
//    matching for the /docs/ section. Real files inside the site root resolve to
//    docs/<page>.md and match correctly. A handful of links that point outside
//    the copied tree (or at content excluded from the copy) are rewritten in
//    place afterwards so the site builds with dead-link checking enabled.
import { cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { basename, dirname, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import SwaggerParser from '@apidevtools/swagger-parser'

const currentDir = dirname(fileURLToPath(import.meta.url))
// currentDir = web/mroki-site/.vitepress/scripts
const siteRoot = resolve(currentDir, '../..') // web/mroki-site
const repoRoot = resolve(currentDir, '../../../..') // repo root

const specEntry = resolve(repoRoot, 'docs/api/openapi/openapi.yaml')
const generatedDir = resolve(siteRoot, '.vitepress/generated')
const outFile = resolve(generatedDir, 'openapi.json')

const repoDocsDir = resolve(repoRoot, 'docs') // canonical source of truth
const docsDir = resolve(siteRoot, 'docs') // git-ignored build-time copy

// Base URL for links that must resolve on GitHub rather than inside the site
// (source files outside docs/, and the raw OpenAPI spec).
const githubBlob = 'https://github.com/pedrobarco/mroki/blob/main'

// Paths (relative to docs/) excluded from the copy. The raw OpenAPI spec is
// bundled separately by bundleSpec(); the generated API reference is superseded
// by the live /api renderer and would otherwise ship as a huge duplicate page.
const excludedPaths = ['api/openapi', 'api/REFERENCE.md']
const excludedNames = new Set(['.DS_Store', '.gitkeep'])

function shouldCopy(src) {
  const rel = relative(repoDocsDir, src)
  if (rel === '') return true // the docs/ root itself
  if (excludedNames.has(basename(src))) return false
  return !excludedPaths.some((ex) => rel === ex || rel.startsWith(ex + sep))
}

// Post-copy link rewrites, keyed by path relative to docs/. Each replacement is
// a literal [find, replace] pair applied to the copied file only; canonical
// docs/ stays untouched. A find string that no longer matches fails the build,
// so these can't silently rot when the source docs change.
const linkRewrites = {
  'production/KUBERNETES.md': [
    ['](../../deployments/kubernetes/', `](${githubBlob}/deployments/kubernetes/`],
  ],
  'api/WALKTHROUGH.md': [
    ['](REFERENCE.md)', '](/api)'],
    ['](openapi/openapi.yaml)', `](${githubBlob}/docs/api/openapi/openapi.yaml)`],
  ],
  'architecture/OVERVIEW.md': [['](../api/REFERENCE.md)', '](/api)']],
  'development/CONTRIBUTING.md': [['](./README.md)', `](${githubBlob}/README.md)`]],
}

function rewriteLinks() {
  for (const [relPath, replacements] of Object.entries(linkRewrites)) {
    const file = resolve(docsDir, relPath)
    let content = readFileSync(file, 'utf8')
    for (const [find, replace] of replacements) {
      if (!content.includes(find)) {
        console.error(`[bundle-openapi] link rewrite no longer matches in ${relPath}: ${find}`)
        process.exit(1)
      }
      content = content.replaceAll(find, replace)
    }
    writeFileSync(file, content)
  }
}

function copyDocsTree() {
  if (!existsSync(repoDocsDir)) {
    console.error(`[bundle-openapi] missing canonical docs dir: ${relative(repoRoot, repoDocsDir)}`)
    process.exit(1)
  }
  rmSync(docsDir, { recursive: true, force: true })
  cpSync(repoDocsDir, docsDir, { recursive: true, filter: shouldCopy })
  rewriteLinks()
  console.log(`[bundle-openapi] copied docs/ tree to ${relative(repoRoot, docsDir)}`)
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

copyDocsTree()
await bundleSpec()
