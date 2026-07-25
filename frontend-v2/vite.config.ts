import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// 新版前端：构建到 frontend-dist/v2，通过 http://localhost:8080/v2/ 访问，
// 与旧版（http://localhost:8080/）并存，方便对比。
export default defineConfig({
  base: '/v2/',
  plugins: [vue()],
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    }
  },
  build: {
    outDir: '../backend/frontend-dist/v2',
    emptyOutDir: true,
  }
})
