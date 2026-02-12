<script setup lang="ts">
import { useRouter } from 'vue-router'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { truncateId } from '@/lib/utils'
import type { Gate } from '@/api'

interface Props {
  gate: Gate
}

const props = defineProps<Props>()
const router = useRouter()

function handleClick() {
  router.push(`/gates/${props.gate.id}`)
}
</script>

<template>
  <Card class="cursor-pointer transition-colors hover:bg-accent" @click="handleClick">
    <CardHeader>
      <CardTitle class="text-lg"> Gate {{ truncateId(gate.id) }} </CardTitle>
    </CardHeader>
    <CardContent class="space-y-2">
      <div>
        <span class="text-sm font-medium text-muted-foreground">Live:</span>
        <p class="text-sm text-foreground break-all">{{ gate.live_url }}</p>
      </div>
      <div>
        <span class="text-sm font-medium text-muted-foreground">Shadow:</span>
        <p class="text-sm text-foreground break-all">{{ gate.shadow_url }}</p>
      </div>
    </CardContent>
  </Card>
</template>
