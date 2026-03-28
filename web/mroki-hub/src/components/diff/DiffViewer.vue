<script setup lang="ts">
import { computed, ref, shallowRef, watch, onMounted, onUnmounted } from 'vue'
import type { Response, PatchOp } from '@/api'
import { buildDiffLines, buildSplitRows, stripPathPrefix, expandCollapsed } from '@/lib/json-diff'
import type { DiffLine, Token, TokenType } from '@/lib/json-diff'

type ViewMode = 'unified' | 'split'
const viewMode = ref<ViewMode>('unified')

const MD_BREAKPOINT = 768
const isMdScreen = ref(window.innerWidth >= MD_BREAKPOINT)

function onResize() {
  isMdScreen.value = window.innerWidth >= MD_BREAKPOINT
  if (!isMdScreen.value && viewMode.value === 'split') {
    viewMode.value = 'unified'
  }
}

onMounted(() => window.addEventListener('resize', onResize))
onUnmounted(() => window.removeEventListener('resize', onResize))

interface Props {
  liveResponse: Response
  shadowResponse: Response
  diffContent: PatchOp[]
}

const props = defineProps<Props>()

function decodeBody(body: string): string | null {
  try {
    return atob(body)
  } catch {
    return null
  }
}
function tryParseJson(str: string): unknown | null {
  try {
    return JSON.parse(str)
  } catch {
    return null
  }
}
function isBinaryContent(decoded: string): boolean {
  const printableRatio = (decoded.match(/[\x20-\x7E\n\r\t]/g) || []).length / decoded.length
  return printableRatio < 0.95
}

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

const bodyOps = computed(() => stripPathPrefix(props.diffContent, '/body'))
const diffLines = shallowRef<DiffLine[]>([])
watch(
  [isJson, liveJson, shadowJson, bodyOps],
  () => {
    diffLines.value = isJson.value
      ? buildDiffLines(liveJson.value, shadowJson.value, bodyOps.value)
      : []
  },
  { immediate: true }
)
const splitRows = computed(() => buildSplitRows(diffLines.value))

function handleExpand(index: number) {
  expandCollapsed(diffLines.value, index)
  // Trigger reactivity on shallowRef by replacing the array reference
  diffLines.value = [...diffLines.value]
}
function handleExpandLine(line: DiffLine) {
  const idx = diffLines.value.indexOf(line)
  if (idx >= 0) handleExpand(idx)
}
const livePlain = computed(() => liveDecoded.value ?? props.liveResponse.body)
const shadowPlain = computed(() => shadowDecoded.value ?? props.shadowResponse.body)
const diffCount = computed(() => props.diffContent.length)

// --- Line styling (background strip) ---
function lineBg(line: DiffLine): string {
  switch (line.type) {
    case 'added':
      return 'bg-green-500/10'
    case 'removed':
      return 'bg-red-500/10'
    case 'replaced-old':
      return 'bg-amber-500/10'
    case 'replaced-new':
      return 'bg-amber-500/10'
    default:
      return ''
  }
}
function gutterChar(line: DiffLine): string {
  switch (line.type) {
    case 'added':
      return '+'
    case 'removed':
      return '−'
    case 'replaced-old':
    case 'replaced-new':
      return '~'
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
    case 'replaced-old':
    case 'replaced-new':
      return 'text-amber-400'
    default:
      return 'text-transparent'
  }
}

// --- Syntax token coloring ---
function tokenClass(token: Token): string {
  const colorMap: Record<TokenType, string> = {
    key: 'text-sky-400',
    string: 'text-violet-400',
    number: 'text-emerald-400',
    boolean: 'text-orange-400',
    null: 'text-pink-400',
    bracket: 'text-zinc-500',
    punctuation: 'text-zinc-500',
  }
  return colorMap[token.type]
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
          <div class="flex items-center gap-1.5 text-dim">
            <span class="w-2.5 h-2.5 rounded-sm bg-amber-500/10 border border-amber-500/30" />
            Changed
          </div>
          <!-- View mode toggle (md+ only) -->
          <div
            v-if="isJson && !isBinary && isMdScreen"
            class="flex items-center rounded-md border border-border ml-2"
          >
            <button
              class="px-2 py-0.5 text-xs rounded-l-md transition-colors"
              :class="
                viewMode === 'unified'
                  ? 'bg-accent text-foreground'
                  : 'text-dim hover:text-foreground'
              "
              @click="viewMode = 'unified'"
            >
              Unified
            </button>
            <button
              class="px-2 py-0.5 text-xs rounded-r-md transition-colors border-l border-border"
              :class="
                viewMode === 'split'
                  ? 'bg-accent text-foreground'
                  : 'text-dim hover:text-foreground'
              "
              @click="viewMode = 'split'"
            >
              Split
            </button>
          </div>
        </div>
      </div>

      <!-- Unified JSON Diff View -->
      <div v-if="isJson && !isBinary && viewMode === 'unified'" class="overflow-x-auto">
        <pre class="text-xs font-mono leading-5 p-0 m-0"><template
            v-for="(line, idx) in diffLines"
            :key="idx"
          ><div
              v-if="line.type === 'collapsed'"
              class="px-4 min-w-fit cursor-pointer hover:bg-accent/50 transition-colors group"
              @click="handleExpand(idx)"
            ><span class="inline-block w-4 mr-2 select-none text-center text-transparent"> </span><span class="whitespace-pre">{{ '  '.repeat(line.indent) }}</span><template
                v-for="(tok, ti) in line.tokens"
                :key="ti"
              ><span :class="tokenClass(tok)">{{ tok.text }}</span></template><span class="text-zinc-600 group-hover:text-zinc-400 ml-2 text-[10px]">▸ expand</span></div><div
              v-else
              :class="lineBg(line)"
              class="px-4 min-w-fit"
            ><span
                :class="gutterClass(line)"
                class="inline-block w-4 mr-2 select-none text-center"
              >{{ gutterChar(line) }}</span><span class="whitespace-pre">{{ '  '.repeat(line.indent) }}</span><template
                v-for="(tok, ti) in line.tokens"
                :key="ti"
              ><span :class="tokenClass(tok)">{{ tok.text }}</span></template></div></template></pre>
      </div>

      <!-- Split JSON Diff View -->
      <div v-else-if="isJson && !isBinary && viewMode === 'split'">
        <div class="grid grid-cols-2 border-b border-border">
          <div class="px-4 py-2 text-xs uppercase tracking-widest text-dim border-r border-border">
            <span class="w-1.5 h-1.5 rounded-full bg-success inline-block mr-1.5" />
            Live
          </div>
          <div class="px-4 py-2 text-xs uppercase tracking-widest text-dim">
            <span class="w-1.5 h-1.5 rounded-full bg-info inline-block mr-1.5" />
            Shadow
          </div>
        </div>
        <div class="overflow-x-auto">
          <div class="grid grid-cols-2 divide-x divide-border">
            <pre class="text-xs font-mono leading-5 p-0 m-0"><template
                v-for="(row, idx) in splitRows"
                :key="'l-' + idx"
              ><div
                  v-if="row.left && row.left.type === 'collapsed'"
                  class="px-4 min-w-fit cursor-pointer hover:bg-accent/50 transition-colors group"
                  @click="handleExpandLine(row.left)"
                ><span class="inline-block w-4 mr-2 select-none text-center text-transparent"> </span><span class="whitespace-pre">{{ '  '.repeat(row.left.indent) }}</span><template
                    v-for="(tok, ti) in row.left.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template><span class="text-zinc-600 group-hover:text-zinc-400 ml-2 text-[10px]">▸ expand</span></div><div
                  v-else-if="row.left"
                  :class="lineBg(row.left)"
                  class="px-4 min-w-fit"
                ><span
                    :class="gutterClass(row.left)"
                    class="inline-block w-4 mr-2 select-none text-center"
                  >{{ gutterChar(row.left) }}</span><span class="whitespace-pre">{{ '  '.repeat(row.left.indent) }}</span><template
                    v-for="(tok, ti) in row.left.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template></div><div
                  v-else
                  class="px-4 min-w-fit text-transparent select-none"
                >&nbsp;</div></template></pre>
            <pre class="text-xs font-mono leading-5 p-0 m-0"><template
                v-for="(row, idx) in splitRows"
                :key="'r-' + idx"
              ><div
                  v-if="row.right && row.right.type === 'collapsed'"
                  class="px-4 min-w-fit cursor-pointer hover:bg-accent/50 transition-colors group"
                  @click="handleExpandLine(row.right)"
                ><span class="inline-block w-4 mr-2 select-none text-center text-transparent"> </span><span class="whitespace-pre">{{ '  '.repeat(row.right.indent) }}</span><template
                    v-for="(tok, ti) in row.right.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template><span class="text-zinc-600 group-hover:text-zinc-400 ml-2 text-[10px]">▸ expand</span></div><div
                  v-else-if="row.right"
                  :class="lineBg(row.right)"
                  class="px-4 min-w-fit"
                ><span
                    :class="gutterClass(row.right)"
                    class="inline-block w-4 mr-2 select-none text-center"
                  >{{ gutterChar(row.right) }}</span><span class="whitespace-pre">{{ '  '.repeat(row.right.indent) }}</span><template
                    v-for="(tok, ti) in row.right.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template></div><div
                  v-else
                  class="px-4 min-w-fit text-transparent select-none"
                >&nbsp;</div></template></pre>
          </div>
        </div>
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
