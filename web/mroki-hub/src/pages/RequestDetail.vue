<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getRequest } from '@/api'
import type { RequestDetail, Response } from '@/api'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import DiffViewer from '@/components/diff/DiffViewer.vue'

const route = useRoute()
const router = useRouter()

const request = ref<RequestDetail | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const gateId = computed(() => route.params.id as string)
const requestId = computed(() => route.params.rid as string)

// Find live and shadow responses
const liveResponse = computed(() =>
  request.value?.responses.find((r) => r.type === 'live') || null
)

const shadowResponse = computed(() =>
  request.value?.responses.find((r) => r.type === 'shadow') || null
)

async function loadRequest() {
  loading.value = true
  error.value = null

  try {
    const response = await getRequest(gateId.value, requestId.value)
    request.value = response.data
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load request'
  } finally {
    loading.value = false
  }
}

function goBack() {
  router.push(`/gates/${gateId.value}`)
}

function formatTimestamp(timestamp: string): string {
  return new Date(timestamp).toLocaleString()
}

onMounted(() => {
  loadRequest()
})
</script>

<template>
  <div class="container mx-auto py-8 space-y-6">
    <!-- Header with Back Button -->
    <div class="flex items-center gap-4">
      <Button variant="outline" size="sm" @click="goBack">
        ← Back to Gate
      </Button>
      <h1 class="text-3xl font-bold">Request Detail</h1>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="flex items-center justify-center py-12">
      <div class="text-center">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary mx-auto mb-4"></div>
        <p class="text-muted-foreground">Loading request...</p>
      </div>
    </div>

    <!-- Error State -->
    <Alert v-else-if="error" variant="destructive">
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>{{ error }}</AlertDescription>
    </Alert>

    <!-- Request Content -->
    <div v-else-if="request" class="space-y-6">
      <!-- Request Metadata -->
      <Card>
        <CardHeader>
          <CardTitle>Request Information</CardTitle>
        </CardHeader>
        <CardContent class="space-y-4">
          <div class="grid grid-cols-2 gap-4">
            <div>
              <span class="text-sm font-medium text-muted-foreground">Method</span>
              <div class="mt-1">
                <Badge>{{ request.method }}</Badge>
              </div>
            </div>
            <div>
              <span class="text-sm font-medium text-muted-foreground">Timestamp</span>
              <p class="mt-1 text-sm">{{ formatTimestamp(request.created_at) }}</p>
            </div>
          </div>

          <div>
            <span class="text-sm font-medium text-muted-foreground">Path</span>
            <p class="mt-1 text-sm font-mono">{{ request.path }}</p>
          </div>

          <div>
            <span class="text-sm font-medium text-muted-foreground">Request ID</span>
            <p class="mt-1 text-xs font-mono text-muted-foreground">{{ request.id }}</p>
          </div>
        </CardContent>
      </Card>

      <!-- Diff Viewer -->
      <Card v-if="liveResponse && shadowResponse">
        <CardHeader>
          <CardTitle>Response Comparison</CardTitle>
        </CardHeader>
        <CardContent>
          <DiffViewer
            :live-response="liveResponse"
            :shadow-response="shadowResponse"
            :diff-content="request.diff.content"
          />
        </CardContent>
      </Card>

      <!-- Missing Responses Warning -->
      <Alert v-else variant="destructive">
        <AlertTitle>Incomplete Data</AlertTitle>
        <AlertDescription>
          This request is missing {{ !liveResponse ? 'live' : 'shadow' }} response data.
        </AlertDescription>
      </Alert>
    </div>
  </div>
</template>
