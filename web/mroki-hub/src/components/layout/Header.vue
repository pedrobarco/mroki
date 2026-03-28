<script setup lang="ts">
import { useRoute } from 'vue-router'

const route = useRoute()

const navItems = [
  { label: 'Gates', to: '/gates', matchPrefix: '/gates' },
  { label: 'Agents', to: '#', matchPrefix: '/agents' },
  { label: 'Requests', to: '#', matchPrefix: '/requests' },
  { label: 'Settings', to: '#', matchPrefix: '/settings' },
]

function isActive(matchPrefix: string): boolean {
  return route.path === matchPrefix || route.path.startsWith(matchPrefix + '/')
}
</script>

<template>
  <header class="border-b border-border sticky top-0 z-50 bg-background/80 backdrop-blur-md">
    <div class="max-w-6xl mx-auto flex items-center justify-between h-14 px-6">
      <div class="flex items-center gap-6">
        <router-link to="/" class="flex items-center gap-2">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" class="text-foreground">
            <path
              d="M12 2L2 7l10 5 10-5-10-5zM2 17l10 5 10-5M2 12l10 5 10-5"
              stroke="currentColor"
              stroke-width="1.8"
              stroke-linecap="round"
              stroke-linejoin="round"
            />
          </svg>
          <span class="font-semibold text-sm tracking-tight text-foreground">mroki hub</span>
        </router-link>

        <nav class="flex items-center gap-1 text-xs">
          <router-link
            v-for="item in navItems"
            :key="item.label"
            :to="item.to"
            class="px-3 py-1.5 rounded-md transition-colors"
            :class="
              isActive(item.matchPrefix)
                ? 'bg-accent text-foreground font-medium'
                : 'text-muted-foreground hover:text-accent-foreground'
            "
          >
            {{ item.label }}
          </router-link>
        </nav>
      </div>

      <div class="flex items-center gap-3">
        <div
          class="flex items-center gap-1.5 text-xs text-success bg-success-dim/30 px-2.5 py-1 rounded-full"
        >
          <span class="w-1.5 h-1.5 rounded-full bg-success animate-pulse" />
          <span>API Connected</span>
        </div>
        <div
          class="w-7 h-7 rounded-full bg-accent border border-border flex items-center justify-center text-xs font-medium text-muted-foreground"
        >
          DK
        </div>
      </div>
    </div>
  </header>
</template>
