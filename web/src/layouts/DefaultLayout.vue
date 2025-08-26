<template>
  <div class="flex flex-col min-h-screen">
    <TopNav />
    <main class="flex-1 px-6 py-4 bg-gray-50">
      <div class="max-w-7xl mx-auto">
        <slot />
      </div>
    </main>
    <Footer />
  </div>
</template>

<script setup lang="ts">
import TopNav from "@/components/TopNav.vue";
import Footer from "@/components/Footer.vue";
</script>
