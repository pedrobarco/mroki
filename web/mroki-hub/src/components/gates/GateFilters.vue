<script setup lang="ts">
import { ref, watch } from 'vue'
import type { AcceptableValue } from 'reka-ui'
import type { GateSortField, SortOrder } from '@/api'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Search } from 'lucide-vue-next'

export interface GateFilterState {
  liveUrl: string
  shadowUrl: string
  sort: GateSortField
  order: SortOrder
}

interface Props {
  modelValue: GateFilterState
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: GateFilterState): void
}>()

const searchInput = ref(props.modelValue.liveUrl)

// Debounce search input so we don't fire on every keystroke
let searchTimeout: ReturnType<typeof setTimeout> | null = null
watch(searchInput, (val) => {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    emitUpdate({ liveUrl: val, shadowUrl: '' })
  }, 400)
})

function onSortChange(val: AcceptableValue) {
  const strVal = String(val)
  switch (strVal) {
    case 'id-desc':
      emitUpdate({ sort: 'id', order: 'desc' })
      break
    case 'id-asc':
      emitUpdate({ sort: 'id', order: 'asc' })
      break
    case 'live_url-asc':
      emitUpdate({ sort: 'live_url', order: 'asc' })
      break
    case 'live_url-desc':
      emitUpdate({ sort: 'live_url', order: 'desc' })
      break
    case 'shadow_url-asc':
      emitUpdate({ sort: 'shadow_url', order: 'asc' })
      break
    case 'shadow_url-desc':
      emitUpdate({ sort: 'shadow_url', order: 'desc' })
      break
  }
}

function currentSortValue(): string {
  return `${props.modelValue.sort}-${props.modelValue.order}`
}

function emitUpdate(partial: Partial<GateFilterState>) {
  emit('update:modelValue', { ...props.modelValue, ...partial })
}
</script>

<template>
  <div class="flex items-center gap-3">
    <!-- URL search (filters both live_url and shadow_url) -->
    <div class="relative flex-1 max-w-sm">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 text-dim h-3.5 w-3.5" />
      <input
        v-model="searchInput"
        type="text"
        placeholder="Search gates by URL..."
        class="w-full bg-card border border-border rounded-lg pl-8 pr-4 py-2 text-xs text-foreground placeholder:text-dim focus:outline-none focus:border-ring focus:ring-1 focus:ring-ring"
      />
    </div>

    <!-- Sort -->
    <Select :model-value="currentSortValue()" @update:model-value="onSortChange">
      <SelectTrigger
        class="w-auto gap-1 text-xs text-dim border border-border rounded-lg px-3 py-2 bg-card h-auto"
      >
        <span class="text-dim">Sort:</span>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="id-desc">Newest first</SelectItem>
        <SelectItem value="id-asc">Oldest first</SelectItem>
        <SelectItem value="live_url-asc">Live URL (A→Z)</SelectItem>
        <SelectItem value="live_url-desc">Live URL (Z→A)</SelectItem>
        <SelectItem value="shadow_url-asc">Shadow URL (A→Z)</SelectItem>
        <SelectItem value="shadow_url-desc">Shadow URL (Z→A)</SelectItem>
      </SelectContent>
    </Select>
  </div>
</template>
