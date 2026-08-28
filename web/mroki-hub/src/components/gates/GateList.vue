<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useQuery, keepPreviousData } from '@tanstack/vue-query'
import { gatesQuery } from '@/api'
import type { ListGatesParams } from '@/api'
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

// Pagination state. `offset` drives the query key, so changing pages simply
// updates it and TanStack Query fetches (or serves cached) the matching page.
const limit = 5
const offset = ref(0)

const queryParams = computed<ListGatesParams>(() => ({
  limit,
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
    offset.value = 0
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

const currentPage = computed(() => Math.floor(offset.value / limit) + 1)
const totalPages = computed(() => Math.ceil(total.value / limit))

function nextPage() {
  if (hasMore.value) {
    offset.value += limit
  }
}

function prevPage() {
  if (offset.value > 0) {
    offset.value = Math.max(0, offset.value - limit)
  }
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
        <GateCard v-for="gate in gates" :key="gate.id" :gate="gate" />
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
