// Prebuild step for the mroki docs site.
//
// 1. Bundles the multi-file OpenAPI 3.1 spec (docs/api/openapi/) into a single
//    JSON document that vitepress-openapi can consume, and also writes it to
//    public/openapi.json so the site serves the full spec as a download. Fails
//    loudly on any unresolvable $ref so a broken spec never ships silently.
// 2. (Re)creates a git-ignored, structure-preserving copy of the canonical,
//    read-only docs/ tree under this site's docs/. Copies (not symlinks) are
//    used deliberately: VitePress resolves a symlink's pageData.relativePath to
//    its realpath outside the site root, which breaks path-keyed sidebar
//    matching for the /docs/ section. Real files inside the site root resolve to
//    docs/<page>.md and match correctly. Canonical docs use relative links
//    throughout so they resolve on GitHub too; the copy step rewrites only the
//    few that point outside the copied tree. Links to repo artifacts (deployment
//    manifests, the root README, the raw OpenAPI spec) become absolute GitHub
//    URLs, and links to the generated API reference (excluded from the copy and
//    superseded by the live /api renderer) are pointed at /api.
// 3. Copies the brand assets (navbar logos + favicons) referenced by
//    .vitepress/config.ts into a git-ignored public/brand/ so VitePress emits
//    them. They are sourced from the canonical docs/assets/brand/; no copies are
//    committed under web/mroki-site.
import {
  cpSync,
  existsSync,
  mkdirSync,
  readFileSync,
  readdirSync,
  rmSync,
  statSync,
  writeFileSync,
} from 'node:fs'
import { basename, dirname, join, relative, resolve, sep } from 'node:path'
import { fileURLToPath } from 'node:url'
import SwaggerParser from '@apidevtools/swagger-parser'

const currentDir = dirname(fileURLToPath(import.meta.url))
// currentDir = web/mroki-site/.vitepress/scripts
const siteRoot = resolve(currentDir, '../..') // web/mroki-site
const repoRoot = resolve(currentDir, '../../../..') // repo root

const specEntry = resolve(repoRoot, 'docs/api/openapi/openapi.yaml')
const generatedDir = resolve(siteRoot, '.vitepress/generated')
const outFile = resolve(generatedDir, 'openapi.json')

// The same bundled spec, served verbatim as a downloadable static asset.
// VitePress copies public/ to the site root, so this is reachable at /openapi.json.
const publicDir = resolve(siteRoot, 'public')
const publicSpecFile = resolve(publicDir, 'openapi.json')

const repoDocsDir = resolve(repoRoot, 'docs') // canonical source of truth
const docsDir = resolve(siteRoot, 'docs') // git-ignored build-time copy

// Brand assets sourced from the canonical docs/assets/brand/ and emitted through
// a git-ignored public/brand/. Only the files config.ts references are copied;
// a missing source fails the build (same fail-fast rule as the link resolver).
const brandSrcDir = resolve(repoDocsDir, 'assets/brand')
const publicBrandDir = resolve(publicDir, 'brand')
const brandAssets = [
  'mroki-logo-icon-light.png',
  'mroki-logo-icon-dark.png',
  'favicon-light-16x16.png',
  'favicon-light-32x32.png',
  'favicon-dark-16x16.png',
  'favicon-dark-32x32.png',
  'favicon-light.ico',
]

// Base URL for canonical-docs links that point at repo artifacts outside the
// copied tree (deployment manifests, the root README, the raw OpenAPI spec).
// Files use /blob/, directories use /tree/.
const githubBase = 'https://github.com/pedrobarco/mroki'

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

// Semantic redirects, keyed by repo-relative POSIX path, applied to a resolved
// link target before the generic repo/GitHub fallback. The generated API
// reference is excluded from the copy and superseded by the live /api renderer,
// so links to it point at /api rather than the raw Markdown on GitHub.
const siteRouteOverrides = {
  'docs/api/REFERENCE.md': '/api',
}

const toPosix = (p) => p.split(sep).join('/')

// Rewrite a single link URL found in a copied doc. Returns null to leave the
// link untouched, or the replacement URL. Relative links are resolved against
// the source file's own directory (mirrored from the canonical docs/ tree):
//   1. a semantic override wins outright;
//   2. a target present in the copied site tree stays relative (valid on GitHub
//      and on the site);
//   3. a target absent from the copy but present in the repo becomes an absolute
//      GitHub URL (/blob/ for files, /tree/ for directories), preserving any
//      #fragment;
//   4. anything else is a dead link and fails the build.
function resolveLink(rawUrl, sourceDir, relPath) {
  const hashIndex = rawUrl.indexOf('#')
  const pathPart = hashIndex === -1 ? rawUrl : rawUrl.slice(0, hashIndex)
  const frag = hashIndex === -1 ? '' : rawUrl.slice(hashIndex)
  if (pathPart === '') return null // pure in-page anchor

  const absTarget = resolve(sourceDir, pathPart)
  const repoRel = toPosix(relative(repoRoot, absTarget))

  if (siteRouteOverrides[repoRel]) return siteRouteOverrides[repoRel]

  const docsRel = relative(repoDocsDir, absTarget)
  if (!docsRel.startsWith('..') && existsSync(join(docsDir, docsRel))) {
    return null // present in the copied tree — keep relative
  }

  if (existsSync(absTarget)) {
    const kind = statSync(absTarget).isDirectory() ? 'tree' : 'blob'
    return `${githubBase}/${kind}/main/${repoRel}${frag}`
  }

  console.error(`[bundle-openapi] unresolvable link in ${relPath}: ${rawUrl}`)
  process.exit(1)
}

// A link URL is a rewrite candidate only if it is a repo-relative path: skip
// in-page anchors (#…), site-absolute links (/…), and anything with a scheme
// (http:, https:, mailto:, …).
function isCandidate(url) {
  if (!url) return false
  if (url.startsWith('#') || url.startsWith('/')) return false
  return !/^[a-z][a-z0-9+.-]*:/i.test(url)
}

// Inline-link matcher: the URL is everything between "](" and the next ")".
// Markdown link URLs cannot contain a literal ")", so this is unambiguous for
// the inline-link forms used across the docs.
const linkPattern = /\]\(([^)]+)\)/g

function rewriteLinksInFile(relPath) {
  const file = join(docsDir, relPath)
  const sourceDir = dirname(join(repoDocsDir, relPath))
  const lines = readFileSync(file, 'utf8').split('\n')
  let inFence = false
  let changed = false

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i]
    if (/^\s*(```|~~~)/.test(line)) {
      inFence = !inFence // don't rewrite links inside fenced code blocks
      continue
    }
    if (inFence || !line.includes('](')) continue

    lines[i] = line.replace(linkPattern, (match, url) => {
      // Split off an optional title: [text](path "title").
      const spaceIndex = url.search(/\s/)
      const target = spaceIndex === -1 ? url : url.slice(0, spaceIndex)
      const title = spaceIndex === -1 ? '' : url.slice(spaceIndex)
      if (!isCandidate(target)) return match
      const replacement = resolveLink(target, sourceDir, relPath)
      if (replacement === null) return match
      changed = true
      return `](${replacement}${title})`
    })
  }

  if (changed) writeFileSync(file, lines.join('\n'))
}

function rewriteLinks() {
  const mdFiles = readdirSync(docsDir, { recursive: true }).filter(
    (p) => typeof p === 'string' && p.endsWith('.md')
  )
  for (const relPath of mdFiles) rewriteLinksInFile(relPath)
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

function copyBrandAssets() {
  if (!existsSync(brandSrcDir)) {
    console.error(`[bundle-openapi] missing brand assets dir: ${relative(repoRoot, brandSrcDir)}`)
    process.exit(1)
  }
  rmSync(publicBrandDir, { recursive: true, force: true })
  mkdirSync(publicBrandDir, { recursive: true })
  for (const name of brandAssets) {
    const src = resolve(brandSrcDir, name)
    if (!existsSync(src)) {
      console.error(`[bundle-openapi] missing brand asset: ${relative(repoRoot, src)}`)
      process.exit(1)
    }
    cpSync(src, resolve(publicBrandDir, name))
  }
  console.log(
    `[bundle-openapi] copied ${brandAssets.length} brand assets to ${relative(repoRoot, publicBrandDir)}`
  )
}

async function bundleSpec() {
  try {
    const bundled = await SwaggerParser.bundle(specEntry)
    const json = `${JSON.stringify(bundled, null, 2)}\n`
    mkdirSync(generatedDir, { recursive: true })
    writeFileSync(outFile, json)
    console.log(`[bundle-openapi] wrote ${relative(repoRoot, outFile)}`)
    // Also serve the fully-resolved spec as a static download at /openapi.json.
    mkdirSync(publicDir, { recursive: true })
    writeFileSync(publicSpecFile, json)
    console.log(`[bundle-openapi] wrote ${relative(repoRoot, publicSpecFile)}`)
  } catch (err) {
    console.error('[bundle-openapi] failed to bundle OpenAPI spec:')
    console.error(err instanceof Error ? err.message : err)
    process.exit(1)
  }
}

copyDocsTree()
copyBrandAssets()
await bundleSpec()
