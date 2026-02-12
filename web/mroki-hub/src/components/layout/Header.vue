<template>
  <header class="border-b bg-background">
    <div class="container mx-auto px-4 py-4 flex items-center justify-between">
      <div class="flex items-center gap-8">
        <router-link to="/" class="flex items-center gap-2">
          <div class="text-2xl font-bold text-primary">mroki</div>
          <div class="text-sm text-muted-foreground">hub</div>
        </router-link>

        <nav class="flex gap-6">
          <router-link
            to="/gates"
            class="text-foreground hover:text-primary transition-colors"
            active-class="text-primary font-semibold"
          >
            Gates
          </router-link>
        </nav>
      </div>

      <div class="text-sm text-muted-foreground">Shadow Traffic Analysis</div>
    </div>
  </header>
</template>

<script setup lang="ts">
// Header component with navigation
</script>
