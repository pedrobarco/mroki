<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useQuery, keepPreviousData } from '@tanstack/vue-query'
import { useVueTable, getCoreRowModel } from '@tanstack/vue-table'
import type { ColumnDef, PaginationState, SortingState, Updater } from '@tanstack/vue-table'
import { gatesQuery } from '@/api'
import type { Gate, ListGatesParams } from '@/api'
import type { GateFilterState } from './GateFilters.vue'
import GateCard from './GateCard.vue'
import Pagination from '@/components/common/Pagination.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { GitCompareArrows, Plus } from 'lucide-vue-next'

interface Props {
  filters: GateFilterState
}

const props = defineProps<Props>()

const emit = defineEmits<{
  create: []
  clearFilters: []
}>()

// A gate list can be empty for two reasons: the user has no gates yet
// (first run), or an active URL filter matched nothing.
const hasActiveFilter = computed(() => Boolean(props.filters.liveUrl || props.filters.shadowUrl))

// Pagination lives in a TanStack Table state object (pageIndex/pageSize) and is
// the single source of paging truth; `offset` is derived from it and drives the
// query key, so changing pages fetches (or serves cached) the matching page.
const pageSize = 5
const pagination = ref<PaginationState>({ pageIndex: 0, pageSize })
const offset = computed(() => pagination.value.pageIndex * pagination.value.pageSize)

const queryParams = computed<ListGatesParams>(() => ({
  limit: pageSize,
  offset: offset.value,
  live_url: props.filters.liveUrl || undefined,
  shadow_url: props.filters.shadowUrl || undefined,
  sort: props.filters.sort,
  order: props.filters.order,
}))

// Reset to the first page whenever the filter set changes; the key change from
// the new params triggers the refetch (no manual reload needed).
watch(
  () => props.filters,
  () => {
    pagination.value = { pageIndex: 0, pageSize }
  },
  { deep: true }
)

// keepPreviousData holds the current page on screen while the next one loads,
// avoiding a loading flash and out-of-order flicker during rapid paging.
const query = useQuery(
  computed(() => ({
    ...gatesQuery(queryParams.value),
    placeholderData: keepPreviousData,
  }))
)

const gates = computed(() => query.data.value?.data ?? [])
const total = computed(() => query.data.value?.pagination.total ?? 0)
const hasMore = computed(() => query.data.value?.pagination.has_more ?? false)
const loading = computed(() => query.isPending.value)
const error = computed(() =>
  query.isError.value ? (query.error.value?.message ?? 'Failed to load gates') : null
)

function loadGates() {
  query.refetch()
}

// Sorting is owned by the filter dropdown (GateFilters) and sent to the server,
// so we mirror it into the table's sorting state read-only — the card layout has
// no clickable column headers to drive it.
const sorting = computed<SortingState>(() => [
  { id: props.filters.sort, desc: props.filters.order === 'desc' },
])

// Minimal column defs: the list renders GateCard, not table cells, so columns
// only need accessor keys matching the server-sortable fields to back the row
// model. The generated ui/table primitives don't fit the card/mobile layout, so
// vue-table is adopted headlessly (state only) rather than for rendering.
const columns: ColumnDef<Gate>[] = [
  { accessorKey: 'name' },
  { accessorKey: 'live_url' },
  { accessorKey: 'shadow_url' },
  { accessorKey: 'created_at' },
]

// Manual (server-driven) mode: the server owns paging and sorting, so the table
// is a headless state layer over the fetched page. `rowCount` comes from the
// server total so getPageCount() matches the real page count.
const table = useVueTable({
  data: gates,
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
// Derive the page count from the reactive server total rather than
// table.getPageCount(): the table's page-count memo can lag when the total
// drops after a filter change (it keeps the previous, larger rowCount), leaving
// a stale "of N". `total`/`pageSize` are the source of truth for paging here.
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)))

function nextPage() {
  table.nextPage()
}

function prevPage() {
  table.previousPage()
}
</script>

<template>
  <div>
    <!-- Loading State -->
    <div v-if="loading" class="text-center py-12">
      <p class="text-muted-foreground">Loading gates...</p>
    </div>

    <!-- Error State -->
    <Alert v-else-if="error" variant="destructive">
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>
        {{ error }}
      </AlertDescription>
      <div class="mt-4">
        <Button variant="outline" size="sm" @click="loadGates"> Retry </Button>
      </div>
    </Alert>

    <!-- Empty State: active filter matched nothing -->
    <div v-else-if="gates.length === 0 && hasActiveFilter" class="text-center py-12">
      <p class="text-muted-foreground mb-4">No gates match your current filter.</p>
      <Button variant="outline" size="sm" @click="emit('clearFilters')">Clear filter</Button>
    </div>

    <!-- Empty State: first run, no gates yet -->
    <div
      v-else-if="gates.length === 0"
      class="flex flex-col items-center text-center px-6 py-16 border border-border rounded-xl bg-card"
    >
      <div class="flex h-12 w-12 items-center justify-center rounded-full bg-accent text-primary">
        <GitCompareArrows class="h-6 w-6" />
      </div>
      <h2 class="mt-5 text-base font-semibold tracking-tight text-foreground">
        Create your first gate
      </h2>
      <p class="mt-2 max-w-md text-sm text-muted-foreground">
        A gate pairs a live service with a shadow service. mroki mirrors your traffic to both and
        highlights every difference in their JSON responses.
      </p>
      <Button class="mt-6 gap-2" @click="emit('create')">
        <Plus class="h-3.5 w-3.5" />
        New gate
      </Button>
      <p class="mt-4 text-xs text-dim">
        Once it exists, send traffic through the proxy to see diffs appear here.
      </p>
    </div>

    <!-- Gates List -->
    <div v-else>
      <div class="space-y-3">
        <GateCard
          v-for="row in table.getRowModel().rows"
          :key="row.original.id"
          :gate="row.original"
        />
      </div>

      <!-- Pagination Controls -->
      <Pagination
        :current-page="currentPage"
        :total-pages="totalPages"
        :disabled-prev="offset === 0"
        :disabled-next="!hasMore"
        @prev="prevPage"
        @next="nextPage"
      >
        <template #meta> · {{ total }} gates</template>
      </Pagination>
    </div>
  </div>
</template>
