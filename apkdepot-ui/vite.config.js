import { fileURLToPath, URL } from 'node:url'

import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    vue(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    proxy: {
      // 字符串简写写法
      // '/api': 'http://localhost:8080',
      // 选项写法
      '/api': {
        target: 'http://localhost:8080', // 你的 Go API 地址
        changeOrigin: true, // 必须设置为 true
        // 如果你的 API 路径没有 /api 前缀，可以用 rewrite 去掉
        // rewrite: (path) => path.replace(/^\/api/, '')
      }
    }
  }
})
