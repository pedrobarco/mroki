import path from 'node:path'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  test: {
    environment: 'happy-dom',
    include: ['src/**/*.spec.ts'],
    // Component specs import the real '@/api' barrel (for the query adapters and
    // types), which loads the API client and requires a base URL at import time.
    // Provide a dummy one so module init doesn't throw; the network itself is
    // always mocked in tests.
    env: {
      VITE_API_BASE_URL: 'http://localhost:8080',
      VITE_API_KEY: 'test-api-key-0000',
    },
  },
})
