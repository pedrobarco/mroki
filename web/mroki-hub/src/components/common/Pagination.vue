<script setup lang="ts">
interface Props {
  currentPage: number
  totalPages: number
  disabledPrev?: boolean
  disabledNext?: boolean
}

defineProps<Props>()
const emit = defineEmits<{
  (e: 'prev'): void
  (e: 'next'): void
}>()
</script>

<template>
  <div v-if="totalPages > 1" class="mt-4 flex items-center justify-between text-xs">
    <span class="text-dim">Page {{ currentPage }} of {{ totalPages }}<slot name="meta" /></span>
    <div class="flex items-center gap-1">
      <button
        class="inline-flex min-h-[44px] min-w-[44px] items-center justify-center rounded-lg border border-border bg-card px-4 py-2.5 transition-colors focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring"
        :class="
          disabledPrev
            ? 'text-dim opacity-40 cursor-not-allowed'
            : 'text-muted-foreground hover:bg-accent'
        "
        :disabled="disabledPrev"
        @click="emit('prev')"
      >
        Previous
      </button>
      <span
        class="inline-flex min-h-[44px] min-w-[44px] items-center justify-center rounded-lg border border-border bg-accent px-4 py-2.5 font-medium text-foreground"
      >
        {{ currentPage }}
      </span>
      <button
        class="inline-flex min-h-[44px] min-w-[44px] items-center justify-center rounded-lg border border-border bg-card px-4 py-2.5 transition-colors focus-visible:outline-none focus-visible:ring-[3px] focus-visible:ring-ring"
        :class="
          disabledNext
            ? 'text-dim opacity-40 cursor-not-allowed'
            : 'text-muted-foreground hover:bg-accent'
        "
        :disabled="disabledNext"
        @click="emit('next')"
      >
        Next
      </button>
    </div>
  </div>
</template>
