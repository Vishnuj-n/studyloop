/// <reference types="vitest" />
/* eslint-env node */
import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  envDir: resolve(__dirname, '..'),
  plugins: [
    vue(),
    {
      name: 'wails-notebooks-fallback',
      configureServer(server) {
        server.middlewares.use((req, res, next) => {
          if (req.url && req.url.startsWith('/notebooks/')) {
            res.statusCode = 404
            res.end('Not Found')
            return
          }
          next()
        })
      }
    }
  ],
  optimizeDeps: {
    include: [
      'highlight.js/lib/core',
      'highlight.js/lib/languages/javascript',
      'highlight.js/lib/languages/typescript',
      'highlight.js/lib/languages/python',
      'highlight.js/lib/languages/go',
      'highlight.js/lib/languages/java',
      'highlight.js/lib/languages/cpp',
      'highlight.js/lib/languages/csharp',
      'highlight.js/lib/languages/sql',
      'highlight.js/lib/languages/bash',
      'highlight.js/lib/languages/json',
      'highlight.js/lib/languages/xml',
      'highlight.js/lib/languages/css',
    ],
  },
  server: {
    watch: {
      ignored: ['**/wailsjs/**', '**/dev_data/**', '**/*.db*'],
    },
  },
  build: {
    emptyOutDir: true
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  test: {
    globals: true,
    environment: 'jsdom',
    pool: 'vmThreads',
  }
})
