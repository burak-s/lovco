import { defineConfig } from '@rsbuild/core';
import { pluginSvelte } from '@rsbuild/plugin-svelte';
import path from 'path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [pluginSvelte()],
  source: {
    alias: {
      '@components': path.resolve(__dirname, 'src/components'),
    },
  },

  server: {
    port: 3000,
  },

  html: {
    title: 'Leftover Service',
  },
});
