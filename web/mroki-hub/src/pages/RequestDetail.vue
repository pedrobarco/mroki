<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getRequest } from '@/api'
import type { RequestDetail } from '@/api'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import DiffViewer from '@/components/diff/DiffViewer.vue'
import { ChevronLeft, Copy, Download } from 'lucide-vue-next'
import { truncateId } from '@/lib/utils'
import { useGateCache } from '@/composables/use-gate-cache'

const route = useRoute()
const router = useRouter()
const { getGateById } = useGateCache()

const gateName = ref<string | null>(null)
const request = ref<RequestDetail | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const gateId = computed(() => route.params.id as string)
const requestId = computed(() => route.params.rid as string)

// Find live and shadow responses
const liveResponse = computed(() => request.value?.responses.find((r) => r.type === 'live') || null)

const shadowResponse = computed(
  () => request.value?.responses.find((r) => r.type === 'shadow') || null
)

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
    const [gate, requestResponse] = await Promise.all([
      getGateById(gateId.value),
      getRequest(gateId.value, requestId.value),
    ])
    gateName.value = gate.name
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
        <Button variant="outline" size="sm" class="gap-1.5 text-xs">
          <Copy class="h-3.5 w-3.5" />
          Copy cURL
        </Button>
        <Button variant="outline" size="sm" class="gap-1.5 text-xs">
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
