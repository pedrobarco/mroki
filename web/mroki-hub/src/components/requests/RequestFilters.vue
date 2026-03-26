<script setup lang="ts">
import { ref, watch } from 'vue'
import type { RequestSortField, SortOrder } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

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

const HTTP_METHODS = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'] as const

const pathInput = ref(props.modelValue.path)

// Debounce path input so we don't fire on every keystroke
let pathTimeout: ReturnType<typeof setTimeout> | null = null
watch(pathInput, (val) => {
  if (pathTimeout) clearTimeout(pathTimeout)
  pathTimeout = setTimeout(() => {
    emitUpdate({ path: val })
  }, 400)
})

function toggleMethod(method: string) {
  const current = [...props.modelValue.methods]
  const idx = current.indexOf(method)
  if (idx >= 0) {
    current.splice(idx, 1)
  } else {
    current.push(method)
  }
  emitUpdate({ methods: current })
}

function isMethodActive(method: string): boolean {
  return props.modelValue.methods.includes(method)
}

function cycleDiffFilter() {
  const current = props.modelValue.hasDiff
  // Cycle: undefined → true → false → undefined
  if (current === undefined) {
    emitUpdate({ hasDiff: true })
  } else if (current === true) {
    emitUpdate({ hasDiff: false })
  } else {
    emitUpdate({ hasDiff: undefined })
  }
}

function diffFilterLabel(): string {
  const val = props.modelValue.hasDiff
  if (val === true) return 'Has diff'
  if (val === false) return 'No diff'
  return 'Any diff'
}

function onSortChange(val: string) {
  emitUpdate({ sort: val as RequestSortField })
}

function onOrderChange(val: string) {
  emitUpdate({ order: val as SortOrder })
}

function clearFilters() {
  pathInput.value = ''
  emitUpdate({
    methods: [],
    path: '',
    hasDiff: undefined,
    sort: 'created_at',
    order: 'desc',
  })
}

function hasActiveFilters(): boolean {
  const f = props.modelValue
  return (
    f.methods.length > 0 ||
    f.path !== '' ||
    f.hasDiff !== undefined ||
    f.sort !== 'created_at' ||
    f.order !== 'desc'
  )
}

function emitUpdate(partial: Partial<FilterState>) {
  emit('update:modelValue', { ...props.modelValue, ...partial })
}
</script>

<template>
  <div class="space-y-3">
    <!-- Row 1: Method toggles + Diff filter + Clear -->
    <div class="flex flex-wrap items-center gap-2">
      <span class="text-sm font-medium text-muted-foreground mr-1">Method:</span>
      <Badge
        v-for="method in HTTP_METHODS"
        :key="method"
        :variant="isMethodActive(method) ? 'default' : 'outline'"
        class="cursor-pointer select-none"
        @click="toggleMethod(method)"
      >
        {{ method }}
      </Badge>

      <div class="w-px h-6 bg-border mx-1" />

      <Badge
        :variant="modelValue.hasDiff !== undefined ? 'default' : 'outline'"
        class="cursor-pointer select-none"
        @click="cycleDiffFilter"
      >
        {{ diffFilterLabel() }}
      </Badge>

      <Button
        v-if="hasActiveFilters()"
        variant="ghost"
        size="sm"
        class="ml-auto text-muted-foreground"
        @click="clearFilters"
      >
        Clear filters
      </Button>
    </div>

    <!-- Row 2: Path search + Sort controls -->
    <div class="flex flex-wrap items-center gap-2">
      <Input
        v-model="pathInput"
        placeholder="Filter by path (e.g. /api/users)"
        class="max-w-xs h-8 text-sm"
      />

      <div class="w-px h-6 bg-border mx-1" />

      <span class="text-sm text-muted-foreground">Sort:</span>
      <Select :model-value="modelValue.sort" @update:model-value="onSortChange">
        <SelectTrigger size="sm" class="w-[130px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="created_at">Date</SelectItem>
          <SelectItem value="method">Method</SelectItem>
          <SelectItem value="path">Path</SelectItem>
        </SelectContent>
      </Select>

      <Select :model-value="modelValue.order" @update:model-value="onOrderChange">
        <SelectTrigger size="sm" class="w-[100px]">
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="desc">Newest</SelectItem>
          <SelectItem value="asc">Oldest</SelectItem>
        </SelectContent>
      </Select>
    </div>
  </div>
</template>
