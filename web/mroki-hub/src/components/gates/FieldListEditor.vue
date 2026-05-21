<script setup lang="ts">
import { ref } from 'vue'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Plus, X } from 'lucide-vue-next'

const props = defineProps<{
  fields: string[]
  placeholder?: string
  disabled?: boolean
}>()

const emit = defineEmits<{
  add: [field: string]
  remove: [index: number]
}>()

const newField = ref('')

function handleAdd() {
  const trimmed = newField.value.trim()
  if (!trimmed) return
  if (props.fields.includes(trimmed)) return
  emit('add', trimmed)
  newField.value = ''
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') {
    e.preventDefault()
    handleAdd()
  }
}
</script>

<template>
  <div class="space-y-2">
    <!-- Existing fields -->
    <div v-for="(field, index) in fields" :key="field" class="flex items-center gap-2 group">
      <div
        class="flex-1 bg-background/60 border border-border/50 rounded-lg px-3 py-2 flex items-center"
      >
        <code class="text-xs font-mono text-muted-foreground">{{ field }}</code>
      </div>
      <Button
        variant="outline"
        size="icon"
        class="h-7 w-7 shrink-0 opacity-0 group-hover:opacity-100 transition-opacity text-dim hover:text-destructive hover:border-destructive/30"
        :disabled="disabled"
        @click="emit('remove', index)"
      >
        <X class="h-3 w-3" />
      </Button>
    </div>

    <!-- Add new field -->
    <div class="flex items-center gap-2">
      <Input
        v-model="newField"
        type="text"
        :placeholder="placeholder ?? 'e.g. headers.X-Custom-Auth'"
        class="flex-1 font-mono text-xs"
        :disabled="disabled"
        @keydown="handleKeydown"
      />
      <Button
        variant="outline"
        size="sm"
        class="gap-1.5 shrink-0"
        :disabled="disabled || !newField.trim()"
        @click="handleAdd"
      >
        <Plus class="h-3 w-3" />
        Add
      </Button>
    </div>
  </div>
</template>
