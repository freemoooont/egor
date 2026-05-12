/**
 * Auth guard regression tests — Bug #1.
 *
 * The login/register submit handler does:
 *
 *   await wrap(persistTokens(access, refresh));
 *   urlAtom.go(ROUTES.decks);
 *
 * Before the fix, the guard read `isAuthenticatedAtom()` (a computed reading
 * `accessTokenAtom`/`refreshTokenPresentAtom`) which could observe a stale
 * snapshot on the very next effect frame and immediately revert the URL back
 * to `/login`. The fix has the guard trust `peekIsAuthenticated()` (the
 * synchronous tokenStore-backed view) so a fresh login/register navigation is
 * not bounced.
 *
 * Reatom's `urlAtom` lives in `@reatom/core/web` and uses `window.location`
 * + `history.pushState`. Vitest defaults to a Node environment here (no
 * jsdom installed); we stub the minimum surface so the atom can flip the
 * pathname without throwing.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

let currentHref = 'http://localhost/login';

const fakeLocation = {
  get href() {
    return currentHref;
  },
  set href(next: string) {
    currentHref = new URL(next, currentHref).href;
  },
  get origin() {
    return new URL(currentHref).origin;
  },
};

const fakeHistory = {
  pushState: (_state: unknown, _title: string, url: string) => {
    currentHref = new URL(url, currentHref).href;
  },
  replaceState: (_state: unknown, _title: string, url: string) => {
    currentHref = new URL(url, currentHref).href;
  },
};

const fakeBody = { addEventListener: () => undefined, removeEventListener: () => undefined };

vi.stubGlobal('window', {
  location: fakeLocation,
  history: fakeHistory,
  document: { body: fakeBody },
  addEventListener: () => undefined,
  removeEventListener: () => undefined,
});
vi.stubGlobal('history', fakeHistory);

// Minimal in-memory localStorage stub so `tokenStore.{get,set}{Access,Refresh}`
// works in the Node test environment (token-store.ts persists everything to
// localStorage now).
const memStorage = (() => {
  const store = new Map<string, string>();
  return {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => {
      store.set(key, String(value));
    },
    removeItem: (key: string) => {
      store.delete(key);
    },
    clear: () => store.clear(),
    key: (index: number) => Array.from(store.keys())[index] ?? null,
    get length() {
      return store.size;
    },
  };
})();
vi.stubGlobal('localStorage', memStorage);

// Imports MUST come after the global stubs above.
const { clearStack, context, peek, urlAtom, wrap } = await import('@reatom/core');
const { persistTokens } = await import('./session.ts');
const { tokenStore } = await import('./token-store.ts');
const { startAuthGuard } = await import('./guard.ts');

clearStack();

const setHref = (path: string): void => {
  currentHref = new URL(path, 'http://localhost/').href;
};

const tick = (): Promise<void> =>
  // setTimeout(_, 0) is what `urlAtom.sync` uses to push to history; resolving
  // a real macrotask plus a couple of microtasks gives effects + reads a
  // chance to settle.
  wrap(new Promise<void>((resolve) => setTimeout(resolve, 5)));

describe('startAuthGuard', () => {
  let frame: ReturnType<typeof context.start>;

  beforeEach(() => {
    tokenStore.clear();
    setHref('/login');
    frame = context.start();
  });

  afterEach(() => {
    tokenStore.clear();
  });

  it('redirects anonymous users away from protected routes', async () => {
    setHref('/decks');
    await frame.run(async () => {
      urlAtom.go('/decks', /* replace */ true);
      startAuthGuard();
      await tick();
      const path = peek(urlAtom).pathname;
      expect(path).toBe('/login');
    });
  });

  it('does NOT redirect to /login when authentication just succeeded', async () => {
    await frame.run(async () => {
      // Start on /login, like the user landing on the login page.
      urlAtom.go('/login', /* replace */ true);
      startAuthGuard();
      await tick();
      expect(peek(urlAtom).pathname).toBe('/login');

      // Mimic the loginForm submit handler order of operations.
      await wrap(persistTokens('access-token-xyz', 'refresh-token-xyz'));
      urlAtom.go('/decks');
      await tick();

      // Bug #1 regression: the guard must not bounce us back to /login.
      expect(peek(urlAtom).pathname).toBe('/decks');
    });
  });

  it('redirects an authenticated user away from /login', async () => {
    await frame.run(async () => {
      // Mimic a user who refreshes the page while already logged in.
      tokenStore.setAccess('access');
      tokenStore.setRefresh('refresh');
      urlAtom.go('/login', /* replace */ true);
      startAuthGuard();
      await tick();
      expect(peek(urlAtom).pathname).toBe('/decks');
    });
  });

  it('does not redirect when an authed user navigates between protected routes', async () => {
    // After Bug #1's fix, the guard reading the synchronous tokenStore peek
    // means subsequent in-app navigations between protected routes never
    // bounce back to /login.
    await frame.run(async () => {
      await wrap(persistTokens('access', 'refresh'));
      urlAtom.go('/decks');
      startAuthGuard();
      await tick();
      expect(peek(urlAtom).pathname).toBe('/decks');

      urlAtom.go('/account');
      await tick();
      expect(peek(urlAtom).pathname).toBe('/account');

      urlAtom.go('/decks/new');
      await tick();
      expect(peek(urlAtom).pathname).toBe('/decks/new');
    });
  });
});
