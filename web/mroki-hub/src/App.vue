<script setup lang="ts">
// mroki-hub - Web interface for visualizing diffs and managing gates
import Header from './components/layout/Header.vue'
</script>

<template>
  <div class="min-h-screen flex flex-col">
    <Header />
    <main class="flex-1">
      <router-view />
    </main>
  </div>
</template>
