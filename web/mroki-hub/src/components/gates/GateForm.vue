<script setup lang="ts">
import { ref, computed } from 'vue'
import { createGate } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'

const emit = defineEmits<{
  success: []
}>()

const liveUrl = ref('')
const shadowUrl = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

function isValidUrl(url: string): boolean {
  if (!url) return false
  try {
    new URL(url)
    return true
  } catch {
    return false
  }
}

const canSubmit = computed(() => {
  return (
    liveUrl.value &&
    shadowUrl.value &&
    isValidUrl(liveUrl.value) &&
    isValidUrl(shadowUrl.value) &&
    !submitting.value
  )
})

async function handleSubmit() {
  if (!canSubmit.value) return

  submitting.value = true
  error.value = null

  try {
    await createGate({
      live_url: liveUrl.value,
      shadow_url: shadowUrl.value,
    })

    // Reset form
    liveUrl.value = ''
    shadowUrl.value = ''

    // Notify parent
    emit('success')
  } catch (err) {
    if (err instanceof Error) {
      error.value = err.message
    } else {
      error.value = 'Failed to create gate'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <form class="space-y-4" @submit.prevent="handleSubmit">
    <!-- Error Alert -->
    <Alert v-if="error" variant="destructive">
      <AlertDescription>
        {{ error }}
      </AlertDescription>
    </Alert>

    <!-- Live URL Field -->
    <div class="space-y-2">
      <Label for="live-url">Live URL</Label>
      <Input
        id="live-url"
        v-model="liveUrl"
        type="url"
        placeholder="https://api.production.example.com"
        required
        :disabled="submitting"
      />
      <p v-if="liveUrl && !isValidUrl(liveUrl)" class="text-sm text-destructive">
        Please enter a valid URL
      </p>
    </div>

    <!-- Shadow URL Field -->
    <div class="space-y-2">
      <Label for="shadow-url">Shadow URL</Label>
      <Input
        id="shadow-url"
        v-model="shadowUrl"
        type="url"
        placeholder="https://api.shadow.example.com"
        required
        :disabled="submitting"
      />
      <p v-if="shadowUrl && !isValidUrl(shadowUrl)" class="text-sm text-destructive">
        Please enter a valid URL
      </p>
    </div>

    <!-- Submit Button -->
    <div class="flex justify-end">
      <Button type="submit" :disabled="!canSubmit">
        {{ submitting ? 'Creating...' : 'Create Gate' }}
      </Button>
    </div>
  </form>
</template>
