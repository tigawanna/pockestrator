import { defineConfig } from 'vite';
import preact from '@preact/preset-vite';

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [preact()],
  build: {
    // Output to the root of pb_public so PocketBase can serve it
    outDir: './',
    emptyOutDir: false, // Don't delete the entire pb_public directory
  },
  server: {
    proxy: {
      // Proxy API requests to the PocketBase backend during development
      '/api': {
        target: 'http://localhost:8090',
        changeOrigin: true,
      },
    },
  },
});