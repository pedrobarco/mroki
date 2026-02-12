<template>
  <div class="container mx-auto py-8">
    <h1 class="text-3xl font-bold mb-6">Request Detail</h1>
    <p class="text-muted-foreground">Gate ID: {{ route.params.id }}</p>
    <p class="text-muted-foreground">Request ID: {{ route.params.rid }}</p>
    <p class="text-muted-foreground mt-4">Diff viewer will go here.</p>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'

// eslint-disable-next-line no-useless-assignment
const route = useRoute()
</script>
