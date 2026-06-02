<script setup lang="ts">
import { computed, ref, shallowRef, watch, onMounted, onUnmounted } from 'vue'
import type { Response, PatchOp, DiffConfig } from '@/api'
import {
  buildDiffLines,
  buildSplitRows,
  stripPathPrefix,
  expandCollapsed,
  buildPatchRows,
  tokenizeJson,
} from '@/lib/json-diff'
import type { DiffLine, Token, TokenType, PatchRow } from '@/lib/json-diff'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { WrapText, SlidersHorizontal, ChevronRight, Check, ListFilter } from 'lucide-vue-next'

type ViewMode = 'unified' | 'split' | 'patch'
const MD_BREAKPOINT = 768
const isMdScreen = ref(window.innerWidth >= MD_BREAKPOINT)
// Default to split view on desktop; auto-fall back to unified below the breakpoint.
const viewMode = ref<ViewMode>(isMdScreen.value ? 'split' : 'unified')
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
  diffConfig?: DiffConfig | null
}

const props = defineProps<Props>()

// --- Diff config snapshot (settings used to compute this diff) ---
const sortArrays = computed(() => props.diffConfig?.sort_arrays ?? false)
const floatTolerance = computed(() => props.diffConfig?.float_tolerance ?? 0)
const ignoredFields = computed(() => props.diffConfig?.ignored_fields ?? [])
const includedFields = computed(() => props.diffConfig?.included_fields ?? [])

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
const liveJson = computed(() => (liveBody.value ? tryParseJson(liveBody.value) : null))
const shadowJson = computed(() => (shadowBody.value ? tryParseJson(shadowBody.value) : null))
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
const expandedPatchRows = ref(new Set<string>())
const baseDiffLines = shallowRef<DiffLine[]>([])
watch(
  [isJson, liveCombined, shadowCombined, combinedOps],
  () => {
    baseDiffLines.value =
      isJson.value && liveCombined.value && shadowCombined.value
        ? buildDiffLines(liveCombined.value, shadowCombined.value, combinedOps.value)
        : []
    expandedPaths.value = new Set()
    expandedPatchRows.value = new Set()
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

// --- Patch list view (#75) ---
type PatchFilter = 'all' | 'add' | 'remove' | 'replace'
const patchFilter = ref<PatchFilter>('all')

const patchRows = computed<PatchRow[]>(() =>
  buildPatchRows(combinedOps.value, liveCombined.value, shadowCombined.value)
)
const patchCounts = computed(() => {
  const c = { all: patchRows.value.length, add: 0, remove: 0, replace: 0 }
  for (const r of patchRows.value) c[r.op]++
  return c
})
const filteredPatchRows = computed(() =>
  patchFilter.value === 'all'
    ? patchRows.value
    : patchRows.value.filter((r) => r.op === patchFilter.value)
)
const patchFilters = computed(() => [
  {
    key: 'all' as const,
    label: 'All',
    n: patchCounts.value.all,
    active: 'bg-accent text-foreground',
  },
  {
    key: 'add' as const,
    label: 'Added',
    n: patchCounts.value.add,
    active: 'bg-green-500/15 text-green-400',
  },
  {
    key: 'remove' as const,
    label: 'Removed',
    n: patchCounts.value.remove,
    active: 'bg-red-500/15 text-red-400',
  },
  {
    key: 'replace' as const,
    label: 'Replaced',
    n: patchCounts.value.replace,
    active: 'bg-amber-500/15 text-amber-400',
  },
])

function togglePatchRow(path: string) {
  const next = new Set(expandedPatchRows.value)
  if (next.has(path)) next.delete(path)
  else next.add(path)
  expandedPatchRows.value = next
}
function isPatchRowExpanded(path: string): boolean {
  return expandedPatchRows.value.has(path)
}

interface OpMeta {
  sign: string
  abbr: string
  badge: string
  rowHover: string
}
function opMeta(op: PatchRow['op']): OpMeta {
  switch (op) {
    case 'add':
      return {
        sign: '+',
        abbr: 'ADD',
        badge: 'bg-green-500/15 text-green-400',
        rowHover: 'hover:bg-green-500/5',
      }
    case 'remove':
      return {
        sign: '−',
        abbr: 'REM',
        badge: 'bg-red-500/15 text-red-400',
        rowHover: 'hover:bg-red-500/5',
      }
    default:
      return {
        sign: '~',
        abbr: 'REP',
        badge: 'bg-amber-500/15 text-amber-400',
        rowHover: 'hover:bg-amber-500/5',
      }
  }
}

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
        <div class="flex items-center gap-2 flex-wrap">
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
          <!-- View mode toggle -->
          <div v-if="isJson && !isBinary" class="flex items-center rounded-md border border-border">
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
              v-if="isMdScreen"
              class="px-2 py-0.5 text-xs transition-colors border-l border-border"
              :class="
                viewMode === 'split'
                  ? 'bg-accent text-foreground'
                  : 'text-dim hover:text-foreground'
              "
              @click="viewMode = 'split'"
            >
              Split
            </button>
            <button
              class="px-2 py-0.5 text-xs rounded-r-md transition-colors border-l border-border"
              :class="
                viewMode === 'patch'
                  ? 'bg-accent text-foreground'
                  : 'text-dim hover:text-foreground'
              "
              @click="viewMode = 'patch'"
            >
              Patch
            </button>
          </div>
          <!-- Diff config snapshot (settings used to compute this diff) -->
          <Popover>
            <PopoverTrigger as-child>
              <button
                class="inline-flex items-center justify-center size-[26px] rounded-md border border-border text-dim hover:text-foreground hover:bg-accent data-[state=open]:bg-accent data-[state=open]:text-foreground transition-colors"
                aria-label="Diff configuration"
              >
                <SlidersHorizontal class="size-3.5" />
              </button>
            </PopoverTrigger>
            <PopoverContent align="end" :side-offset="6" class="w-80 p-0 overflow-hidden">
              <div class="px-4 py-3 border-b border-border">
                <div class="text-sm font-semibold text-foreground">Diff configuration</div>
                <div class="text-xs text-muted-foreground mt-0.5">
                  Snapshot used to compute this diff
                </div>
              </div>
              <div class="px-4 py-3 space-y-3 text-xs">
                <div class="flex items-center justify-between">
                  <span class="text-muted-foreground">Sort arrays</span>
                  <span
                    v-if="sortArrays"
                    class="px-2 py-0.5 rounded-md font-mono border border-border bg-accent/40 text-foreground"
                  >
                    On
                  </span>
                  <span v-else class="text-dim font-mono">Off</span>
                </div>
                <div class="flex items-center justify-between">
                  <span class="text-muted-foreground">Float tolerance</span>
                  <span
                    v-if="floatTolerance > 0"
                    class="px-2 py-0.5 rounded-md font-mono border border-border bg-accent/40 text-foreground"
                  >
                    ±{{ floatTolerance }}
                  </span>
                  <span v-else class="text-dim font-mono">Exact</span>
                </div>
                <div>
                  <div class="flex items-center justify-between mb-1.5">
                    <span class="text-muted-foreground">Ignored fields</span>
                    <span class="text-dim font-mono">{{ ignoredFields.length }}</span>
                  </div>
                  <div
                    v-if="ignoredFields.length > 0"
                    class="rounded-md border border-border bg-accent/40 divide-y divide-border/60 font-mono text-muted-foreground"
                  >
                    <div v-for="(field, i) in ignoredFields" :key="i" class="px-2.5 py-1 truncate">
                      {{ field }}
                    </div>
                  </div>
                  <span v-else class="text-dim font-mono">None</span>
                </div>
                <div v-if="includedFields.length > 0">
                  <div class="flex items-center justify-between mb-1.5">
                    <span class="text-muted-foreground">Included fields</span>
                    <span class="text-dim font-mono">{{ includedFields.length }}</span>
                  </div>
                  <div
                    class="rounded-md border border-border bg-accent/40 divide-y divide-border/60 font-mono text-muted-foreground"
                  >
                    <div v-for="(field, i) in includedFields" :key="i" class="px-2.5 py-1 truncate">
                      {{ field }}
                    </div>
                  </div>
                </div>
                <div v-else class="flex items-center justify-between">
                  <span class="text-dim">Included fields</span>
                  <span class="text-dim">All fields <span class="font-mono">(default)</span></span>
                </div>
              </div>
              <div class="px-4 py-2 border-t border-border text-[10px] text-muted-foreground">
                Captured from gate config · may differ from current
              </div>
            </PopoverContent>
          </Popover>
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

      <!-- Patch list view (#75) -->
      <div v-else-if="isJson && !isBinary && viewMode === 'patch'">
        <!-- Filter toolbar -->
        <div
          class="flex items-center justify-between gap-3 flex-wrap px-5 py-2.5 border-b border-border"
        >
          <div class="flex items-center gap-1.5 text-xs">
            <button
              v-for="f in patchFilters"
              :key="f.key"
              class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md border border-border transition-colors"
              :class="patchFilter === f.key ? f.active : 'text-dim hover:text-foreground'"
              @click="patchFilter = f.key"
            >
              {{ f.label }}
              <span class="text-[10px] font-mono opacity-70">{{ f.n }}</span>
            </button>
          </div>
          <div class="text-[11px] text-dim font-mono hidden sm:block">
            RFC 6902 JSON Patch · {{ filteredPatchRows.length }} shown
          </div>
        </div>

        <!-- Change rows -->
        <div v-if="filteredPatchRows.length">
          <div
            v-for="row in filteredPatchRows"
            :key="row.path"
            class="grid grid-cols-[auto_auto_1fr] items-start gap-3 px-5 py-2.5 border-b border-border/60 transition-colors"
            :class="opMeta(row.op).rowHover"
          >
            <!-- Expand chevron / spacer -->
            <button
              v-if="row.expandable"
              class="shrink-0 text-dim hover:text-foreground mt-1 transition-transform"
              :class="isPatchRowExpanded(row.path) ? 'rotate-90' : ''"
              :aria-label="isPatchRowExpanded(row.path) ? 'Collapse value' : 'Expand value'"
              @click="togglePatchRow(row.path)"
            >
              <ChevronRight class="size-3" />
            </button>
            <span v-else class="w-3 shrink-0" />

            <!-- Op badge -->
            <span
              class="shrink-0 inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-mono font-semibold mt-0.5"
              :class="opMeta(row.op).badge"
            >
              <span class="text-xs leading-none">{{ opMeta(row.op).sign }}</span>
              {{ opMeta(row.op).abbr }}
            </span>

            <!-- Path + value + expandable detail -->
            <div class="min-w-0">
              <div class="font-mono text-[12.5px] leading-tight">
                <span class="text-dim">/</span
                ><template v-for="(seg, si) in row.pathPrefix" :key="si"
                  ><span class="text-dim">{{ seg }}</span
                  ><span class="text-dim">/</span></template
                ><span
                  :class="row.leafIsIndex ? 'text-muted-foreground' : 'text-foreground font-medium'"
                  >{{ row.leafLabel }}</span
                >
              </div>
              <div
                class="font-mono text-[12.5px] leading-snug mt-1 truncate"
                :title="row.valueTitle"
              >
                <template v-if="row.op === 'replace'"
                  ><span class="line-through decoration-red-400/50"
                    ><span v-for="(t, ti) in row.oldInline" :key="ti" :class="tokenClass(t)">{{
                      t.text
                    }}</span></span
                  ><span class="text-dim mx-1.5">→</span
                  ><span
                    ><span v-for="(t, ti) in row.newInline" :key="ti" :class="tokenClass(t)">{{
                      t.text
                    }}</span></span
                  ></template
                ><template v-else-if="row.op === 'add'"
                  ><span v-for="(t, ti) in row.newInline" :key="ti" :class="tokenClass(t)">{{
                    t.text
                  }}</span></template
                ><template v-else
                  ><span class="line-through decoration-red-400/40"
                    ><span v-for="(t, ti) in row.oldInline" :key="ti" :class="tokenClass(t)">{{
                      t.text
                    }}</span></span
                  ></template
                >
              </div>

              <!-- Expanded before/after detail -->
              <div
                v-if="row.expandable && isPatchRowExpanded(row.path)"
                class="grid gap-2 mt-2 sm:grid-cols-2"
              >
                <div v-if="row.hasOld">
                  <div class="text-[10px] uppercase tracking-widest text-red-400/70 mb-1">
                    live · old
                  </div>
                  <pre
                    class="text-xs font-mono leading-relaxed whitespace-pre-wrap break-words bg-red-500/5 border border-red-500/20 rounded-md px-3 py-2 m-0"
                  ><span v-for="(t, ti) in tokenizeJson(row.oldValue, true)" :key="ti" :class="tokenClass(t)">{{ t.text }}</span></pre>
                </div>
                <div v-if="row.hasNew">
                  <div class="text-[10px] uppercase tracking-widest text-green-400/70 mb-1">
                    shadow · new
                  </div>
                  <pre
                    class="text-xs font-mono leading-relaxed whitespace-pre-wrap break-words bg-green-500/5 border border-green-500/20 rounded-md px-3 py-2 m-0"
                  ><span v-for="(t, ti) in tokenizeJson(row.newValue, true)" :key="ti" :class="tokenClass(t)">{{ t.text }}</span></pre>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Empty states -->
        <div v-else class="flex flex-col items-center justify-center text-center py-14 px-6">
          <template v-if="patchRows.length === 0">
            <div
              class="w-10 h-10 rounded-full bg-green-500/10 border border-green-500/30 flex items-center justify-center mb-3"
            >
              <Check class="size-[18px] text-green-400" />
            </div>
            <div class="text-[13px] font-medium text-muted-foreground">No differences</div>
            <div class="text-xs text-dim mt-1">
              The live and shadow responses are identical — nothing to patch.
            </div>
          </template>
          <template v-else>
            <div
              class="w-10 h-10 rounded-full bg-accent border border-border flex items-center justify-center mb-3"
            >
              <ListFilter class="size-[18px] text-dim" />
            </div>
            <div class="text-[13px] font-medium text-muted-foreground">
              No operations of this type
            </div>
            <div class="text-xs text-dim mt-1">
              Try a different filter, or switch back to “All”.
            </div>
          </template>
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
