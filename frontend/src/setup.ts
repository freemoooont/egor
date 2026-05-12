import { clearStack, connectLogger, context } from '@reatom/core';

import { startAuthGuard } from '@/shared/auth/index.ts';

// Must be imported before any other reatom code!
clearStack();

export const rootFrame = context.start();

if (import.meta.env.DEV) {
  rootFrame.run(connectLogger);
}

// Auth guard — keep `/login` as the entry while no access token exists. The
// effect must run inside the root frame because reatom subscriptions are
// frame-scoped.
rootFrame.run(startAuthGuard);
