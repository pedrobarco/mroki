<script setup lang="ts">
import { ref } from 'vue'
import GateList from '@/components/gates/GateList.vue'
import GateForm from '@/components/gates/GateForm.vue'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Plus, Search, ChevronDown } from 'lucide-vue-next'

const dialogOpen = ref(false)
const searchQuery = ref('')

const stats = [
  { label: 'TOTAL GATES', value: '4' },
  { label: 'ACTIVE AGENTS', value: '3' },
  { label: 'REQUESTS (24H)', value: '12,847' },
  { label: 'DIFF RATE', value: '4.2%', highlight: true },
]
const listKey = ref(0)

function handleGateCreated() {
  dialogOpen.value = false
  listKey.value++ // Force GateList to reload
}
</script>

<template>
  <div class="max-w-6xl mx-auto px-6 py-8">
    <!-- Page Header -->
    <div class="flex items-center justify-between mb-8">
      <div>
        <h1 class="text-xl font-semibold tracking-tight mb-1">Gates</h1>
        <p class="text-xs text-muted-foreground">
          Manage live/shadow service pairs and monitor traffic diffs.
        </p>
      </div>

      <!-- Create Gate Dialog -->
      <Dialog v-model:open="dialogOpen">
        <DialogTrigger as-child>
          <Button class="gap-2">
            <Plus class="h-3.5 w-3.5" />
            New Gate
          </Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Gate</DialogTitle>
            <DialogDescription>
              Enter the URLs for your live and shadow services to create a new gate.
            </DialogDescription>
          </DialogHeader>
          <GateForm @success="handleGateCreated" />
        </DialogContent>
      </Dialog>
    </div>

    <!-- Stats Bar -->
    <div class="grid grid-cols-4 gap-4 mb-6">
      <div
        v-for="stat in stats"
        :key="stat.label"
        class="bg-card border border-border rounded-xl px-4 py-3.5"
      >
        <div class="text-xs uppercase tracking-widest text-dim mb-1">{{ stat.label }}</div>
        <div
          class="text-lg font-semibold tracking-tight"
          :class="stat.highlight ? 'text-warning' : 'text-foreground'"
        >
          {{ stat.value }}
        </div>
      </div>
    </div>

    <!-- Search & Sort Row -->
    <div class="flex items-center gap-3 mb-5">
      <div class="relative flex-1 max-w-sm">
        <Search class="absolute left-3 top-1/2 -translate-y-1/2 text-dim h-3.5 w-3.5" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search gates..."
          class="w-full bg-card border border-border rounded-lg pl-8 pr-4 py-2 text-xs text-foreground placeholder:text-dim focus:outline-none focus:border-ring focus:ring-1 focus:ring-ring"
        />
      </div>
      <button
        class="flex items-center gap-1.5 text-xs text-dim border border-border rounded-lg px-3 py-2 bg-card hover:bg-accent transition-colors"
      >
        Sort by: Last active
        <ChevronDown class="h-3 w-3" />
      </button>
    </div>

    <!-- Gates List -->
    <GateList :key="listKey" />
  </div>
</template>
