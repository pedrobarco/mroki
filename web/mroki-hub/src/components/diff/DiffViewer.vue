<script setup lang="ts">
import { computed } from 'vue'
import type { Response, PatchOp } from '@/api'
import { buildDiffLines, stripPathPrefix } from '@/lib/json-diff'
import type { DiffLine } from '@/lib/json-diff'

interface Props {
  liveResponse: Response
  shadowResponse: Response
  diffContent: PatchOp[]
}

const props = defineProps<Props>()

// Decode base64 body, return null if not decodable
function decodeBody(body: string): string | null {
  try {
    return atob(body)
  } catch {
    return null
  }
}

// Try to parse JSON, return null if not valid
function tryParseJson(str: string): unknown | null {
  try {
    return JSON.parse(str)
  } catch {
    return null
  }
}

// Check if decoded content is binary
function isBinaryContent(decoded: string): boolean {
  const printableRatio = (decoded.match(/[\x20-\x7E\n\r\t]/g) || []).length / decoded.length
  return printableRatio < 0.95
}

// Compute decoded/parsed body data
const liveDecoded = computed(() => decodeBody(props.liveResponse.body))
const shadowDecoded = computed(() => decodeBody(props.shadowResponse.body))

const isBinary = computed(
  () =>
    (liveDecoded.value !== null && isBinaryContent(liveDecoded.value)) ||
    (shadowDecoded.value !== null && isBinaryContent(shadowDecoded.value))
)

const liveJson = computed(() => (liveDecoded.value ? tryParseJson(liveDecoded.value) : null))
const shadowJson = computed(() => (shadowDecoded.value ? tryParseJson(shadowDecoded.value) : null))
const isJson = computed(() => liveJson.value !== null && shadowJson.value !== null)

// Strip /body prefix from ops that target the response body
const bodyOps = computed(() => stripPathPrefix(props.diffContent, '/body'))

// Build annotated diff lines from the pre-computed PatchOps
const diffLines = computed<DiffLine[]>(() => {
  if (!isJson.value) return []
  return buildDiffLines(liveJson.value, shadowJson.value, bodyOps.value)
})

// For non-JSON content, fall back to plain text display
const livePlain = computed(() => liveDecoded.value ?? props.liveResponse.body)
const shadowPlain = computed(() => shadowDecoded.value ?? props.shadowResponse.body)

// Count of diff operations
const diffCount = computed(() => props.diffContent.length)

// Line type to CSS class mapping
function lineClass(line: DiffLine): string {
  switch (line.type) {
    case 'added':
      return 'bg-green-500/10 text-green-300'
    case 'removed':
      return 'bg-red-500/10 text-red-300'
    default:
      return 'text-foreground/80'
  }
}

// Line type to gutter indicator
function gutterChar(line: DiffLine): string {
  switch (line.type) {
    case 'added':
      return '+'
    case 'removed':
      return '−'
    default:
      return ' '
  }
}

function gutterClass(line: DiffLine): string {
  switch (line.type) {
    case 'added':
      return 'text-green-400'
    case 'removed':
      return 'text-red-400'
    default:
      return 'text-transparent'
  }
}
</script>

<template>
  <div>
    <div class="bg-card border border-border rounded-xl overflow-hidden">
      <!-- Card Header -->
      <div class="flex items-center justify-between px-5 py-3.5 border-b border-border">
        <div class="flex items-center gap-2">
          <h3 class="text-sm font-semibold">Response Comparison</h3>
          <span class="text-xs text-dim bg-accent px-2 py-0.5 rounded-md font-mono">
            {{ isJson ? 'json' : 'text' }}
          </span>
          <span
            v-if="diffCount > 0"
            class="text-xs px-2 py-0.5 rounded-md font-mono bg-amber-500/15 text-amber-400"
          >
            {{ diffCount }} change{{ diffCount > 1 ? 's' : '' }}
          </span>
          <span
            v-else
            class="text-xs px-2 py-0.5 rounded-md font-mono bg-green-500/15 text-green-400"
          >
            identical
          </span>
        </div>
        <div class="flex items-center gap-3 text-xs">
          <div class="flex items-center gap-1.5 text-dim">
            <span class="w-2.5 h-2.5 rounded-sm bg-red-500/10 border border-red-500/30" />
            Removed
          </div>
          <div class="flex items-center gap-1.5 text-dim">
            <span class="w-2.5 h-2.5 rounded-sm bg-green-500/10 border border-green-500/30" />
            Added
          </div>
        </div>
      </div>

      <!-- JSON Diff View -->
      <div v-if="isJson && !isBinary" class="overflow-x-auto">
        <pre class="text-xs font-mono leading-5 p-0 m-0"><template
            v-for="(line, idx) in diffLines"
            :key="idx"
          ><div
              :class="lineClass(line)"
              class="px-4 min-w-fit"
            ><span
                :class="gutterClass(line)"
                class="inline-block w-4 mr-2 select-none text-center"
              >{{ gutterChar(line) }}</span><span
                class="whitespace-pre"
              >{{ '  '.repeat(line.indent) }}{{ line.content }}</span></div></template></pre>
      </div>

      <!-- Non-JSON fallback: plain text side-by-side -->
      <div v-else-if="!isBinary" class="grid grid-cols-2 divide-x divide-border">
        <div class="p-4">
          <div class="text-xs uppercase tracking-widest text-dim mb-2">Live</div>
          <pre class="text-xs font-mono whitespace-pre-wrap text-foreground/80">{{
            livePlain
          }}</pre>
        </div>
        <div class="p-4">
          <div class="text-xs uppercase tracking-widest text-dim mb-2">Shadow</div>
          <pre class="text-xs font-mono whitespace-pre-wrap text-foreground/80">{{
            shadowPlain
          }}</pre>
        </div>
      </div>

      <!-- Binary fallback -->
      <div v-else class="text-center py-8 text-muted-foreground">
        <p class="text-sm">Binary content cannot be displayed in diff viewer.</p>
      </div>
    </div>
  </div>
</template>
