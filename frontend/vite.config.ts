import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { defineConfig, loadEnv } from 'vite-plus';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

const __dirname = dirname(fileURLToPath(import.meta.url));

const env = loadEnv('development', process.cwd(), '');

// https://vite.dev/config/
export default defineConfig({
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: env.VITE_API_TARGET || 'http://127.0.0.1:8080',
        changeOrigin: true,
        secure: false,
      },
    },
  },
  staged: {
    '*': 'vp check --fix',
  },
  lint: {
    ignorePatterns: ['dist/**', 'node_modules/**', 'public/**'],
    options: { typeAware: true, typeCheck: true },
    plugins: ['react'],
  },
  build: {
    outDir: 'dist',
  },
  plugins: [react(), tailwindcss()],
  fmt: {
    ignorePatterns: ['dist/**', 'node_modules/**'],
    arrowParens: 'always',
    singleQuote: true,
    trailingComma: 'all',
    printWidth: 100,
  },
  resolve: {
    alias: {
      '@': resolve(__dirname, './src'),
    },
  },
});
