import { atom, action } from '@reatom/core';

import { tokenStore } from './token-store.ts';

/**
 * Session bookkeeping atoms — auth guard primitives.
 *
 * - `accessTokenAtom` mirrors the access token persisted in localStorage at the
 *   moment of the last `persistTokens` / `clearSession` call.
 * - `refreshTokenPresentAtom` reflects whether localStorage still has a refresh
 *   token (initialised on module load).
 *
 * The atoms are written explicitly inside `persistTokens` / `clearSession`
 * (which run in the reatom frame). The silent-refresh path in
 * `shared/api/client.ts` writes directly to `tokenStore` outside any frame,
 * so the atoms can briefly lag — but that's harmless: `peekIsAuthenticated()`
 * reads `tokenStore` synchronously and is the authoritative source for
 * `guard.ts`. A silent refresh keeps the user authenticated, so no
 * guard-driven reroute is required between the store write and the next
 * frame-driven atom set.
 *
 * NOTE: This is the "thin guard atom" referenced in the task spec. The actual
 * 401-then-refresh logic lives in `shared/api/client.ts`.
 */
export const accessTokenAtom = atom<string | null>(
  tokenStore.getAccess(),
  'auth.session.accessToken',
);

export const refreshTokenPresentAtom = atom<boolean>(
  Boolean(tokenStore.getRefresh()),
  'auth.session.refreshTokenPresent',
);

/** Persist tokens via the store and update guard atoms in one shot. */
export const persistTokens = action(async (access: string, refresh: string) => {
  tokenStore.setAccess(access);
  tokenStore.setRefresh(refresh);
  accessTokenAtom.set(access);
  refreshTokenPresentAtom.set(true);
}, 'auth.session.persistTokens');

/** Drop everything (logout / refresh failure). */
export const clearSession = action(async () => {
  tokenStore.clear();
  accessTokenAtom.set(null);
  refreshTokenPresentAtom.set(false);
}, 'auth.session.clearSession');

/**
 * Convenience guard — true while we have either an access token or a refresh
 * token persisted (the API client will swap the latter for a fresh access on
 * the first protected request).
 */
export const isAuthenticatedAtom = atom((): boolean => {
  return accessTokenAtom() !== null || refreshTokenPresentAtom();
}, 'auth.session.isAuthenticated');

/** Used by the rare imperative caller that needs the value without subscribing. */
export const peekIsAuthenticated = (): boolean => {
  return tokenStore.getAccess() !== null || tokenStore.getRefresh() !== null;
};
