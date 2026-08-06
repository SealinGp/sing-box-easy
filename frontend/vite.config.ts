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
  define: {
    // App version: set by CI from the release tag (VITE_APP_VERSION), and
    // falls back to 'dev' for local builds. (package.json version is unused.)
    __APP_VERSION__: JSON.stringify(process.env.VITE_APP_VERSION || 'dev'),
  },
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
          // Monaco Editor chunks - simplified to avoid circular dependencies
          if (id.includes('monaco-editor')) {
            // Keep all Monaco editor code in a single chunk to avoid circular deps
            return 'monaco-editor';
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

          // Heroicons
          if (id.includes('@heroicons')) {
            return 'heroicons';
          }

          // Axios HTTP client
          if (id.includes('axios')) {
            return 'axios';
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
    // Default 5173 is taken by another project; pin this one to 5179.
    // strictPort avoids silently falling back to a different port.
    port: 5179,
    strictPort: true,
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