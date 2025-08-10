<template>
  <DefaultLayout>
    <div class="container mx-auto p-4">
      <h1 class="text-2xl font-bold mb-4">Gates</h1>
      <GatesList :gates="gates" @select="handleSelectGate" />
    </div>
  </DefaultLayout>
</template>

<script setup lang="ts">
import { ref } from "vue";
import DefaultLayout from "@/layouts/DefaultLayout.vue";
import GatesList from "@/components/GatesList.vue";
import { router } from "@/router";
import type { Gate } from "@/types";

// TODO: replace with actual API call to fetch gates
const gates: Gate[] = [
  {
    id: "auth-gate",
    live_url: "https://live.example.com/auth",
    shadow_url: "https://shadow.example.com/auth",
    created_at: "",
  },
  {
    id: "search-gate",
    live_url: "https://live.example.com/search",
    shadow_url: "https://shadow.example.com/search-v2",
    created_at: "",
  },
  {
    id: "checkout-gate",
    live_url: "https://live.example.com/checkout",
    shadow_url: "https://shadow.example.com/checkout-canary",
    created_at: "",
  },
];

const selectedGate = ref<Gate | null>(null);

function handleSelectGate(gate: Gate) {
  selectedGate.value = gate;
  router.push({
    name: "Requests",
    params: { id: gate.id },
  });
}
</script>
