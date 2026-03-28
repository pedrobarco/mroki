<script setup lang="ts">
import { ref, watch } from 'vue'
import type { AcceptableValue } from 'reka-ui'
import type { RequestSortField, SortOrder } from '@/api'
import { Switch } from '@/components/ui/switch'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Search } from 'lucide-vue-next'

export interface FilterState {
  methods: string[]
  path: string
  hasDiff: boolean | undefined
  sort: RequestSortField
  order: SortOrder
}

interface Props {
  modelValue: FilterState
}

const props = defineProps<Props>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: FilterState): void
}>()

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'DELETE'] as const

const pathInput = ref(props.modelValue.path)

// Debounce path input so we don't fire on every keystroke
let pathTimeout: ReturnType<typeof setTimeout> | null = null
watch(pathInput, (val) => {
  if (pathTimeout) clearTimeout(pathTimeout)
  pathTimeout = setTimeout(() => {
    emitUpdate({ path: val })
  }, 400)
})

const activeMethod = ref<string | null>(null)

function selectMethod(method: string | null) {
  if (activeMethod.value === method) {
    activeMethod.value = null
    emitUpdate({ methods: [] })
  } else {
    activeMethod.value = method
    emitUpdate({ methods: method ? [method] : [] })
  }
}

function onDiffToggle(checked: boolean) {
  emitUpdate({ hasDiff: checked ? true : undefined })
}

function onSortChange(val: AcceptableValue) {
  // Combine sort field and order into the sort select
  const strVal = String(val)
  if (strVal === 'newest') {
    emitUpdate({ sort: 'created_at', order: 'desc' })
  } else if (strVal === 'oldest') {
    emitUpdate({ sort: 'created_at', order: 'asc' })
  }
}

function emitUpdate(partial: Partial<FilterState>) {
  emit('update:modelValue', { ...props.modelValue, ...partial })
}
</script>

<template>
  <div class="flex items-center gap-3">
    <!-- Method filter buttons -->
    <div
      class="flex items-center gap-0 text-xs border border-border rounded-lg bg-card overflow-hidden"
    >
      <button
        class="px-2.5 py-1.5 transition-colors"
        :class="
          activeMethod === null
            ? 'bg-accent text-foreground font-medium'
            : 'text-dim hover:text-muted-foreground'
        "
        @click="selectMethod(null)"
      >
        All
      </button>
      <button
        v-for="method in HTTP_METHODS"
        :key="method"
        class="px-2.5 py-1.5 transition-colors"
        :class="
          activeMethod === method
            ? 'bg-accent text-foreground font-medium'
            : 'text-dim hover:text-muted-foreground'
        "
        @click="selectMethod(method)"
      >
        {{ method }}
      </button>
    </div>

    <!-- Path search -->
    <div class="relative flex-1 max-w-xs">
      <Search class="absolute left-3 top-1/2 -translate-y-1/2 text-dim h-3.5 w-3.5" />
      <input
        v-model="pathInput"
        type="text"
        placeholder="Filter by path..."
        class="w-full bg-card border border-border rounded-lg pl-8 pr-4 py-1.5 text-xs text-foreground placeholder:text-dim focus:outline-none focus:border-ring focus:ring-1 focus:ring-ring"
      />
    </div>

    <!-- Has diff toggle -->
    <label
      class="flex items-center gap-2 text-xs text-dim border border-border rounded-lg px-3 py-1.5 bg-card cursor-pointer select-none"
    >
      <Switch :checked="modelValue.hasDiff === true" @update:checked="onDiffToggle" />
      Has diff only
    </label>

    <!-- Sort -->
    <Select
      :model-value="modelValue.order === 'desc' ? 'newest' : 'oldest'"
      @update:model-value="onSortChange"
    >
      <SelectTrigger
        class="w-auto gap-1 text-xs text-dim border border-border rounded-lg px-3 py-1.5 bg-card h-auto"
      >
        <span class="text-dim">Sort:</span>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="newest">Newest first</SelectItem>
        <SelectItem value="oldest">Oldest first</SelectItem>
      </SelectContent>
    </Select>
  </div>
</template>
