/**
 * Route registry — importing this module registers every page route with the
 * router. The order of imports is irrelevant; each page module exports its
 * `*Route` and the act of importing it is what attaches it to `rootRoute`.
 *
 * See `docs/design.md` for the canonical screen list. The `ROUTES` constant
 * lives in `shared/config/routes.ts` so widgets can reference it without
 * pulling in every page slice (which would create a circular import).
 */

import { authLoginRoute } from '@/pages/auth-login/index.ts';
import { authRegisterRoute } from '@/pages/auth-register/index.ts';
import { decksListRoute } from '@/pages/decks-list/index.ts';
import { accountSettingsRoute } from '@/pages/account-settings/index.ts';
import { deckCreateRoute } from '@/pages/deck-create/index.ts';
import { deckPracticeRoute } from '@/pages/deck-practice/index.ts';
import { practiceResultsRoute } from '@/pages/practice-results/index.ts';

export {
  authLoginRoute,
  authRegisterRoute,
  decksListRoute,
  accountSettingsRoute,
  deckCreateRoute,
  deckPracticeRoute,
  practiceResultsRoute,
};

export { ROUTES } from '@/shared/config/routes.ts';
