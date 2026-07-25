import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 新版前端（已采用）：构建到 frontend-dist 根，通过 http://localhost:8080/ 访问。
export default defineConfig({
  plugins: [vue()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    }
  },
  build: {
    outDir: '../backend/frontend-dist',
    emptyOutDir: true,
  }
})
