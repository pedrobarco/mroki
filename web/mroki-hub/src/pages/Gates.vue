<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { getGlobalStats } from '@/api'
import type { GlobalStats } from '@/api'
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
import { Plus } from 'lucide-vue-next'

const dialogOpen = ref(false)

const filters = reactive<GateFilterState>({
  liveUrl: '',
  shadowUrl: '',
  sort: 'created_at',
  order: 'desc',
})

const globalStats = ref<GlobalStats | null>(null)

const stats = computed(() => [
  { label: 'TOTAL GATES', value: globalStats.value?.total_gates.toLocaleString() ?? '—' },
  { label: 'REQUESTS (24H)', value: globalStats.value?.total_requests_24h.toLocaleString() ?? '—' },
  {
    label: 'DIFF RATE',
    value: globalStats.value ? `${globalStats.value.total_diff_rate.toFixed(1)}%` : '—',
    highlight: true,
  },
])
const listKey = ref(0)

async function loadStats() {
  try {
    const response = await getGlobalStats()
    globalStats.value = response.data
  } catch {
    // Stats are non-critical; leave as null
  }
}

function handleGateCreated() {
  dialogOpen.value = false
  listKey.value++ // Force GateList to reload
  loadStats() // Refresh stats after gate creation
}

function onFiltersUpdate(newFilters: GateFilterState) {
  Object.assign(filters, newFilters)
}

onMounted(() => {
  loadStats()
})
</script>

<template>
  <div class="max-w-6xl mx-auto px-6 py-8">
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
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
            New Gate
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Gate</DialogTitle>
            <DialogDescription>
              Enter the URLs for your live and shadow services to create a new gate.
            </DialogDescription>
          </DialogHeader>
          <GateForm @success="handleGateCreated" />
        </DialogContent>
      </Dialog>
    </div>

    <!-- Stats Bar -->
    <div class="grid grid-cols-3 gap-4 mb-6">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="bg-card border border-border rounded-xl px-4 py-3.5"
      >
        <div class="text-xs uppercase tracking-widest text-dim mb-1">{{ stat.label }}</div>
        <div
          class="text-lg font-semibold tracking-tight"
          :class="stat.highlight ? 'text-warning' : 'text-foreground'"
        >
          {{ stat.value }}
        </div>
      </div>
    </div>

    <!-- Filters & Sort Row -->
    <div class="mb-5">
      <GateFilters :model-value="filters" @update:model-value="onFiltersUpdate" />
    </div>

    <!-- Gates List -->
    <GateList :key="listKey" :filters="filters" />
  </div>
</template>
