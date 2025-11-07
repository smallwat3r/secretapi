import { defineConfig } from 'vite';
import preact from '@preact/preset-vite';

export default defineConfig(({ mode }) => ({
  base: '/static/dist/',
  plugins: [preact()],
  build: {
    outDir: './static/dist',
    assetsDir: '',
    sourcemap: mode !== 'production',
    manifest: true,
  },
}));
