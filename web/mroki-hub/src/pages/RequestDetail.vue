<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getRequest } from '@/api'
import type { Gate, RequestDetail } from '@/api'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import DiffViewer from '@/components/diff/DiffViewer.vue'
import { ChevronLeft, Copy, Download, ChevronDown, Check } from 'lucide-vue-next'
import { truncateId } from '@/lib/utils'
import { useGateCache } from '@/composables/use-gate-cache'

const route = useRoute()
const router = useRouter()
const { getGateById } = useGateCache()

const gate = ref<Gate | null>(null)
const gateName = computed(() => gate.value?.name ?? null)
const request = ref<RequestDetail | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const copied = ref(false)

const gateId = computed(() => route.params.id as string)
const requestId = computed(() => route.params.rid as string)

const liveResponse = computed(() => request.value?.live_response ?? null)
const shadowResponse = computed(() => request.value?.shadow_response ?? null)

const methodColors: Record<string, string> = {
  GET: 'bg-blue-500/15 text-blue-400',
  POST: 'bg-green-500/15 text-green-400',
  PUT: 'bg-amber-500/15 text-amber-400',
  PATCH: 'bg-amber-500/15 text-amber-400',
  DELETE: 'bg-red-500/15 text-red-400',
}

function getMethodClasses(method: string): string {
  return methodColors[method.toUpperCase()] || 'bg-muted text-muted-foreground'
}

const diffCount = computed(() => request.value?.diff?.content?.length ?? 0)

async function loadRequest() {
  loading.value = true
  error.value = null

  try {
    const [gateData, requestResponse] = await Promise.all([
      getGateById(gateId.value),
      getRequest(gateId.value, requestId.value),
    ])
    gate.value = gateData
    request.value = requestResponse.data
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load request'
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push(`/gates/${gateId.value}`)
}

function formatTimestamp(timestamp: string): string {
  return new Date(timestamp).toLocaleString()
}

function buildCurl(targetUrl: string): string {
  const req = request.value
  if (!req) return ''

  const parts: string[] = [`curl -X ${req.method} '${targetUrl}${req.path}'`]

  // Add request headers
  if (req.headers) {
    for (const [name, values] of Object.entries(req.headers)) {
      for (const value of values) {
        parts.push(`  -H '${name}: ${value}'`)
      }
    }
  }

  // Add request body (base64-decoded)
  if (req.body) {
    try {
      const decoded = atob(req.body)
      parts.push(`  -d '${decoded.replace(/'/g, "'\\''")}'`)
    } catch {
      parts.push(`  --data-binary '${req.body}'`)
    }
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
</script>

<template>
  <div class="max-w-6xl mx-auto px-6 py-6">
    <!-- Back link + breadcrumb -->
    <div class="flex items-center gap-2 mb-5">
      <a
        class="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors cursor-pointer"
        @click="goBack"
      >
        <ChevronLeft class="h-3.5 w-3.5" />
        Back to Gate
      </a>
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
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-3">
            <span
              class="inline-flex items-center justify-center text-xs font-bold font-mono px-2.5 py-1 rounded-md tracking-wide"
              :class="getMethodClasses(request.method)"
            >
              {{ request.method }}
            </span>
            <code class="text-sm font-mono text-foreground">{{ request.path }}</code>
          </div>
          <div v-if="diffCount > 0" class="flex items-center gap-2">
            <span
              class="inline-flex items-center gap-1.5 text-xs px-2 py-0.5 rounded-full bg-amber-500/15 text-amber-400 font-medium"
            >
              {{ diffCount }} diff{{ diffCount > 1 ? 's' : '' }} found
            </span>
          </div>
        </div>
        <div class="grid grid-cols-4 gap-4">
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
