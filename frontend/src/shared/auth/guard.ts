import { effect, urlAtom } from '@reatom/core';

import { isAuthenticatedAtom, peekIsAuthenticated } from './session.ts';

/**
 * Public auth routes — paths that anonymous users may visit. Anything else is
 * gated by the guard below. Keep in sync with `app/routes.ts`.
 */
const PUBLIC_PATHS = new Set(['/login', '/register']);

/**
 * Auth guard — call once inside the reatom root frame (see `src/setup.ts`).
 *
 * When the user lands on `/` (or any non-public path) without a token, replace
 * the URL with `/login`. When an authenticated user lands on `/login` or
 * `/register`, send them to `/decks`. Subscribes to both `urlAtom` and
 * `isAuthenticatedAtom` so a subsequent login/logout naturally re-routes.
 *
 * Returns the underlying `effect` for symmetry; the side-effect of subscribing
 * is what guards the app.
 *
 * IMPORTANT: the authoritative auth check reads `tokenStore` synchronously via
 * `peekIsAuthenticated()`. Reading `isAuthenticatedAtom()` exists only to
 * subscribe to logout/login transitions — but during a login submit handler
 * the page-level `urlAtom.go(ROUTES.decks)` runs in the same frame as the
 * `persistTokens` action that sets the auth atoms, and the effect would
 * otherwise observe a stale atom snapshot and bounce the user back to
 * `/login`. The token store is updated synchronously in `persistTokens`, so
 * the peek is the trustworthy source of truth.
 */
export function startAuthGuard(): ReturnType<typeof effect> {
  return effect(() => {
    const url = urlAtom();
    const path = url.pathname;
    // Read the atom to subscribe — logout / async refresh failure flips this
    // and we need to react to it. The result alone isn't trustworthy in the
    // login-just-succeeded frame (Bug #1), so we OR it with the synchronous
    // tokenStore peek which is updated immediately by `persistTokens`.
    const atomAuthed = isAuthenticatedAtom();
    const peekAuthed = peekIsAuthenticated();
    // Authoritative answer: authed iff EITHER source thinks so. After
    // `persistTokens` the peek is true even on the first effect frame; after
    // `clearSession` both are false so logout still reroutes.
    const authed = atomAuthed || peekAuthed;

    if (!authed && !PUBLIC_PATHS.has(path)) {
      urlAtom.go('/login', /* replace */ true);
      return;
    }
    if (authed && PUBLIC_PATHS.has(path)) {
      urlAtom.go('/decks', /* replace */ true);
    }
  }, 'auth.guard');
}
