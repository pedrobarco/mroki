<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { getGates } from '@/api'
import type { Gate } from '@/api'
import type { GateFilterState } from './GateFilters.vue'
import GateCard from './GateCard.vue'
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

const gates = ref<Gate[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

// Pagination state
const limit = 5
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
    loadGates()
  },
  { deep: true }
)

async function loadGates() {
  loading.value = true
  error.value = null

  try {
    const response = await getGates({
      limit,
      offset: offset.value,
      live_url: props.filters.liveUrl || undefined,
      shadow_url: props.filters.shadowUrl || undefined,
      sort: props.filters.sort,
      order: props.filters.order,
    })
    gates.value = response.data
    total.value = response.pagination.total
    hasMore.value = response.pagination.has_more
  } catch (err) {
    if (err instanceof Error) {
      error.value = err.message
    } else {
      error.value = 'Failed to load gates'
    }
  } finally {
    loading.value = false
  }
}

function nextPage() {
  if (hasMore.value) {
    offset.value += limit
    loadGates()
  }
}

function prevPage() {
  if (offset.value > 0) {
    offset.value = Math.max(0, offset.value - limit)
    loadGates()
  }
}

onMounted(() => {
  loadGates()
})
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
      <div v-if="totalPages > 1" class="flex items-center justify-between mt-4 text-xs">
        <span class="text-dim">Page {{ currentPage }} of {{ totalPages }} · {{ total }} gates</span>
        <div class="flex items-center gap-1">
          <button
            class="inline-flex items-center justify-center min-h-[44px] min-w-[44px] px-4 py-2.5 rounded-lg border border-border bg-card text-dim transition-colors"
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
            class="inline-flex items-center justify-center min-h-[44px] min-w-[44px] px-4 py-2.5 rounded-lg border border-border bg-accent text-foreground font-medium"
          >
            {{ currentPage }}
          </span>
          <button
            class="inline-flex items-center justify-center min-h-[44px] min-w-[44px] px-4 py-2.5 rounded-lg border border-border bg-card transition-colors"
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
