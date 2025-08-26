<template>
  <header class="w-full border-b border-zinc-200">
    <div class="flex items-center justify-between px-6 h-16">
      <!-- Left: Logo + Nav -->
      <div class="flex items-center gap-6">
        <!-- Logo / Brand -->
        <RouterLink
          to="/"
          class="flex items-center gap-2 font-semibold text-lg"
        >
          <span class="text-black dark:text-white">mroki</span>
          <Badge variant="outline" class="uppercase text-xs">HUB</Badge>
        </RouterLink>

        <!-- Navigation Links -->
        <nav
          class="flex items-center gap-6 text-sm text-zinc-600 dark:text-zinc-300"
        >
          <RouterLink
            v-for="route in routes"
            :to="route.path"
            class="flex items-center h-16 hover:text-black dark:hover:text-white transition-colors"
            active-class="font-weight-bold text-black dark:text-white"
          >
            {{ route.name }}
          </RouterLink>
        </nav>
      </div>

      <!-- Right: Buttons, User -->
      <div class="flex items-center gap-4"></div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { RouterLink } from "vue-router";
import { Badge } from "@/components/ui/badge";

interface Route {
  name: string;
  path: string;
}

const routes: Route[] = [
  { name: "Agents", path: "/agents" },
  { name: "Gates", path: "/gates" },
];
</script>
