import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import Components from 'unplugin-vue-components/vite';
import { PrimeVueResolver } from '@primevue/auto-import-resolver';

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    Components({
      resolvers: [
        PrimeVueResolver()
      ]
    })
  ],
  build: {
    outDir: '../dist',
    emptyOutDir: true,
    // Increase chunk size warning limit since Monaco Editor is large
    chunkSizeWarningLimit: 1000,
    // Reduce parallelism for Docker builds to avoid memory issues
    sourcemap: false,
    rollupOptions: {
      maxParallelFileOps: 2,
      output: {
        // Manual chunking strategy
        manualChunks(id) {
          // Monaco Editor chunks
          if (id.includes('monaco-editor')) {
            // Split Monaco Editor by language/worker
            if (id.includes('/esm/vs/language/typescript')) {
              return 'monaco-typescript';
            }
            if (id.includes('/esm/vs/language/css')) {
              return 'monaco-css';
            }
            if (id.includes('/esm/vs/language/html')) {
              return 'monaco-html';
            }
            if (id.includes('/esm/vs/language/json')) {
              return 'monaco-json';
            }
            if (id.includes('/esm/vs/editor')) {
              return 'monaco-editor-core';
            }
            return 'monaco-vendor';
          }

          // Vue ecosystem
          if (id.includes('vue') || id.includes('@vue')) {
            return 'vue-vendor';
          }

          // Vue Router
          if (id.includes('vue-router')) {
            return 'vue-router';
          }

          // Pinia state management
          if (id.includes('pinia')) {
            return 'pinia';
          }

          // PrimeVue UI components
          if (id.includes('primevue') || id.includes('@primevue')) {
            return 'primevue-vendor';
          }

          // HeadlessUI components
          if (id.includes('@headlessui')) {
            return 'headlessui-vendor';
          }

          // Heroicons
          if (id.includes('@heroicons')) {
            return 'heroicons';
          }

          // Axios HTTP client
          if (id.includes('axios')) {
            return 'axios';
          }

          // DaisyUI and Tailwind utilities
          if (id.includes('daisyui')) {
            return 'daisyui';
          }

          // Other node_modules as vendor chunk
          if (id.includes('node_modules')) {
            return 'vendor';
          }
        },
        // Asset file naming
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) {
            return 'assets/css/[name]-[hash][extname]';
          }
          if (assetInfo.name?.match(/\.(woff2?|eot|ttf|otf)$/)) {
            return 'assets/fonts/[name]-[hash][extname]';
          }
          if (assetInfo.name?.match(/\.(png|jpe?g|gif|svg|ico)$/)) {
            return 'assets/images/[name]-[hash][extname]';
          }
          return 'assets/[name]-[hash][extname]';
        },
        // Chunk file naming
        chunkFileNames: 'assets/js/[name]-[hash].js',
        // Entry file naming
        entryFileNames: 'assets/js/[name]-[hash].js',
      },
    },
    // Minification options - use default esbuild for better performance
    minify: 'esbuild',
  },
  server: {
    proxy: {
      '/api': {
        target: 'http://localhost:5100',
        changeOrigin: true,
      },
    },
  },
  css: {
    postcss: {
      plugins: [],
    },
  }
})