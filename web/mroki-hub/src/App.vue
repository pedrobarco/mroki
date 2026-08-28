<script setup lang="ts">
// mroki-hub - Web interface for visualizing diffs and managing gates
import { defineAsyncComponent } from 'vue'
import Header from './components/layout/Header.vue'

// Dev-only: async import keeps the devtools out of the production bundle.
const VueQueryDevtools = import.meta.env.DEV
  ? defineAsyncComponent(() =>
      import('@tanstack/vue-query-devtools').then((m) => m.VueQueryDevtools)
    )
  : null
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <Header />
    <main class="flex-1">
      <router-view />
    </main>
    <VueQueryDevtools v-if="VueQueryDevtools" />
  </div>
</template>
