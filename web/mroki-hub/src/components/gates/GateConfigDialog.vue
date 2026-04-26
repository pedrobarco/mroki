<script setup lang="ts">
import { ref, watch } from 'vue'
import { updateGate } from '@/api'
import type { Gate } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

const props = defineProps<{
  gate: Gate
}>()

const open = defineModel<boolean>('open', { default: false })

const emit = defineEmits<{
  success: [gate: Gate]
}>()

const name = ref('')
const ignoredFields = ref('')
const includedFields = ref('')
const floatTolerance = ref('')
const submitting = ref(false)
const error = ref<string | null>(null)

// Populate form when dialog opens
watch(open, (isOpen) => {
  if (isOpen) {
    name.value = props.gate.name
    ignoredFields.value = props.gate.diff_config.ignored_fields?.join(', ') ?? ''
    includedFields.value = props.gate.diff_config.included_fields?.join(', ') ?? ''
    floatTolerance.value = props.gate.diff_config.float_tolerance
      ? props.gate.diff_config.float_tolerance.toString()
      : ''
    error.value = null
  }
})

function parseCommaSeparated(value: string): string[] {
  return value
    .split(',')
    .map((s) => s.trim())
    .filter(Boolean)
}

async function handleSubmit() {
  submitting.value = true
  error.value = null

  try {
    const response = await updateGate(props.gate.id, {
      name: name.value.trim(),
      diff_config: {
        ignored_fields: parseCommaSeparated(ignoredFields.value),
        included_fields: parseCommaSeparated(includedFields.value),
        float_tolerance: floatTolerance.value ? parseFloat(floatTolerance.value) : 0,
      },
    })

    emit('success', response.data)
    open.value = false
  } catch (err) {
    if (err instanceof Error) {
      error.value = err.message
    } else {
      error.value = 'Failed to update gate'
    }
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Dialog v-model:open="open">
    <DialogContent>
      <DialogHeader>
        <DialogTitle>Configure Gate</DialogTitle>
        <DialogDescription> Update the gate name and diff configuration. </DialogDescription>
      </DialogHeader>

      <form class="space-y-4" @submit.prevent="handleSubmit">
        <!-- Error Alert -->
        <Alert v-if="error" variant="destructive">
          <AlertDescription>{{ error }}</AlertDescription>
        </Alert>

        <!-- Name Field -->
        <div class="space-y-2">
          <Label for="config-name">Name</Label>
          <Input
            id="config-name"
            v-model="name"
            type="text"
            placeholder="checkout-api"
            required
            :disabled="submitting"
          />
        </div>

        <!-- Diff Config Section -->
        <div class="space-y-3 pt-2">
          <p class="text-sm font-medium">Diff Configuration</p>

          <div class="space-y-2">
            <Label for="config-ignored">Ignored Fields</Label>
            <Input
              id="config-ignored"
              v-model="ignoredFields"
              type="text"
              placeholder="timestamp, request_id, trace_id"
              :disabled="submitting"
            />
            <p class="text-xs text-muted-foreground">
              Comma-separated JSON paths to ignore in diffs
            </p>
          </div>

          <div class="space-y-2">
            <Label for="config-included">Included Fields</Label>
            <Input
              id="config-included"
              v-model="includedFields"
              type="text"
              placeholder="body.status, body.data"
              :disabled="submitting"
            />
            <p class="text-xs text-muted-foreground">
              Comma-separated JSON paths to include (empty = all)
            </p>
          </div>

          <div class="space-y-2">
            <Label for="config-tolerance">Float Tolerance</Label>
            <Input
              id="config-tolerance"
              v-model="floatTolerance"
              type="number"
              step="any"
              min="0"
              placeholder="0.001"
              :disabled="submitting"
            />
            <p class="text-xs text-muted-foreground">
              Tolerance for floating-point comparisons (0 = exact)
            </p>
          </div>
        </div>

        <DialogFooter>
          <Button type="submit" :disabled="!name.trim() || submitting">
            {{ submitting ? 'Saving...' : 'Save Changes' }}
          </Button>
        </DialogFooter>
      </form>
    </DialogContent>
  </Dialog>
</template>
