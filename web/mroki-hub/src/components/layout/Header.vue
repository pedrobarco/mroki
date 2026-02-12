<template>
  <header class="border-b bg-white">
    <div class="container mx-auto px-4 py-4 flex items-center justify-between">
      <div class="flex items-center gap-8">
        <router-link to="/" class="flex items-center gap-2">
          <div class="text-2xl font-bold text-blue-600">mroki</div>
          <div class="text-sm text-gray-500">hub</div>
        </router-link>

        <nav class="flex gap-6">
          <router-link
            to="/gates"
            class="text-gray-700 hover:text-blue-600 transition-colors"
            active-class="text-blue-600 font-semibold"
          >
            Gates
          </router-link>
        </nav>
      </div>

      <div class="text-sm text-gray-500">Shadow Traffic Analysis</div>
    </div>
  </header>
</template>

<script setup lang="ts">
// Header component with navigation
</script>
