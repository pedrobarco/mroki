<script setup lang="ts">
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { getRequests } from '@/api'
import type { Request } from '@/api'
import type { FilterState } from '@/components/requests/RequestFilters.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

interface Props {
  gateId: string
  filters: FilterState
}

const props = defineProps<Props>()
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

function getMethodVariant(method: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  switch (method.toUpperCase()) {
    case 'GET':
      return 'default'
    case 'POST':
      return 'secondary'
    case 'PUT':
    case 'PATCH':
      return 'outline'
    case 'DELETE':
      return 'destructive'
    default:
      return 'outline'
  }
}

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp)
  return date.toLocaleString()
}

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
      <AlertDescription>
        {{ error }}
      </AlertDescription>
      <div class="mt-4">
        <Button variant="outline" size="sm" @click="loadRequests"> Retry </Button>
      </div>
    </Alert>

    <!-- Empty State -->
    <div v-else-if="requests.length === 0" class="text-center py-12">
      <p class="text-muted-foreground">
        No requests captured yet. Send traffic through this gate to see requests here.
      </p>
    </div>

    <!-- Requests Table -->
    <div v-else>
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>Method</TableHead>
            <TableHead>Path</TableHead>
            <TableHead>Timestamp</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow
            v-for="request in requests"
            :key="request.id"
            class="cursor-pointer hover:bg-accent"
            @click="handleRequestClick(request.id)"
          >
            <TableCell>
              <Badge :variant="getMethodVariant(request.method)">
                {{ request.method }}
              </Badge>
            </TableCell>
            <TableCell class="font-mono text-sm">{{ request.path }}</TableCell>
            <TableCell class="text-muted-foreground">
              {{ formatTimestamp(request.created_at) }}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>

      <!-- Pagination Controls -->
      <div class="flex items-center justify-between mt-4">
        <p class="text-sm text-muted-foreground">
          Showing {{ offset + 1 }}-{{ Math.min(offset + limit, total) }} of {{ total }} requests
        </p>
        <div class="flex gap-2">
          <Button variant="outline" size="sm" :disabled="offset === 0" @click="prevPage">
            Previous
          </Button>
          <span class="flex items-center px-3 text-sm text-muted-foreground">
            Page {{ currentPage }} of {{ totalPages }}
          </span>
          <Button variant="outline" size="sm" :disabled="!hasMore" @click="nextPage"> Next </Button>
        </div>
      </div>
    </div>
  </div>
</template>
