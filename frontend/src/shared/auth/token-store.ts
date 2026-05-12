/**
 * Auth token storage.
 *
 * Single source of truth: `localStorage` for BOTH access and refresh tokens.
 * No dev/prod branch, no in-memory copy, no `sessionStorage` path. A reload —
 * fresh tab or prod build — picks the access token straight back up so the
 * first protected request carries a Bearer header (no 401 → refresh round
 * trip every page load).
 *
 * The store also exposes a tiny `onChange` subscription so atoms (see
 * `session.ts`) can re-sync when `client.ts` mutates tokens from outside the
 * reatom frame (the silent refresh path).
 *
 * NOTE: This is infrastructure, not domain logic. Authentication flows
 * (login/register/refresh) live in `pages/auth-*` for now (FSD pages-first);
 * extract to `features/auth/` only when a 2nd consumer appears.
 */

const ACCESS_KEY = 'micocards.auth.access';
const REFRESH_KEY = 'micocards.auth.refresh';

const read = (key: string): string | null => {
  try {
    return globalThis.localStorage?.getItem(key) ?? null;
  } catch {
    return null;
  }
};

const write = (key: string, value: string | null): void => {
  try {
    if (value === null) globalThis.localStorage?.removeItem(key);
    else globalThis.localStorage?.setItem(key, value);
  } catch {
    /* private mode etc. */
  }
};

const subscribers = new Set<() => void>();
const notify = (): void => {
  for (const cb of subscribers) cb();
};

export const tokenStore = {
  getAccess: (): string | null => read(ACCESS_KEY),
  setAccess: (v: string | null): void => {
    write(ACCESS_KEY, v);
    notify();
  },
  getRefresh: (): string | null => read(REFRESH_KEY),
  setRefresh: (v: string | null): void => {
    write(REFRESH_KEY, v);
    notify();
  },
  clear: (): void => {
    write(ACCESS_KEY, null);
    write(REFRESH_KEY, null);
    notify();
  },
  /** Fires after every set/clear. Returns an unsubscribe handle. */
  onChange(cb: () => void): () => void {
    subscribers.add(cb);
    return () => {
      subscribers.delete(cb);
    };
  },
};
