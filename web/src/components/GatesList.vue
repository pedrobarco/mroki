<template>
  <div class="space-y-4">
    <Card
      v-for="gate in gates"
      :key="gate.id"
      class="cursor-pointer hover:shadow transition"
      @click="() => selectGate(gate)"
    >
      <CardHeader>
        <!-- Icon should be added on the left side -->
        <div class="flex items-center space-x-4">
          <div class="p-2 rounded-md bg-black">
            <!-- Placeholder for Gate Icon -->
            <img
              src="@/assets/server-proxy-svgrepo-com.svg"
              alt="Gate Icon"
              class="w-5 h-5"
            />
          </div>
          <CardTitle class="text-md">{{ gate.id }}</CardTitle>
        </div>
        <!-- Description should be on the right side -->
        <CardDescription class="mt-1">
          Live: {{ gate.live_url }}<br />
          Shadow: {{ gate.shadow_url }}
        </CardDescription>
      </CardHeader>
    </Card>
  </div>
</template>

<script setup lang="ts">
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
} from "@/components/ui/card";
import type { Gate } from "@/types";

const { gates } = defineProps<{
  gates: Gate[];
}>();

function selectGate(gate: Gate) {
  emit("select", gate);
}

const emit = defineEmits(["select"]);
</script>
