<template>
  <footer class="border-t border-zinc-200 text-sm">
    <div
      class="flex flex-col md:flex-row items-center justify-between p-8 gap-4"
    >
      <!-- Left: Brand + Status -->
      <div class="flex items-center gap-2">
        <span class="font-semibold text-black dark:text-white">mroki</span>
        <span class="flex items-center gap-1 text-green-500">
          <span class="w-2 h-2 rounded-full bg-green-500"></span>
          All services are online
        </span>
      </div>

      <!-- Right: Copyright -->
      <div class="flex items-center gap-4">
        <a
          href="https://github.com/pedrobarco/mroki"
          target="_blank"
          aria-label="GitHub"
          class="text-zinc-500 hover:text-zinc-900"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            xml:space="preserve"
            viewBox="0 0 512 512"
            class="h-4 w-4 fill-current"
          >
            <path
              d="M256 6.3C114.6 6.3 0 120.9 0 262.3c0 113.3 73.3 209 175 242.9 12.8 2.2 17.6-5.4 17.6-12.2 0-6.1-.3-26.2-.3-47.7-64.3 11.8-81-15.7-86.1-30.1-2.9-7.4-15.4-30.1-26.2-36.2-9-4.8-21.8-16.6-.3-17 20.2-.3 34.6 18.6 39.4 26.2 23 38.7 59.8 27.8 74.6 21.1 2.2-16.6 9-27.8 16.3-34.2-57-6.4-116.5-28.5-116.5-126.4 0-27.8 9.9-50.9 26.2-68.8-2.6-6.4-11.5-32.6 2.6-67.8 0 0 21.4-6.7 70.4 26.2 20.5-5.8 42.2-8.6 64-8.6s43.5 2.9 64 8.6c49-33.3 70.4-26.2 70.4-26.2 14.1 35.2 5.1 61.4 2.6 67.8 16.3 17.9 26.2 40.6 26.2 68.8 0 98.2-59.8 120-116.8 126.4 9.3 8 17.3 23.4 17.3 47.4 0 34.2-.3 61.8-.3 70.4 0 6.7 4.8 14.7 17.6 12.2C438.7 471.3 512 375.3 512 262.3c0-141.4-114.6-256-256-256"
              style="fill-rule: evenodd; clip-rule: evenodd"
            />
          </svg>
        </a>
        <div class="text-zinc-500 text-xs text-center md:text-right">
          © 2025 mroki - All rights reserved.
        </div>
      </div>
    </div>
  </footer>
</template>
