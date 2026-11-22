import { fileURLToPath } from 'node:url'
import { defineConfig, mergeConfig } from 'vite'
import appConfig from './vite.config'

export default mergeConfig(appConfig, defineConfig({
  root: fileURLToPath(new URL('./demo', import.meta.url)),
  base: process.env.PAGES_BASE_PATH || '/sing-box-easy/',
  publicDir: false,
  resolve: {
    alias: [{ find: /^(?:\.\.\/)+utils\/browserNetwork$/, replacement: fileURLToPath(new URL('./src/demo/browserNetwork.ts', import.meta.url)) }],
  },
  build: {
    outDir: fileURLToPath(new URL('../pages-dist', import.meta.url)),
    emptyOutDir: true,
  },
}))
