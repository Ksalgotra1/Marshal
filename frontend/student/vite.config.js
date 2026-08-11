import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, '.', '')
  const backendUrl = env.VITE_API_URL || 'http://localhost:8080'
  const port = env.PORT ? parseInt(env.PORT) : 5173

  return {
    plugins: [react(), tailwindcss()],
    server: {
      port: port,
      proxy: {
        '/api': backendUrl,
        '/events': backendUrl,
        '/healthz': backendUrl,
        '/ws': {
          target: backendUrl,
          ws: true,
        },
      },
    },
  }
})
