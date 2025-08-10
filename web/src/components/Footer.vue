<template>
  <footer class="border-t border-zinc-200 text-sm">
    <div
      class="flex flex-col md:flex-row items-center justify-between px-6 py-4 gap-4"
    >
      <!-- Left: Brand + Status -->
      <div class="flex items-center gap-2">
        <span class="font-semibold text-black dark:text-white">mroki</span>
        <span class="flex items-center gap-1 text-green-500">
          <span class="w-2 h-2 rounded-full bg-green-500"></span>
          All services are online
        </span>
      </div>

      <!-- Middle: Social Links -->
      <div class="flex items-center gap-4 text-zinc-500">
        <a
          href="#"
          class="hover:text-zinc-900 dark:hover:text-white"
          aria-label="GitHub"
        >
          <i class="ri-github-fill text-lg"></i>
        </a>
        <a
          href="#"
          class="hover:text-zinc-900 dark:hover:text-white"
          aria-label="Twitter"
        >
          <i class="ri-twitter-fill text-lg"></i>
        </a>
        <a
          href="#"
          class="hover:text-zinc-900 dark:hover:text-white"
          aria-label="LinkedIn"
        >
          <i class="ri-linkedin-fill text-lg"></i>
        </a>
      </div>

      <!-- Right: Copyright -->
      <div class="flex items-center gap-2">
        <div class="text-zinc-500 text-xs text-center md:text-right">
          © 2025 mroki - All rights reserved.
        </div>
      </div>
    </div>
  </footer>
</template>
