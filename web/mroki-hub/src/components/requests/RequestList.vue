<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useQuery, keepPreviousData } from '@tanstack/vue-query'
import { useVueTable, getCoreRowModel } from '@tanstack/vue-table'
import type { ColumnDef, PaginationState, SortingState, Updater } from '@tanstack/vue-table'
import { requestsQuery } from '@/api'
import type { ListRequestsParams, Request } from '@/api'
import type { FilterState } from '@/components/requests/RequestFilters.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import Pagination from '@/components/common/Pagination.vue'
import { methodColorClass, formatLatency } from '@/lib/utils'
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

// Pagination lives in a TanStack Table state object (pageIndex/pageSize) and is
// the single source of paging truth; `offset` is derived from it and feeds the
// query key, so paging fetches (or serves cached) the matching page.
const pageSize = 20
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize })
const offset = computed(() => pagination.value.pageIndex * pagination.value.pageSize)

const queryParams = computed<ListRequestsParams>(() => ({
  limit: pageSize,
  offset: offset.value,
  method: props.filters.methods.length > 0 ? props.filters.methods : undefined,
  path: props.filters.path || undefined,
  has_diff: props.filters.hasDiff,
  sort: props.filters.sort,
  order: props.filters.order,
}))

// Reset to the first page whenever the filter set changes; the resulting key
// change triggers the refetch.
watch(
  () => props.filters,
  () => {
    pagination.value = { pageIndex: 0, pageSize }
  },
  { deep: true }
)

// keepPreviousData holds the current page while the next one loads, preventing a
// loading flash and out-of-order flicker during rapid paging.
const query = useQuery(
  computed(() => ({
    ...requestsQuery(props.gateId, queryParams.value),
    placeholderData: keepPreviousData,
  }))
)

const requests = computed(() => query.data.value?.data ?? [])
const total = computed(() => query.data.value?.pagination.total ?? 0)
const hasMore = computed(() => query.data.value?.pagination.has_more ?? false)
const loading = computed(() => query.isPending.value)
const error = computed(() =>
  query.isError.value ? (query.error.value?.message ?? 'Failed to load requests') : null
)

function loadRequests() {
  query.refetch()
}

// Bubble the list totals up to the parent header each time a page resolves.
watch(
  () => query.data.value,
  (data) => {
    if (!data) return
    emit('update:total', data.pagination.total)
    emit('update:showing', data.data.length)
  },
  { immediate: true }
)

// Sorting is owned by the filter bar (RequestFilters) and sent to the server, so
// we mirror it into the table's sorting state read-only — the row layout has no
// clickable column headers to drive it.
const sorting = computed<SortingState>(() => [
  { id: props.filters.sort, desc: props.filters.order === 'desc' },
])

// Minimal column defs: the list renders custom rows, not table cells, so columns
// only need accessor keys matching the server-sortable fields to back the row
// model. The generated ui/table primitives don't fit the row/mobile layout, so
// vue-table is adopted headlessly (state only) rather than for rendering.
const columns: ColumnDef<Request>[] = [
  { accessorKey: 'method' },
  { accessorKey: 'path' },
  { accessorKey: 'created_at' },
]

// Manual (server-driven) mode: the server owns paging and sorting, so the table
// is a headless state layer over the fetched page. `rowCount` comes from the
// server total so getPageCount() matches the real page count.
const table = useVueTable({
  data: requests,
  columns,
  getCoreRowModel: getCoreRowModel(),
  manualPagination: true,
  manualSorting: true,
  get rowCount() {
    return total.value
  },
  state: {
    get pagination() {
      return pagination.value
    },
    get sorting() {
      return sorting.value
    },
  },
  onPaginationChange: (updater: Updater<PaginationState>) => {
    pagination.value = typeof updater === 'function' ? updater(pagination.value) : updater
  },
  // Filters own sorting; the table never mutates it, so this no-op keeps the
  // controlled sorting state without a "missing onSortingChange" warning.
  onSortingChange: () => {},
})

const currentPage = computed(() => table.getState().pagination.pageIndex + 1)
const totalPages = computed(() => table.getPageCount())

function nextPage() {
  table.nextPage()
}

function prevPage() {
  table.previousPage()
}

function handleRequestClick(requestId: string) {
  router.push(`/gates/${props.gateId}/requests/${requestId}`)
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

// Keyed off the table's row model (the same source the template renders) so the
// truncation map and the visible rows never diverge.
const truncatedQueries = computed(() => {
  const map = new Map<string, ReturnType<typeof smartTruncateQuery>>()
  for (const { original: r } of table.getRowModel().rows) {
    map.set(r.id, smartTruncateQuery(r.path, r.raw_query))
  }
  return map
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
            v-for="{ original: request } in table.getRowModel().rows"
            :key="request.id"
            role="button"
            tabindex="0"
            :aria-label="`View request ${request.method} ${request.path}`"
            class="flex flex-col gap-2 px-5 py-3.5 cursor-pointer transition-colors hover:bg-accent focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring sm:flex-row sm:items-center sm:gap-0"
            @click="handleRequestClick(request.id)"
            @keydown.enter.prevent="handleRequestClick(request.id)"
            @keydown.space.prevent="handleRequestClick(request.id)"
          >
            <div class="flex items-center gap-3 flex-1 min-w-0">
              <span
                class="inline-flex items-center justify-center text-xs font-bold font-mono px-2 py-0.5 rounded-md tracking-wide w-14 text-center shrink-0"
                :class="methodColorClass(request.method)"
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
            <div
              class="flex items-center gap-x-4 gap-y-1 flex-wrap pl-[4.25rem] sm:flex-nowrap sm:shrink-0 sm:ml-4 sm:pl-0"
            >
              <!-- Diff badge -->
              <span
                v-if="request.has_diff"
                class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-danger/10 text-danger"
              >
                <span class="w-1 h-1 rounded-full bg-danger" />
                Diff
              </span>
              <span
                v-else
                class="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-success/10 text-success"
              >
                No diff
              </span>
              <!-- Status codes -->
              <span class="text-xs font-mono text-dim whitespace-nowrap sm:w-24 sm:text-right">
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
              <span class="text-xs font-mono text-dim whitespace-nowrap sm:w-36 sm:text-right">
                {{ formatLatency(request.live_response?.latency_ms) }} /
                {{ formatLatency(request.shadow_response?.latency_ms) }}
              </span>
              <!-- Timestamp -->
              <div class="text-xs text-dim sm:w-20 sm:text-right">
                {{ formatTimestamp(request.created_at) }}
              </div>
              <ChevronRight class="hidden h-3.5 w-3.5 text-dim/40 shrink-0 sm:block" />
            </div>
          </div>
        </div>
      </TooltipProvider>

      <!-- Pagination Controls -->
      <Pagination
        :current-page="currentPage"
        :total-pages="totalPages"
        :disabled-prev="offset === 0"
        :disabled-next="!hasMore"
        @prev="prevPage"
        @next="nextPage"
      />
    </div>
  </div>
</template>
