import { defineConfig } from 'vite';
import preact from '@preact/preset-vite';

export default defineConfig({
  base: '/static/dist/',
  plugins: [preact()],
  build: {
    outDir: './static/dist',
    assetsDir: '',
    sourcemap: true,
    manifest: true,
  },
});
