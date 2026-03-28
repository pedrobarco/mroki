<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { getGates } from '@/api'
import type { Gate } from '@/api'
import type { GateFilterState } from './GateFilters.vue'
import GateCard from './GateCard.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

interface Props {
  filters: GateFilterState
}

const props = defineProps<Props>()

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

    <!-- Empty State -->
    <div v-else-if="gates.length === 0" class="text-center py-12">
      <p class="text-muted-foreground">
        No gates found. Try adjusting your filters or create a new gate.
      </p>
    </div>

    <!-- Gates List -->
    <div v-else>
      <div class="space-y-3">
        <GateCard v-for="(gate, i) in gates" :key="gate.id" :gate="gate" :index="i" />
      </div>

      <!-- Pagination Controls -->
      <div v-if="totalPages > 1" class="flex items-center justify-between mt-4 text-xs">
        <span class="text-dim">Page {{ currentPage }} of {{ totalPages }} · {{ total }} gates</span>
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
