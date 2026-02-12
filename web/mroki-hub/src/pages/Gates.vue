<template>
  <div class="container mx-auto p-6">
    <!-- Page Header -->
    <div class="flex justify-between items-center mb-6">
      <div>
        <h1 class="text-3xl font-bold mb-2">Gates</h1>
        <p class="text-muted-foreground">
          Manage your live/shadow service pairs. Click on a gate to view captured requests.
        </p>
      </div>

      <!-- Create Gate Dialog -->
      <Dialog v-model:open="dialogOpen">
        <DialogTrigger as-child>
          <Button>Create Gate</Button>
        </DialogTrigger>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Create New Gate</DialogTitle>
          </DialogHeader>
          <GateForm @success="handleGateCreated" />
        </DialogContent>
      </Dialog>
    </div>

    <!-- Gates List -->
    <GateList :key="listKey" />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import GateList from '@/components/gates/GateList.vue'
import GateForm from '@/components/gates/GateForm.vue'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

const dialogOpen = ref(false)
const listKey = ref(0)

function handleGateCreated() {
  dialogOpen.value = false
  listKey.value++ // Force GateList to reload
}
</script>
