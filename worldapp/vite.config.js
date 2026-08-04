import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  server: {
    port: 5175,
    proxy: {
      '/api': {
        target: 'http://localhost:48092',
        changeOrigin: true
      }
    }
  },
  base: './',
  build: {
    outDir: '../uiteg',
    emptyOutDir: true
  }
});