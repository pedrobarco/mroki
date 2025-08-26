<template>
  <DefaultLayout>
    <div class="container mx-auto p-4">
      <template v-if="agents.length > 0">
        <h1 class="text-2xl font-semibold mb-4">Agents</h1>
        <AgentsTable :agents="agents" />
      </template>
      <template v-else>
        <div
          class="flex items-center justify-center min-h-96 bg-white border border-gray-200 rounded-md"
        >
          <div class="text-center space-y-6">
            <!-- Icon -->
            <div class="flex justify-center">
              <div class="bg-gray-100 p-4 rounded-full">
                <!-- You can replace this with an actual icon component -->
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-8 w-8 text-gray-500"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M9 2v2m6-2v2M4 9h2m12 0h2M4 15h2m12 0h2M9 20v2m6-2v2M8 12a4 4 0 108 0 4 4 0 00-8 0z"
                  />
                </svg>
              </div>
            </div>

            <!-- Title -->
            <h2 class="text-2xl font-semibold">Connect New Agent</h2>

            <!-- Description -->
            <p class="text-gray-500 max-w-md mx-auto">
              Connect running agents to your application to start managing and
              monitoring them.
            </p>

            <!-- Button -->
            <Button @click="onConnectNewWorker" variant="outline">
              Connect New Worker
            </Button>
          </div>
        </div>
      </template>
    </div>
  </DefaultLayout>
</template>

<script setup lang="ts">
import DefaultLayout from "@/layouts/DefaultLayout.vue";
import AgentsTable from "@/components/AgentsTable.vue";
import type { Agent } from "@/types";
import { Button } from "@/components/ui/button";
import { ref } from "vue";

const agents = ref<Agent[]>([]);

function onConnectNewWorker() {
  agents.value = [
    {
      id: "7208C391-7A17-4482-848B-9ADE67156A56",
      name: "agent-paul-smith",
      status: "Idle",
    },
    {
      id: "0BF2CC85-3CE2-4B48-AF5E-879D76F614AA",
      name: "agent-johnson",
      status: "Failed",
    },
    {
      id: "9CF01DE7-018A-4BEC-B0AA-9B28429A5989",
      name: "agent-brown",
      status: "Running",
    },
  ];
}
</script>
