<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getGates } from '@/api'
import type { Gate } from '@/api'
import GateCard from './GateCard.vue'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'

const gates = ref<Gate[]>([])
const loading = ref(true)
const error = ref<string | null>(null)

async function loadGates() {
  loading.value = true
  error.value = null

  try {
    const response = await getGates()
    gates.value = response.data
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
      <p class="text-muted-foreground">No gates yet. Create your first gate to get started!</p>
    </div>

    <!-- Gates List -->
    <div v-else class="space-y-3">
      <GateCard v-for="(gate, i) in gates" :key="gate.id" :gate="gate" :index="i" />
    </div>
  </div>
</template>
