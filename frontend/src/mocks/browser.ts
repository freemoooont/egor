import { setupWorker } from 'msw/browser';

import { handlers } from './handlers.ts';

/**
 * MSW worker for dev. Started from `src/main.tsx` behind `import.meta.env.DEV`.
 * Handlers cover `/api/auth/{register,login,refresh}` and `/api/me` with an
 * in-memory user store and JWT-shaped (unsigned) tokens.
 */
export const worker = setupWorker(...handlers);
