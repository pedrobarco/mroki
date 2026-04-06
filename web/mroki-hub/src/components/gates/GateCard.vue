<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { truncateId } from '@/lib/utils'
import { ChevronRight } from 'lucide-vue-next'
import type { Gate } from '@/api'

interface Props {
  gate: Gate
}

const props = defineProps<Props>()
const router = useRouter()

function handleClick() {
  router.push(`/gates/${props.gate.id}`)
}

function formatRelativeTime(isoString: string | null): string {
  if (!isoString) return 'Never'
  const diff = Date.now() - new Date(isoString).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'Just now'
  if (mins < 60) return `${mins} min ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

const gateName = computed(() => props.gate.name)
const requests24h = computed(() => props.gate.stats.request_count_24h.toLocaleString())
const diffs = computed(() => props.gate.stats.diff_count_24h.toLocaleString())
const diffRate = computed(() => `${props.gate.stats.diff_rate.toFixed(1)}%`)
const lastActive = computed(() => formatRelativeTime(props.gate.stats.last_active))
const isActive = computed(() => props.gate.stats.last_active !== null)
</script>

<template>
  <div
    class="block bg-card border border-border rounded-xl p-5 cursor-pointer transition-colors hover:border-ring hover:bg-accent"
    @click="handleClick"
  >
    <!-- Top row: name + ID + status + last active -->
    <div class="flex items-start justify-between mb-4">
      <div class="flex items-center gap-3">
        <div
          class="w-9 h-9 rounded-lg bg-accent border border-border flex items-center justify-center"
        >
          <svg
            width="16"
            height="16"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            class="text-muted-foreground"
          >
            <path d="M12 2L2 7l10 5 10-5-10-5z" />
            <path d="M2 17l10 5 10-5" />
            <path d="M2 12l10 5 10-5" />
          </svg>
        </div>
        <div>
          <div class="flex items-center gap-2">
            <span class="font-semibold text-sm text-foreground">{{ gateName }}</span>
            <code class="text-xs font-mono text-dim bg-accent px-1.5 py-0.5 rounded">
              {{ truncateId(gate.id) }}
            </code>
            <span
              class="w-1.5 h-1.5 rounded-full"
              :class="isActive ? 'bg-success animate-pulse' : 'bg-dim'"
            />
          </div>
        </div>
      </div>
      <span class="text-xs" :class="isActive ? 'text-muted-foreground' : 'text-dim'">
        {{ lastActive }}
      </span>
    </div>

    <!-- Live / Shadow URLs -->
    <div class="grid grid-cols-2 gap-3 mb-4">
      <div class="bg-background/60 rounded-lg px-3 py-2.5 border border-border/50">
        <div class="text-xs uppercase tracking-widest text-dim mb-1.5 flex items-center gap-1.5">
          <span class="w-1.5 h-1.5 rounded-full bg-success" />
          Live
        </div>
        <code class="text-xs font-mono text-muted-foreground break-all leading-relaxed">
          {{ gate.live_url }}
        </code>
      </div>
      <div class="bg-background/60 rounded-lg px-3 py-2.5 border border-border/50">
        <div class="text-xs uppercase tracking-widest text-dim mb-1.5 flex items-center gap-1.5">
          <span class="w-1.5 h-1.5 rounded-full bg-info" />
          Shadow
        </div>
        <code class="text-xs font-mono text-muted-foreground break-all leading-relaxed">
          {{ gate.shadow_url }}
        </code>
      </div>
    </div>

    <!-- Footer stats -->
    <div class="flex items-center justify-between pt-3 border-t border-border/60">
      <div class="flex items-center gap-5 text-xs">
        <div>
          <span class="text-dim">Requests 24h</span>
          <span class="text-muted-foreground ml-1">{{ requests24h }}</span>
        </div>
        <div>
          <span class="text-dim">Diffs</span>
          <span class="text-muted-foreground ml-1">{{ diffs }}</span>
        </div>
        <div>
          <span class="text-dim">Diff rate</span>
          <span class="text-warning ml-1">{{ diffRate }}</span>
        </div>
      </div>
      <ChevronRight class="h-4 w-4 text-dim" />
    </div>
  </div>
</template>
