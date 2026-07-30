<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getGate, getRequest, updateGate } from '@/api'
import type { Gate, RequestDetail } from '@/api'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import DiffViewer from '@/components/diff/DiffViewer.vue'
import { ChevronLeft, Copy, Download, ChevronDown, Check } from 'lucide-vue-next'
import { truncateId } from '@/lib/utils'
import { useGateCache } from '@/composables/use-gate-cache'

const route = useRoute()
const router = useRouter()
const { getCachedGate, setGate: cacheGate } = useGateCache()

const gate = ref<Gate | null>(null)
const gateName = computed(() => gate.value?.name ?? null)
const request = ref<RequestDetail | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const copied = ref(false)

// Lightweight inline toast (no global toast primitive in the hub yet).
const toast = ref<{ message: string; type: 'success' | 'error' } | null>(null)
let toastTimer: ReturnType<typeof setTimeout> | null = null
function notify(message: string, type: 'success' | 'error' = 'success') {
  if (toastTimer) clearTimeout(toastTimer)
  toast.value = { message, type }
  toastTimer = setTimeout(() => {
    toast.value = null
  }, 3000)
}

const gateId = computed(() => route.params.id as string)
const requestId = computed(() => route.params.rid as string)

const liveResponse = computed(() => request.value?.live_response ?? null)
const shadowResponse = computed(() => request.value?.shadow_response ?? null)

const methodColors: Record<string, string> = {
  GET: 'bg-info/15 text-info',
  POST: 'bg-success/15 text-success',
  PUT: 'bg-warning/15 text-warning',
  PATCH: 'bg-warning/15 text-warning',
  DELETE: 'bg-danger/15 text-danger',
}

function getMethodClasses(method: string): string {
  return methodColors[method.toUpperCase()] || 'bg-muted text-muted-foreground'
}

const diffCount = computed(() => request.value?.diff?.content?.length ?? 0)

const TRUNCATION_CHAR_BUDGET = 80

function smartTruncateQuery(path: string, rawQuery?: string) {
  if (!rawQuery) return { display: path, queryDisplay: '', remaining: 0 }
  const params = rawQuery.split('&')
  const budget = TRUNCATION_CHAR_BUDGET - path.length - 1
  if (budget <= 0) {
    return { display: path, queryDisplay: '', remaining: params.length }
  }
  const visible: string[] = []
  let charCount = 0
  for (const p of params) {
    const added = charCount === 0 ? p.length : p.length + 1
    if (charCount + added > budget && visible.length > 0) break
    visible.push(p)
    charCount += added
  }
  const remaining = params.length - visible.length
  return { display: path, queryDisplay: visible.join('&'), remaining }
}

const truncatedQuery = computed(() => {
  if (!request.value) return null
  return smartTruncateQuery(request.value.path, request.value.raw_query)
})

async function loadRequest() {
  loading.value = true
  error.value = null

  try {
    // RequestDetail only needs the gate name/URLs — use cache if available
    const cached = getCachedGate(gateId.value)
    if (cached) {
      gate.value = cached
      const requestResponse = await getRequest(gateId.value, requestId.value)
      request.value = requestResponse.data
    } else {
      // Cache miss (e.g. direct link) — fetch both in parallel
      const [gateResponse, requestResponse] = await Promise.all([
        getGate(gateId.value),
        getRequest(gateId.value, requestId.value),
      ])
      gate.value = gateResponse.data
      cacheGate(gateResponse.data)
      request.value = requestResponse.data
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load request'
  } finally {
    loading.value = false
  }
}

async function onIgnoreField(gjsonPath: string) {
  const g = gate.value
  if (!g) return
  const ignored = g.diff_config.ignored_fields
  if (ignored.includes(gjsonPath)) return
  try {
    const res = await updateGate(g.id, {
      diff_config: { ...g.diff_config, ignored_fields: [...ignored, gjsonPath] },
    })
    gate.value = res.data
    cacheGate(res.data)
    notify(`Ignoring "${gjsonPath}" in future diffs`)
  } catch (err) {
    notify(err instanceof Error ? err.message : 'Failed to update gate', 'error')
  }
}

function goBack() {
  router.push(`/gates/${gateId.value}`)
}

function formatTimestamp(timestamp: string): string {
  return new Date(timestamp).toLocaleString()
}

function shellEscape(s: string): string {
  return s.replace(/'/g, "'\\''")
}

function buildCurl(targetUrl: string): string {
  const req = request.value
  if (!req) return ''

  const fullPath = req.raw_query ? `${req.path}?${req.raw_query}` : req.path
  const parts: string[] = [`curl -X ${req.method} '${shellEscape(targetUrl + fullPath)}'`]

  // Add request headers
  if (req.headers) {
    for (const [name, values] of Object.entries(req.headers)) {
      for (const value of values) {
        parts.push(`  -H '${shellEscape(`${name}: ${value}`)}'`)
      }
    }
  }

  // Add request body
  if (req.body) {
    parts.push(`  -d '${shellEscape(req.body)}'`)
  }

  return parts.join(' \\\n')
}

async function copyCurl(target: 'live' | 'shadow') {
  const url = target === 'live' ? gate.value?.live_url : gate.value?.shadow_url
  if (!url) return

  const curl = buildCurl(url)
  await navigator.clipboard.writeText(curl)

  copied.value = true
  setTimeout(() => {
    copied.value = false
  }, 2000)
}

function exportJson() {
  const req = request.value
  if (!req) return

  const json = JSON.stringify(req, null, 2)
  const blob = new Blob([json], { type: 'application/json' })
  const url = URL.createObjectURL(blob)

  const a = document.createElement('a')
  a.href = url
  a.download = `request-${truncateId(req.id)}.json`
  a.click()

  URL.revokeObjectURL(url)
}

onMounted(() => {
  loadRequest()
})

onUnmounted(() => {
  if (toastTimer) clearTimeout(toastTimer)
})
</script>

<template>
  <div class="max-w-6xl mx-auto px-6 py-6">
    <!-- Inline toast notification -->
    <div
      v-if="toast"
      class="fixed bottom-6 right-6 z-50 w-full max-w-sm"
      role="status"
      aria-live="polite"
    >
      <Alert
        :variant="toast.type === 'error' ? 'destructive' : 'default'"
        :class="
          toast.type === 'success'
            ? 'border-success/40 shadow-lg *:data-[slot=alert-description]:text-success'
            : 'shadow-lg'
        "
      >
        <AlertDescription>{{ toast.message }}</AlertDescription>
      </Alert>
    </div>

    <!-- Back link + breadcrumb -->
    <div class="flex items-center gap-2 mb-5">
      <button
        type="button"
        class="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
        @click="goBack"
      >
        <ChevronLeft class="h-3.5 w-3.5" />
        Back to Gate
      </button>
      <span class="text-dim text-xs">·</span>
      <span class="text-xs font-mono text-dim">{{ gateName ?? '...' }}</span>
      <span class="text-dim text-xs">·</span>
      <code class="text-xs font-mono text-dim bg-accent px-1.5 py-0.5 rounded">
        {{ truncateId(gateId) }}
      </code>
    </div>

    <!-- Page Header -->
    <div class="flex items-center justify-between mb-6">
      <h1 class="text-xl font-semibold tracking-tight">Request Detail</h1>
      <div class="flex items-center gap-2">
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <Button variant="outline" size="sm" class="gap-1.5 text-xs">
              <component :is="copied ? Check : Copy" class="h-3.5 w-3.5" />
              {{ copied ? 'Copied!' : 'Copy cURL' }}
              <ChevronDown class="h-3 w-3 ml-0.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem @click="copyCurl('live')">
              <span class="w-1.5 h-1.5 rounded-full bg-success mr-2" />
              Live endpoint
            </DropdownMenuItem>
            <DropdownMenuItem @click="copyCurl('shadow')">
              <span class="w-1.5 h-1.5 rounded-full bg-info mr-2" />
              Shadow endpoint
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button variant="outline" size="sm" class="gap-1.5 text-xs" @click="exportJson">
          <Download class="h-3.5 w-3.5" />
          Export JSON
        </Button>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="text-center py-12">
      <p class="text-muted-foreground">Loading request...</p>
    </div>

    <!-- Error State -->
    <Alert v-else-if="error" variant="destructive">
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- Request Content -->
    <div v-else-if="request" class="space-y-6">
      <!-- Request Metadata Card -->
      <div class="bg-card border border-border rounded-xl p-5">
        <TooltipProvider :delay-duration="200">
          <div class="flex items-center justify-between mb-4">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-3">
                <span
                  class="inline-flex items-center justify-center text-xs font-bold font-mono px-2.5 py-1 rounded-md tracking-wide shrink-0"
                  :class="getMethodClasses(request.method)"
                >
                  {{ request.method }}
                </span>
                <div
                  v-if="request.raw_query && truncatedQuery"
                  class="min-w-0 flex items-center overflow-hidden"
                >
                  <code
                    class="text-sm font-mono text-foreground whitespace-nowrap overflow-hidden text-ellipsis"
                  >
                    {{ truncatedQuery.display
                    }}<span v-if="truncatedQuery.queryDisplay" class="text-muted-foreground"
                      >?{{ truncatedQuery.queryDisplay }}</span
                    >
                  </code>
                  <Tooltip v-if="truncatedQuery.remaining > 0">
                    <TooltipTrigger as-child>
                      <span
                        class="inline-flex items-center text-[10px] px-1.5 py-0.5 rounded bg-accent text-muted-foreground font-mono ml-1.5 whitespace-nowrap shrink-0 cursor-default"
                      >
                        +{{ truncatedQuery.remaining }} param{{
                          truncatedQuery.remaining > 1 ? 's' : ''
                        }}
                      </span>
                    </TooltipTrigger>
                    <TooltipContent side="bottom" align="start" class="max-w-lg p-3">
                      <div class="grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
                        <template v-for="(param, i) in request.raw_query!.split('&')" :key="i">
                          <span class="text-[11px] font-mono text-muted-foreground">{{
                            param.split('=')[0]
                          }}</span>
                          <span class="text-[11px] font-mono text-foreground break-all">{{
                            param.split('=').slice(1).join('=')
                          }}</span>
                        </template>
                      </div>
                    </TooltipContent>
                  </Tooltip>
                </div>
                <code v-else class="text-sm font-mono text-foreground truncate">
                  {{ request.path }}
                </code>
              </div>
            </div>
            <div v-if="diffCount > 0" class="flex items-center gap-2 shrink-0 ml-4">
              <span
                class="inline-flex items-center gap-1.5 text-xs px-2 py-0.5 rounded-full bg-warning/15 text-warning font-medium"
              >
                {{ diffCount }} diff{{ diffCount > 1 ? 's' : '' }}
              </span>
            </div>
          </div>
        </TooltipProvider>
        <div class="grid grid-cols-2 gap-4 sm:grid-cols-4">
          <div>
            <div class="text-xs uppercase tracking-widest text-dim mb-1">Request ID</div>
            <code class="text-xs font-mono text-muted-foreground">
              {{ truncateId(request.id, 16) }}
            </code>
          </div>
          <div>
            <div class="text-xs uppercase tracking-widest text-dim mb-1">Timestamp</div>
            <span class="text-xs text-muted-foreground">
              {{ formatTimestamp(request.created_at) }}
            </span>
          </div>
          <div v-if="liveResponse">
            <div class="text-xs uppercase tracking-widest text-dim mb-1">Live Status</div>
            <div class="flex items-center gap-1.5">
              <span
                class="text-xs font-mono font-medium"
                :class="liveResponse.status_code < 400 ? 'text-success' : 'text-danger'"
              >
                {{ liveResponse.status_code }}
              </span>
              <span class="text-xs text-dim">{{ liveResponse.latency_ms }}ms</span>
            </div>
          </div>
          <div v-if="shadowResponse">
            <div class="text-xs uppercase tracking-widest text-dim mb-1">Shadow Status</div>
            <div class="flex items-center gap-1.5">
              <span
                class="text-xs font-mono font-medium"
                :class="shadowResponse.status_code < 400 ? 'text-success' : 'text-danger'"
              >
                {{ shadowResponse.status_code }}
              </span>
              <span class="text-xs text-dim">{{ shadowResponse.latency_ms }}ms</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Diff Viewer -->
      <DiffViewer
        v-if="liveResponse && shadowResponse"
        :live-response="liveResponse"
        :shadow-response="shadowResponse"
        :diff-content="request.diff.content"
        :diff-config="gate?.diff_config ?? request.diff.config"
        @ignore-field="onIgnoreField"
      />

      <!-- Missing Responses Warning -->
      <Alert v-else variant="destructive">
        <AlertTitle>Incomplete Data</AlertTitle>
        <AlertDescription>
          This request is missing {{ !liveResponse ? 'live' : 'shadow' }} response data.
        </AlertDescription>
      </Alert>
    </div>
  </div>
</template>
