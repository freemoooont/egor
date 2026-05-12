import { z } from 'zod';

/**
 * Environment schema validated at module load. Throws early in production
 * when a required variable is missing — the dev server keeps working with
 * the proxy default in `vite.config.ts`.
 */
const envSchema = z.object({
  VITE_API_TARGET: z.string().url().optional(),
  VITE_API_BASE: z.string().optional(),
  VITE_USE_MOCKS: z.string().optional(),
  MODE: z.enum(['development', 'production', 'test']).default('development'),
  DEV: z.boolean(),
  PROD: z.boolean(),
});

const rawEnv = {
  VITE_API_TARGET: import.meta.env.VITE_API_TARGET,
  VITE_API_BASE: import.meta.env.VITE_API_BASE,
  VITE_USE_MOCKS: import.meta.env.VITE_USE_MOCKS,
  MODE: import.meta.env.MODE as 'development' | 'production' | 'test',
  DEV: Boolean(import.meta.env.DEV),
  PROD: Boolean(import.meta.env.PROD),
};

export const env = envSchema.parse(rawEnv);

export function resolveApiBase(): string {
  if (env.VITE_API_BASE) return env.VITE_API_BASE;
  if (env.DEV) return '/api';
  if (env.VITE_API_TARGET) return `${env.VITE_API_TARGET}/api`;
  // Same-origin fallback (e.g. behind nginx that proxies /api)
  return '/api';
}
