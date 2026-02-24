<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getGate } from '@/api'
import type { Gate } from '@/api'
import RequestList from '@/components/requests/RequestList.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { truncateId } from '@/lib/utils'

const route = useRoute()
const router = useRouter()

const gate = ref<Gate | null>(null)
const loading = ref(true)
const error = ref<string | null>(null)

const gateId = computed(() => route.params.id as string)

async function loadGate() {
  loading.value = true
  error.value = null

  try {
    const response = await getGate(gateId.value)
    gate.value = response.data
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
  <div class="container mx-auto p-6">
    <!-- Back Button -->
    <div class="mb-6">
      <Button variant="ghost" @click="goBack"> ← Back to Gates </Button>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="text-center py-12">
      <p class="text-muted-foreground">Loading gate details...</p>
    </div>

    <!-- Error State -->
    <Alert v-else-if="error" variant="destructive">
      <AlertTitle>Error</AlertTitle>
      <AlertDescription>
        {{ error }}
      </AlertDescription>
      <div class="mt-4">
        <Button variant="outline" size="sm" @click="loadGate"> Retry </Button>
      </div>
    </Alert>

    <!-- Gate Details & Requests -->
    <div v-else-if="gate">
      <!-- Gate Info Card -->
      <Card class="mb-6">
        <CardHeader>
          <CardTitle>Gate {{ truncateId(gate.id) }}</CardTitle>
        </CardHeader>
        <CardContent class="space-y-4">
          <div>
            <span class="text-sm font-medium text-muted-foreground">Live Service:</span>
            <p class="text-sm text-foreground break-all font-mono">{{ gate.live_url }}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-muted-foreground">Shadow Service:</span>
            <p class="text-sm text-foreground break-all font-mono">{{ gate.shadow_url }}</p>
          </div>
          <div>
            <span class="text-sm font-medium text-muted-foreground">Gate ID:</span>
            <p class="text-sm text-foreground font-mono">{{ gate.id }}</p>
          </div>
        </CardContent>
      </Card>

      <!-- Requests Section -->
      <div>
        <h2 class="text-2xl font-bold mb-4">Captured Requests</h2>
        <RequestList :gate-id="gateId" />
      </div>
    </div>
  </div>
</template>
