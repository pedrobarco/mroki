<script setup lang="ts">
import { ref, reactive, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getGate, deleteGate } from '@/api'
import type { Gate } from '@/api'
import { useGateCache } from '@/composables/use-gate-cache'
import RequestList from '@/components/requests/RequestList.vue'
import RequestFilters from '@/components/requests/RequestFilters.vue'
import type { FilterState } from '@/components/requests/RequestFilters.vue'
import GateConfigDialog from '@/components/gates/GateConfigDialog.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { ChevronLeft, Settings, Trash2 } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const { setGate: cacheGate } = useGateCache()

const gate = ref<Gate | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)
const requestTotal = ref<number | null>(null)
const requestShowing = ref<number | null>(null)
const configDialogOpen = ref(false)
const deleting = ref(false)

function handleConfigSuccess(updatedGate: Gate) {
  gate.value = updatedGate
  cacheGate(updatedGate)
}

async function handleDelete() {
  deleting.value = true
  try {
    await deleteGate(gateId.value)
    router.push('/gates')
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to delete gate'
    deleting.value = false
  }
}

const gateId = computed(() => route.params.id as string)

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

async function loadGate() {
  loading.value = true
  error.value = null

  try {
    const response = await getGate(gateId.value)
    gate.value = response.data
    cacheGate(response.data)
  } catch (err) {
    if (err instanceof Error) {
      error.value = err.message
    } else {
      error.value = 'Failed to load gate'
    }
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push('/gates')
}

onMounted(() => {
  loadGate()
})
</script>

<template>
  <div class="max-w-6xl mx-auto px-6 py-6">
    <!-- Back link -->
    <a
      class="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors mb-5 cursor-pointer"
      @click="goBack"
    >
      <ChevronLeft class="h-3.5 w-3.5" />
      Back to Gates
    </a>

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
        <div class="flex items-start justify-between mb-5">
          <div>
            <div class="flex items-center gap-2.5 mb-1.5">
              <h1 class="text-xl font-semibold tracking-tight">{{ gate.name }}</h1>
            </div>
            <code class="text-xs font-mono text-dim">{{ gate.id }}</code>
          </div>
          <div class="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              class="gap-1.5 text-xs"
              @click="configDialogOpen = true"
            >
              <Settings class="h-3.5 w-3.5" />
              Configure
            </Button>

            <AlertDialog>
              <AlertDialogTrigger as-child>
                <Button variant="outline" size="sm" class="gap-1.5 text-xs text-destructive">
                  <Trash2 class="h-3.5 w-3.5" />
                  Delete
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>Delete gate</AlertDialogTitle>
                  <AlertDialogDescription>
                    This will permanently delete
                    <strong>{{ gate.name }}</strong>
                    and all its captured requests. This action cannot be undone.
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>Cancel</AlertDialogCancel>
                  <AlertDialogAction
                    class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                    :disabled="deleting"
                    @click="handleDelete"
                  >
                    {{ deleting ? 'Deleting...' : 'Delete' }}
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>

          <!-- Configure Dialog -->
          <GateConfigDialog
            v-model:open="configDialogOpen"
            :gate="gate"
            @success="handleConfigSuccess"
          />
        </div>

        <!-- Live / Shadow URLs -->
        <div class="grid grid-cols-2 gap-3 mb-4">
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
            <span class="text-warning ml-1">{{ gate.stats.diff_rate.toFixed(1) }}%</span>
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
