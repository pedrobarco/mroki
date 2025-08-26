<template>
  <div class="space-y-4">
    <Card
      v-for="gate in gates"
      :key="gate.id"
      @click="selectGate(gate)"
      class="p-4 space-y-3 cursor-pointer hover:bg-gray-50"
    >
      <!-- Top Row: Icon + Name + Status -->
      <div class="flex items-start justify-between">
        <div class="flex items-center space-x-4">
          <!-- Icon -->
          <div
            class="w-10 h-10 flex items-center justify-center rounded-md bg-black"
          >
            <svg
              xmlns="http://www.w3.org/2000/svg"
              width="24"
              height="24"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
              class="lucide lucide-waypoints-icon lucide-waypoints text-white"
            >
              <circle cx="12" cy="4.5" r="2.5" />
              <path d="m10.2 6.3-3.9 3.9" />
              <circle cx="4.5" cy="12" r="2.5" />
              <path d="M7 12h10" />
              <circle cx="19.5" cy="12" r="2.5" />
              <path d="m13.8 17.7 3.9-3.9" />
              <circle cx="12" cy="19.5" r="2.5" />
            </svg>
          </div>
          <!-- Title and Info -->
          <div>
            <CardTitle> {{ gate.id }} </CardTitle>
            <p class="text-xs text-gray-500 mt-1">
              Live URL: <span class="font-mono"> {{ gate.live_url }} </span>
            </p>
            <p class="text-xs text-gray-500">
              Shadow URL: <span class="font-mono"> {{ gate.shadow_url }} </span>
            </p>
          </div>
        </div>

        <!-- Status Last 5 Days -->
        <div class="flex space-x-1 pt-1">
          <template v-for="i in 5" :key="i">
            <div
              :class="[
                'w-2 h-2 rounded-full',
                i <= 3 ? 'bg-green-500' : 'bg-green-300',
              ]"
            ></div>
          </template>
        </div>
      </div>

      <!-- % Live vs Shadow -->
      <div class="flex space-x-1">
        <template v-for="i in 10" :key="i">
          <div
            :class="['w-2 h-2 rounded-sm', i <= 5 ? 'bg-black' : 'bg-gray-300']"
          ></div>
        </template>
      </div>
    </Card>
  </div>
</template>

<script setup lang="ts">
import { Card, CardTitle } from "@/components/ui/card";
import type { Gate } from "@/types";

const { gates } = defineProps<{
  gates: Gate[];
}>();

function selectGate(gate: Gate) {
  emit("select", gate);
}

const emit = defineEmits(["select"]);
</script>
