<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getRequests } from '@/api'
import type { Request } from '@/api'
import type { FilterState } from '@/components/requests/RequestFilters.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { ChevronRight } from 'lucide-vue-next'

interface Props {
  gateId: string
  filters: FilterState
}

const props = defineProps<Props>()
const router = useRouter()

const requests = ref<Request[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Pagination state
const limit = 20
const offset = ref(0)
const total = ref(0)
const hasMore = ref(false)

const currentPage = computed(() => Math.floor(offset.value / limit) + 1)
const totalPages = computed(() => Math.ceil(total.value / limit))

// Reset pagination and reload when filters change
watch(
  () => props.filters,
  () => {
    offset.value = 0
    loadRequests()
  },
  { deep: true }
)

async function loadRequests() {
  loading.value = true
  error.value = null

  try {
    const response = await getRequests(props.gateId, {
      limit,
      offset: offset.value,
      method: props.filters.methods.length > 0 ? props.filters.methods : undefined,
      path: props.filters.path || undefined,
      has_diff: props.filters.hasDiff,
      sort: props.filters.sort,
      order: props.filters.order,
    })
    requests.value = response.data
    total.value = response.pagination.total
    hasMore.value = response.pagination.has_more
  } catch (err) {
    if (err instanceof Error) {
      error.value = err.message
    } else {
      error.value = 'Failed to load requests'
    }
  } finally {
    loading.value = false
  }
}

function nextPage() {
  if (hasMore.value) {
    offset.value += limit
    loadRequests()
  }
}

function prevPage() {
  if (offset.value > 0) {
    offset.value = Math.max(0, offset.value - limit)
    loadRequests()
  }
}

function handleRequestClick(requestId: string) {
  router.push(`/gates/${props.gateId}/requests/${requestId}`)
}

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

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  if (diffMin < 1) return 'just now'
  if (diffMin < 60) return `${diffMin} min ago`
  const diffHrs = Math.floor(diffMin / 60)
  if (diffHrs < 24) return `${diffHrs}h ago`
  return date.toLocaleDateString()
}

// Dummy per-request metadata (not available in API yet)
const dummyRowMeta = [
  { diffs: 3, liveStatus: 200, shadowStatus: 200, liveLatency: 142, shadowLatency: 187 },
  { diffs: 0, liveStatus: 200, shadowStatus: 200, liveLatency: 38, shadowLatency: 42 },
  { diffs: 0, liveStatus: 200, shadowStatus: 200, liveLatency: 4, shadowLatency: 6 },
  { diffs: 1, liveStatus: 200, shadowStatus: 200, liveLatency: 89, shadowLatency: 124 },
  { diffs: 0, liveStatus: 200, shadowStatus: 200, liveLatency: 67, shadowLatency: 71 },
  { diffs: 5, liveStatus: 204, shadowStatus: 500, liveLatency: 23, shadowLatency: 312 },
]

const defaultMeta = {
  diffs: 0,
  liveStatus: 200,
  shadowStatus: 200,
  liveLatency: 0,
  shadowLatency: 0,
}

function getRowMeta(idx: number) {
  return dummyRowMeta[idx % dummyRowMeta.length] ?? defaultMeta
}

onMounted(() => {
  loadRequests()
})
</script>

<template>
  <div>
    <!-- Loading State -->
    <div v-if="loading" class="text-center py-12">
      <p class="text-muted-foreground">Loading requests...</p>
    </div>

    <!-- Error State -->
    <Alert v-else-if="error" variant="destructive">
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
      <div class="mt-4">
        <Button variant="outline" size="sm" @click="loadRequests">Retry</Button>
      </div>
    </Alert>

    <!-- Empty State -->
    <div v-else-if="requests.length === 0" class="text-center py-12">
      <p class="text-muted-foreground">
        No requests captured yet. Send traffic through this gate to see requests here.
      </p>
    </div>

    <!-- Request Rows -->
    <div v-else>
      <div class="bg-card border border-border rounded-xl overflow-hidden divide-y divide-border">
        <div
          v-for="(request, idx) in requests"
          :key="request.id"
          class="flex items-center px-5 py-3.5 cursor-pointer transition-colors hover:bg-accent"
          @click="handleRequestClick(request.id)"
        >
          <div class="flex items-center gap-3 flex-1 min-w-0">
            <span
              class="inline-flex items-center justify-center text-xs font-bold font-mono px-2 py-0.5 rounded-md tracking-wide w-14 text-center shrink-0"
              :class="getMethodClasses(request.method)"
            >
              {{ request.method }}
            </span>
            <code class="text-xs font-mono text-foreground truncate">
              {{ request.path }}
            </code>
          </div>
          <div class="flex items-center gap-4 shrink-0 ml-4">
            <!-- Diff badge -->
            <span
              v-if="getRowMeta(idx).diffs > 0"
              class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-red-500/10 text-red-400"
            >
              <span class="w-1 h-1 rounded-full bg-red-400" />
              {{ getRowMeta(idx).diffs }} diff{{ getRowMeta(idx).diffs > 1 ? 's' : '' }}
            </span>
            <span
              v-else
              class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-green-500/10 text-green-400"
            >
              No diff
            </span>
            <!-- Status codes -->
            <span class="text-xs font-mono text-dim w-16 text-right">
              <span
                :class="getRowMeta(idx).liveStatus < 400 ? 'text-muted-foreground' : 'text-danger'"
                >{{ getRowMeta(idx).liveStatus }}</span
              >
              <span class="text-dim"> / </span>
              <span
                :class="
                  getRowMeta(idx).shadowStatus < 400 ? 'text-muted-foreground' : 'text-danger'
                "
                >{{ getRowMeta(idx).shadowStatus }}</span
              >
            </span>
            <!-- Latency -->
            <span class="text-xs font-mono text-dim w-24 text-right">
              {{ getRowMeta(idx).liveLatency }}ms / {{ getRowMeta(idx).shadowLatency }}ms
            </span>
            <!-- Timestamp -->
            <div class="text-xs text-dim w-20 text-right">
              {{ formatTimestamp(request.created_at) }}
            </div>
            <ChevronRight class="h-3.5 w-3.5 text-dim/40 shrink-0" />
          </div>
        </div>
      </div>

      <!-- Pagination Controls -->
      <div class="flex items-center justify-between mt-4 text-xs">
        <span class="text-dim">Page {{ currentPage }} of {{ totalPages }}</span>
        <div class="flex items-center gap-1">
          <button
            class="px-3 py-1.5 rounded-lg border border-border bg-card text-dim transition-colors"
            :class="
              offset === 0
                ? 'opacity-40 cursor-not-allowed'
                : 'text-muted-foreground hover:bg-accent'
            "
            :disabled="offset === 0"
            @click="prevPage"
          >
            Previous
          </button>
          <span
            class="px-3 py-1.5 rounded-lg border border-border bg-accent text-foreground font-medium"
          >
            {{ currentPage }}
          </span>
          <button
            class="px-3 py-1.5 rounded-lg border border-border bg-card transition-colors"
            :class="
              !hasMore
                ? 'text-dim opacity-40 cursor-not-allowed'
                : 'text-muted-foreground hover:bg-accent'
            "
            :disabled="!hasMore"
            @click="nextPage"
          >
            Next
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
