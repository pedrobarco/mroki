<script setup lang="ts">
import { Button } from '@/components/ui/button'
import { ChevronLeft } from 'lucide-vue-next'
</script>

<template>
  <div class="flex-1 flex items-center justify-center">
    <div class="text-center">
      <p
        class="text-9xl font-bold tracking-tighter leading-none text-foreground/5 select-none mb-2"
      >
        404
      </p>
      <h1 class="text-xl font-semibold tracking-tight -mt-6 mb-2">Page not found</h1>
      <p class="text-sm text-muted-foreground mb-8 max-w-xs mx-auto leading-relaxed">
        The page you're looking for doesn't exist or has been moved.
      </p>
      <router-link to="/gates">
        <Button class="gap-2">
          <ChevronLeft class="h-3.5 w-3.5" />
          Go to Gates
        </Button>
      </router-link>
    </div>
  </div>
</template>
