<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getRequests } from '@/api'
import type { Request } from '@/api'
import type { FilterState } from '@/components/requests/RequestFilters.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { ChevronRight } from 'lucide-vue-next'

interface Props {
  gateId: string
  filters: FilterState
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:total', total: number): void
  (e: 'update:showing', showing: number): void
}>()
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
    emit('update:total', total.value)
    emit('update:showing', requests.value.length)
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

const TRUNCATION_CHAR_BUDGET = 80

function smartTruncateQuery(path: string, rawQuery?: string) {
  if (!rawQuery) return { display: path, queryDisplay: '', remaining: 0 }
  const params = rawQuery.split('&')
  const budget = TRUNCATION_CHAR_BUDGET - path.length - 1 // -1 for '?'
  if (budget <= 0) {
    return { display: path, queryDisplay: '', remaining: params.length }
  }
  const visible: string[] = []
  let charCount = 0
  for (const p of params) {
    const added = charCount === 0 ? p.length : p.length + 1 // +1 for '&'
    if (charCount + added > budget && visible.length > 0) break
    visible.push(p)
    charCount += added
  }
  const remaining = params.length - visible.length
  return { display: path, queryDisplay: visible.join('&'), remaining }
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

const truncatedQueries = computed(() => {
  const map = new Map<string, ReturnType<typeof smartTruncateQuery>>()
  for (const r of requests.value) {
    map.set(r.id, smartTruncateQuery(r.path, r.raw_query))
  }
  return map
})

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
      <TooltipProvider :delay-duration="300">
        <div class="bg-card border border-border rounded-xl divide-y divide-border">
          <div
            v-for="request in requests"
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
              <div v-if="request.raw_query" class="min-w-0 flex items-center overflow-hidden">
                <code
                  class="text-xs font-mono text-foreground whitespace-nowrap overflow-hidden text-ellipsis"
                >
                  {{ truncatedQueries.get(request.id)!.display
                  }}<span
                    v-if="truncatedQueries.get(request.id)!.queryDisplay"
                    class="text-muted-foreground"
                    >?{{ truncatedQueries.get(request.id)!.queryDisplay }}</span
                  >
                </code>
                <Tooltip v-if="truncatedQueries.get(request.id)!.remaining > 0">
                  <TooltipTrigger as-child>
                    <span
                      class="inline-flex items-center text-[10px] px-1.5 py-0.5 rounded bg-accent text-muted-foreground font-mono ml-1.5 whitespace-nowrap shrink-0 cursor-default"
                    >
                      +{{ truncatedQueries.get(request.id)!.remaining }} param{{
                        truncatedQueries.get(request.id)!.remaining > 1 ? 's' : ''
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
              <code v-else class="text-xs font-mono text-foreground truncate">
                {{ request.path }}
              </code>
            </div>
            <div class="flex items-center gap-4 shrink-0 ml-4">
              <!-- Diff badge -->
              <span
                v-if="request.has_diff"
                class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-red-500/10 text-red-400"
              >
                <span class="w-1 h-1 rounded-full bg-red-400" />
                Diff
              </span>
              <span
                v-else
                class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-green-500/10 text-green-400"
              >
                No diff
              </span>
              <!-- Status codes -->
              <span class="text-xs font-mono text-dim w-24 text-right whitespace-nowrap">
                <span
                  :class="
                    (request.live_response?.status_code ?? 0) < 400
                      ? 'text-muted-foreground'
                      : 'text-danger'
                  "
                  >{{ request.live_response?.status_code ?? '—' }}</span
                >
                <span class="text-dim"> / </span>
                <span
                  :class="
                    (request.shadow_response?.status_code ?? 0) < 400
                      ? 'text-muted-foreground'
                      : 'text-danger'
                  "
                  >{{ request.shadow_response?.status_code ?? '—' }}</span
                >
              </span>
              <!-- Latency -->
              <span class="text-xs font-mono text-dim w-36 text-right whitespace-nowrap">
                {{ request.live_response?.latency_ms ?? '—' }}ms /
                {{ request.shadow_response?.latency_ms ?? '—' }}ms
              </span>
              <!-- Timestamp -->
              <div class="text-xs text-dim w-20 text-right">
                {{ formatTimestamp(request.created_at) }}
              </div>
              <ChevronRight class="h-3.5 w-3.5 text-dim/40 shrink-0" />
            </div>
          </div>
        </div>
      </TooltipProvider>

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
