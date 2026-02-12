<template>
  <div class="container mx-auto py-8">
    <h1 class="text-3xl font-bold mb-6">Request Detail</h1>
    <p class="text-gray-600">Gate ID: {{ route.params.id }}</p>
    <p class="text-gray-600">Request ID: {{ route.params.rid }}</p>
    <p class="text-gray-600 mt-4">Diff viewer will go here.</p>
  </div>
</template>

<script setup lang="ts">
import { useRoute } from 'vue-router'

// eslint-disable-next-line no-useless-assignment
const route = useRoute()
</script>
