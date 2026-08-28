<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { globalStatsQuery, queryKeys } from '@/api'
import { diffRateColorClass } from '@/lib/utils'
import GateList from '@/components/gates/GateList.vue'
import GateForm from '@/components/gates/GateForm.vue'
import GateFilters from '@/components/gates/GateFilters.vue'
import type { GateFilterState } from '@/components/gates/GateFilters.vue'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Plus, RefreshCw } from 'lucide-vue-next'

const dialogOpen = ref(false)

const filters = reactive<GateFilterState>({
  liveUrl: '',
  shadowUrl: '',
  sort: 'created_at',
  order: 'desc',
})

const queryClient = useQueryClient()

// Global stats poll on a 30s interval, matching the previous setInterval cadence
// but managed by TanStack Query (dedup, cache sharing, cleanup on unmount).
const statsQuery = useQuery({
  ...globalStatsQuery(),
  refetchInterval: 30_000,
})
const { data: globalStats, isError: statsError, isFetching: statsLoading } = statsQuery

function loadStats() {
  statsQuery.refetch()
}

// Only surface a timestamp once stats have actually landed; until then the age
// label stays hidden.
const statsUpdatedAt = computed(() => (globalStats.value ? statsQuery.dataUpdatedAt.value : null))

// Re-render the "updated Xs ago" label on a ticking clock without refetching.
const now = ref(Date.now())

const statsAgeLabel = computed(() => {
  if (statsUpdatedAt.value == null) return ''
  const secs = Math.max(0, Math.floor((now.value - statsUpdatedAt.value) / 1000))
  if (secs < 5) return 'updated just now'
  if (secs < 60) return `updated ${secs}s ago`
  const mins = Math.floor(secs / 60)
  return `updated ${mins}m ago`
})

const stats = computed(() => [
  {
    label: 'Total gates',
    value: globalStats.value?.total_gates.toLocaleString() ?? '—',
    color: 'text-foreground',
  },
  {
    label: 'Requests 24h',
    value: globalStats.value?.total_requests_24h.toLocaleString() ?? '—',
    color: 'text-foreground',
  },
  {
    label: 'Diff rate',
    value: globalStats.value ? `${globalStats.value.total_diff_rate.toFixed(1)}%` : '—',
    color: globalStats.value
      ? diffRateColorClass(globalStats.value.total_diff_rate)
      : 'text-foreground',
  },
])
function handleGateCreated() {
  dialogOpen.value = false
  // Refresh the reads affected by the new gate: every gate list variant and the
  // global stats. Invalidation lets TanStack Query refetch the active queries
  // rather than remounting the list.
  queryClient.invalidateQueries({ queryKey: queryKeys.gates.all })
  queryClient.invalidateQueries({ queryKey: queryKeys.stats.global })
}

function onFiltersUpdate(newFilters: GateFilterState) {
  Object.assign(filters, newFilters)
}

function clearFilters() {
  Object.assign(filters, { liveUrl: '', shadowUrl: '' })
}

let clockTimer: ReturnType<typeof setInterval> | undefined

onMounted(() => {
  // Tick the relative "updated Xs ago" label every second. Stats polling itself
  // is handled by the query's refetchInterval.
  clockTimer = setInterval(() => {
    now.value = Date.now()
  }, 1000)
})

onUnmounted(() => {
  if (clockTimer) clearInterval(clockTimer)
})
</script>

<template>
  <div class="max-w-6xl mx-auto px-6 py-8">
    <!-- Page Header -->
    <div class="flex flex-col gap-4 mb-8 sm:flex-row sm:items-center sm:justify-between">
      <div>
        <h1 class="text-xl font-semibold tracking-tight mb-1">Gates</h1>
        <p class="text-xs text-muted-foreground">
          Manage live/shadow service pairs and monitor traffic diffs.
        </p>
      </div>

      <!-- Create Gate Dialog -->
      <Dialog v-model:open="dialogOpen">
        <DialogTrigger as-child>
          <Button class="gap-2">
            <Plus class="h-3.5 w-3.5" />
            New gate
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create gate</DialogTitle>
            <DialogDescription>
              Enter the URLs for your live and shadow services to create a new gate.
            </DialogDescription>
          </DialogHeader>
          <GateForm @success="handleGateCreated" />
        </DialogContent>
      </Dialog>
    </div>

    <!-- Stats Bar -->
    <div class="mb-6">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-3">
        <div
          v-for="stat in stats"
          :key="stat.label"
          class="bg-card border border-border rounded-xl px-4 py-3.5"
        >
          <div class="text-xs uppercase tracking-widest text-dim mb-1">{{ stat.label }}</div>
          <div class="text-lg font-semibold tracking-tight" :class="stat.color">
            {{ stat.value }}
          </div>
        </div>
      </div>

      <!-- Stats freshness + manual refresh -->
      <div class="mt-2 flex items-center gap-2 text-xs text-dim">
        <button
          type="button"
          class="inline-flex items-center gap-1 rounded-sm text-dim hover:text-muted-foreground transition-colors focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background disabled:opacity-50"
          :disabled="statsLoading"
          @click="loadStats"
        >
          <RefreshCw class="h-3 w-3" :class="statsLoading ? 'animate-spin' : ''" />
          Refresh
        </button>
        <span v-if="statsError" class="text-warning">Stats unavailable — retry</span>
        <span v-else-if="statsAgeLabel">{{ statsAgeLabel }}</span>
      </div>
    </div>

    <!-- Filters & Sort Row -->
    <div class="mb-5">
      <GateFilters :model-value="filters" @update:model-value="onFiltersUpdate" />
    </div>

    <!-- Gates List -->
    <GateList :filters="filters" @create="dialogOpen = true" @clear-filters="clearFilters" />
  </div>
</template>
