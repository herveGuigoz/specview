import { defineConfig } from 'vite'
export default defineConfig({
  build: { outDir: '../internal/web/dist', emptyOutDir: true, target: 'es2022' },
})
