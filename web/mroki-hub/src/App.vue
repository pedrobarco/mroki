<script setup lang="ts">
// mroki-hub - Web interface for visualizing diffs and managing gates
import Header from './components/layout/Header.vue'
import Footer from './components/layout/Footer.vue'
</script>

<template>
  <div id="app" class="min-h-screen flex flex-col">
    <Header />
    <main class="flex-1">
      <router-view />
    </main>
    <Footer />
  </div>
</template>

<style scoped>
#app {
  font-family:
    system-ui,
    -apple-system,
    sans-serif;
}
</style>
