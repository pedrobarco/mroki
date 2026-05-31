<script setup lang="ts">
import { computed, ref, shallowRef, watch, onMounted, onUnmounted } from 'vue'
import type { Response, PatchOp } from '@/api'
import {
  buildDiffLines,
  buildSplitRows,
  stripPathPrefix,
  expandCollapsed,
  sortArraysInTree,
} from '@/lib/json-diff'
import type { DiffLine, Token, TokenType } from '@/lib/json-diff'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { WrapText } from 'lucide-vue-next'

type ViewMode = 'unified' | 'split'
const viewMode = ref<ViewMode>('unified')
const wrapLines = ref(false)

// Synchronized horizontal scroll for split view panes
const splitLeftRef = ref<HTMLPreElement | null>(null)
const splitRightRef = ref<HTMLPreElement | null>(null)
let isSyncingScroll = false

function onSplitScroll(source: 'left' | 'right') {
  if (isSyncingScroll) return
  isSyncingScroll = true
  const from = source === 'left' ? splitLeftRef.value : splitRightRef.value
  const to = source === 'left' ? splitRightRef.value : splitLeftRef.value
  if (from && to) {
    to.scrollLeft = from.scrollLeft
  }
  requestAnimationFrame(() => {
    isSyncingScroll = false
  })
}

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
  diffContent: PatchOp[] | null
  sortArrays?: boolean
}

const props = defineProps<Props>()

function tryParseJson(str: string): unknown | null {
  try {
    return JSON.parse(str)
  } catch {
    return null
  }
}
function isBinaryContent(str: string): boolean {
  const printableRatio = (str.match(/[\x20-\x7E\n\r\t]/g) || []).length / str.length
  return printableRatio < 0.95
}

const liveBody = computed(() => props.liveResponse.body)
const shadowBody = computed(() => props.shadowResponse.body)
const isBinary = computed(
  () =>
    (liveBody.value !== null && isBinaryContent(liveBody.value)) ||
    (shadowBody.value !== null && isBinaryContent(shadowBody.value))
)
const liveJsonRaw = computed(() => (liveBody.value ? tryParseJson(liveBody.value) : null))
const shadowJsonRaw = computed(() => (shadowBody.value ? tryParseJson(shadowBody.value) : null))
const liveJson = computed(() =>
  props.sortArrays && liveJsonRaw.value !== null
    ? sortArraysInTree(liveJsonRaw.value)
    : liveJsonRaw.value
)
const shadowJson = computed(() =>
  props.sortArrays && shadowJsonRaw.value !== null
    ? sortArraysInTree(shadowJsonRaw.value)
    : shadowJsonRaw.value
)
const isJson = computed(() => liveJson.value !== null && shadowJson.value !== null)

const liveCombined = computed(() =>
  isJson.value ? { headers: props.liveResponse.headers, body: liveJson.value } : null
)
const shadowCombined = computed(() =>
  isJson.value ? { headers: props.shadowResponse.headers, body: shadowJson.value } : null
)

const combinedOps = computed(() => {
  const ops = props.diffContent ?? []
  const bodyOps = stripPathPrefix(ops, '/body').map((op) => ({
    ...op,
    path: '/body' + (op.path === '/' ? '' : op.path),
  }))
  const headerOps = stripPathPrefix(ops, '/headers').map((op) => ({
    ...op,
    path: '/headers' + (op.path === '/' ? '' : op.path),
  }))
  return [...headerOps, ...bodyOps]
})

const expandedPaths = ref(new Set<string>())
const baseDiffLines = shallowRef<DiffLine[]>([])
watch(
  [isJson, liveCombined, shadowCombined, combinedOps],
  () => {
    baseDiffLines.value =
      isJson.value && liveCombined.value && shadowCombined.value
        ? buildDiffLines(liveCombined.value, shadowCombined.value, combinedOps.value)
        : []
    expandedPaths.value = new Set()
  },
  { immediate: true }
)

const diffLines = computed(() => {
  if (expandedPaths.value.size === 0) return baseDiffLines.value
  const result = [...baseDiffLines.value]
  // Expand in reverse order so indices stay valid
  for (let i = result.length - 1; i >= 0; i--) {
    const line = result[i]
    if (line && line.type === 'collapsed' && expandedPaths.value.has(line.path)) {
      expandCollapsed(result, i)
    }
  }
  return result
})

const splitRows = computed(() => buildSplitRows(diffLines.value))

function toggleCollapsed(path: string) {
  const next = new Set(expandedPaths.value)
  if (next.has(path)) {
    next.delete(path)
  } else {
    next.add(path)
  }
  expandedPaths.value = next
}

function handleExpand(index: number) {
  const line = diffLines.value[index]
  if (line?.type === 'collapsed') toggleCollapsed(line.path)
}
function handleExpandLine(line: DiffLine) {
  if (line?.type === 'collapsed') toggleCollapsed(line.path)
}

function isExpandedRoot(line: DiffLine): boolean {
  return line.type === 'normal' && expandedPaths.value.has(line.path)
}
function handleCollapse(line: DiffLine) {
  if (expandedPaths.value.has(line.path)) toggleCollapsed(line.path)
}
const livePlain = computed(() => liveBody.value ?? '')
const shadowPlain = computed(() => shadowBody.value ?? '')
const diffCount = computed(() => props.diffContent?.length ?? 0)

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
            v-if="sortArrays"
            class="text-xs px-2 py-0.5 rounded-md font-mono bg-sky-500/15 text-sky-400"
          >
            arrays sorted
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
          <!-- Wrap toggle -->
          <div
            v-if="isJson && !isBinary"
            class="flex items-center rounded-md border border-border ml-2"
          >
            <TooltipProvider :delay-duration="300">
              <Tooltip>
                <TooltipTrigger as-child>
                  <button
                    class="inline-flex items-center gap-1 px-2 py-0.5 text-xs rounded-md transition-colors"
                    :class="
                      wrapLines ? 'bg-accent text-foreground' : 'text-dim hover:text-foreground'
                    "
                    @click="wrapLines = !wrapLines"
                  >
                    <WrapText class="size-3.5" />
                    <span>Wrap</span>
                  </button>
                </TooltipTrigger>
                <TooltipContent side="top"> Soft-wrap long lines </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
          <!-- View mode toggle (md+ only) -->
          <div
            v-if="isJson && !isBinary && isMdScreen"
            class="flex items-center rounded-md border border-border"
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
        <pre
          :key="'unified-' + wrapLines"
          :class="[
            'text-xs font-mono leading-5 p-0 m-0',
            wrapLines ? 'whitespace-pre-wrap break-words' : '',
          ]"
        ><template
            v-for="(line, idx) in diffLines"
            :key="idx"
          ><div
              v-if="line.type === 'collapsed'"
              :class="['px-4 cursor-pointer hover:bg-accent/50 transition-colors group', wrapLines ? '' : 'min-w-fit']"
              @click="handleExpand(idx)"
            ><span class="inline-block w-4 mr-2 select-none text-center text-transparent"> </span><span class="whitespace-pre">{{ '  '.repeat(line.indent) }}</span><template
                v-for="(tok, ti) in line.tokens"
                :key="ti"
              ><span :class="tokenClass(tok)">{{ tok.text }}</span></template><span class="text-zinc-600 group-hover:text-zinc-400 ml-2 text-[10px]">▸ expand</span></div><div
              v-else-if="isExpandedRoot(line)"
              :class="['px-4 cursor-pointer hover:bg-accent/50 transition-colors group', wrapLines ? '' : 'min-w-fit']"
              @click="handleCollapse(line)"
            ><span class="inline-block w-4 mr-2 select-none text-center text-transparent"> </span><span class="whitespace-pre">{{ '  '.repeat(line.indent) }}</span><template
                v-for="(tok, ti) in line.tokens"
                :key="ti"
              ><span :class="tokenClass(tok)">{{ tok.text }}</span></template><span class="text-zinc-600 group-hover:text-zinc-400 ml-2 text-[10px]">▾ collapse</span></div><div
              v-else
              :class="['px-4', lineBg(line), wrapLines ? '' : 'min-w-fit']"
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
        <div>
          <div class="grid grid-cols-2 divide-x divide-border">
            <pre
              ref="splitLeftRef"
              :key="'split-l-' + wrapLines"
              :class="[
                'text-xs font-mono leading-5 p-0 m-0 min-w-0',
                wrapLines ? 'whitespace-pre-wrap break-words overflow-hidden' : 'overflow-x-auto',
              ]"
              @scroll="onSplitScroll('left')"
            ><template
                v-for="(row, idx) in splitRows"
                :key="'l-' + idx"
              ><div
                  v-if="row.left && row.left.type === 'collapsed'"
                  :class="['px-4 cursor-pointer hover:bg-accent/50 transition-colors group', wrapLines ? '' : 'min-w-fit']"
                  @click="handleExpandLine(row.left)"
                ><span class="inline-block w-4 mr-2 select-none text-center text-transparent"> </span><span class="whitespace-pre">{{ '  '.repeat(row.left.indent) }}</span><template
                    v-for="(tok, ti) in row.left.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template><span class="text-zinc-600 group-hover:text-zinc-400 ml-2 text-[10px]">▸ expand</span></div><div
                  v-else-if="row.left && isExpandedRoot(row.left)"
                  :class="['px-4 cursor-pointer hover:bg-accent/50 transition-colors group', wrapLines ? '' : 'min-w-fit']"
                  @click="handleCollapse(row.left)"
                ><span class="inline-block w-4 mr-2 select-none text-center text-transparent"> </span><span class="whitespace-pre">{{ '  '.repeat(row.left.indent) }}</span><template
                    v-for="(tok, ti) in row.left.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template><span class="text-zinc-600 group-hover:text-zinc-400 ml-2 text-[10px]">▾ collapse</span></div><div
                  v-else-if="row.left"
                  :class="['px-4', lineBg(row.left), wrapLines ? '' : 'min-w-fit']"
                ><span
                    :class="gutterClass(row.left)"
                    class="inline-block w-4 mr-2 select-none text-center"
                  >{{ gutterChar(row.left) }}</span><span class="whitespace-pre">{{ '  '.repeat(row.left.indent) }}</span><template
                    v-for="(tok, ti) in row.left.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template></div><div
                  v-else
                  :class="['px-4 text-transparent select-none', wrapLines ? '' : 'min-w-fit']"
                >&nbsp;</div></template></pre>
            <pre
              ref="splitRightRef"
              :key="'split-r-' + wrapLines"
              :class="[
                'text-xs font-mono leading-5 p-0 m-0 min-w-0',
                wrapLines ? 'whitespace-pre-wrap break-words overflow-hidden' : 'overflow-x-auto',
              ]"
              @scroll="onSplitScroll('right')"
            ><template
                v-for="(row, idx) in splitRows"
                :key="'r-' + idx"
              ><div
                  v-if="row.right && row.right.type === 'collapsed'"
                  :class="['px-4 cursor-pointer hover:bg-accent/50 transition-colors group', wrapLines ? '' : 'min-w-fit']"
                  @click="handleExpandLine(row.right)"
                ><span class="inline-block w-4 mr-2 select-none text-center text-transparent"> </span><span class="whitespace-pre">{{ '  '.repeat(row.right.indent) }}</span><template
                    v-for="(tok, ti) in row.right.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template><span class="text-zinc-600 group-hover:text-zinc-400 ml-2 text-[10px]">▸ expand</span></div><div
                  v-else-if="row.right && isExpandedRoot(row.right)"
                  :class="['px-4 cursor-pointer hover:bg-accent/50 transition-colors group', wrapLines ? '' : 'min-w-fit']"
                  @click="handleCollapse(row.right)"
                ><span class="inline-block w-4 mr-2 select-none text-center text-transparent"> </span><span class="whitespace-pre">{{ '  '.repeat(row.right.indent) }}</span><template
                    v-for="(tok, ti) in row.right.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template><span class="text-zinc-600 group-hover:text-zinc-400 ml-2 text-[10px]">▾ collapse</span></div><div
                  v-else-if="row.right"
                  :class="['px-4', lineBg(row.right), wrapLines ? '' : 'min-w-fit']"
                ><span
                    :class="gutterClass(row.right)"
                    class="inline-block w-4 mr-2 select-none text-center"
                  >{{ gutterChar(row.right) }}</span><span class="whitespace-pre">{{ '  '.repeat(row.right.indent) }}</span><template
                    v-for="(tok, ti) in row.right.tokens"
                    :key="ti"
                  ><span :class="tokenClass(tok)">{{ tok.text }}</span></template></div><div
                  v-else
                  :class="['px-4 text-transparent select-none', wrapLines ? '' : 'min-w-fit']"
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
