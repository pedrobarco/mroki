<script setup lang="ts">
import { ref, reactive, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import { gateQuery } from '@/api'
import { diffRateColorClass } from '@/lib/utils'
import RequestList from '@/components/requests/RequestList.vue'
import RequestFilters from '@/components/requests/RequestFilters.vue'
import type { FilterState } from '@/components/requests/RequestFilters.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { ChevronLeft, Settings } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()

const requestTotal = ref<number | null>(null)
const requestShowing = ref<number | null>(null)

const gateId = computed(() => route.params.id as string)

// GateDetail displays volatile stats, so it always reads through the shared
// query cache. The detail entry is keyed by gate id and reused by Settings and
// RequestDetail (no bespoke in-memory cache needed).
const gateQueryResult = useQuery(computed(() => gateQuery(gateId.value)))
const { data: gate, isPending: loading, refetch: loadGate } = gateQueryResult
const error = computed(() =>
  gateQueryResult.isError.value
    ? (gateQueryResult.error.value?.message ?? 'Failed to load gate')
    : null
)

const filters = reactive<FilterState>({
  methods: [],
  path: '',
  hasDiff: undefined,
  sort: 'created_at',
  order: 'desc',
})

function onFiltersUpdate(newFilters: FilterState) {
  Object.assign(filters, newFilters)
}

function goBack() {
  router.push('/gates')
}
</script>

<template>
  <div class="max-w-6xl mx-auto px-6 py-6">
    <!-- Back link -->
    <button
      type="button"
      class="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors mb-5 cursor-pointer rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background"
      @click="goBack"
    >
      <ChevronLeft class="h-3.5 w-3.5" />
      Back to Gates
    </button>

    <!-- Loading State -->
    <div v-if="loading" class="text-center py-12">
      <p class="text-muted-foreground">Loading gate details...</p>
    </div>

    <!-- Error State -->
    <Alert v-else-if="error" variant="destructive">
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
      <div class="mt-4">
        <Button variant="outline" size="sm" @click="loadGate">Retry</Button>
      </div>
    </Alert>

    <!-- Gate Details & Requests -->
    <div v-else-if="gate">
      <!-- Gate Info Card -->
      <div class="bg-card border border-border rounded-xl p-5 mb-8">
        <div class="flex flex-col gap-4 mb-5 sm:flex-row sm:items-start sm:justify-between">
          <div>
            <div class="flex items-center gap-2.5 mb-1.5">
              <h1 class="text-xl font-semibold tracking-tight">{{ gate.name }}</h1>
            </div>
            <code class="text-xs font-mono text-dim">{{ gate.id }}</code>
          </div>
          <Button
            variant="outline"
            size="sm"
            class="gap-1.5 text-xs"
            @click="router.push(`/gates/${gateId}/settings`)"
          >
            <Settings class="h-3.5 w-3.5" />
            Settings
          </Button>
        </div>

        <!-- Live / Shadow URLs -->
        <div class="grid grid-cols-1 gap-3 mb-4 sm:grid-cols-2">
          <div class="bg-background/60 rounded-lg px-3.5 py-3 border border-border/50">
            <div
              class="text-xs uppercase tracking-widest text-dim mb-1.5 flex items-center gap-1.5"
            >
              <span class="w-1.5 h-1.5 rounded-full bg-success" />
              Live
            </div>
            <code class="text-xs font-mono text-muted-foreground">
              {{ gate.live_url }}
            </code>
          </div>
          <div class="bg-background/60 rounded-lg px-3.5 py-3 border border-border/50">
            <div
              class="text-xs uppercase tracking-widest text-dim mb-1.5 flex items-center gap-1.5"
            >
              <span class="w-1.5 h-1.5 rounded-full bg-info" />
              Shadow
            </div>
            <code class="text-xs font-mono text-muted-foreground">
              {{ gate.shadow_url }}
            </code>
          </div>
        </div>

        <!-- Stats footer -->
        <div class="flex items-center gap-6 text-xs pt-3 border-t border-border/50">
          <div>
            <span class="text-dim">Created</span>
            <span class="text-muted-foreground ml-1">{{
              new Date(gate.created_at).toLocaleDateString('en-US', {
                month: 'short',
                day: 'numeric',
                year: 'numeric',
              })
            }}</span>
          </div>
          <div>
            <span class="text-dim">Requests 24h</span>
            <span class="text-muted-foreground ml-1">{{
              gate.stats.request_count_24h.toLocaleString()
            }}</span>
          </div>
          <div>
            <span class="text-dim">Diff rate</span>
            <span class="ml-1" :class="diffRateColorClass(gate.stats.diff_rate)"
              >{{ gate.stats.diff_rate.toFixed(1) }}%</span
            >
          </div>
        </div>
      </div>

      <!-- Captured Requests Section -->
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-base font-semibold tracking-tight">Captured Requests</h2>
        <span v-if="requestTotal !== null" class="text-xs text-dim">
          Showing {{ requestShowing ?? 0 }} of {{ requestTotal }} request{{
            requestTotal !== 1 ? 's' : ''
          }}
        </span>
      </div>

      <!-- Filters -->
      <div class="mb-4">
        <RequestFilters :model-value="filters" @update:model-value="onFiltersUpdate" />
      </div>

      <RequestList
        :gate-id="gateId"
        :filters="filters"
        @update:total="requestTotal = $event"
        @update:showing="requestShowing = $event"
      />
    </div>
  </div>
</template>
