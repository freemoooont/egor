/**
 * apiFetch single-flight refresh regression tests — Bug #2.
 *
 * Repro: cold start fires `GET /api/me` and `GET /api/decks` in parallel; both
 * 401; the previous implementation released the in-flight refresh slot on the
 * next microtask, so the second 401 saw `refreshFlight === null` and launched
 * a *second* refresh. Two refreshes rotated the token family twice; any third
 * call returned `iam: refresh token reused` and the SPA wedged.
 *
 * The fixed `attemptRefresh` keeps the flight slot held for the whole
 * lifetime of the in-flight promise, so every concurrent 401 awaits the same
 * promise and only ONE `/auth/refresh` POST hits the server.
 */

import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';

vi.stubGlobal('window', { location: { href: 'http://localhost/' } });

// Minimal in-memory localStorage stub so `tokenStore.getRefresh()` works.
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

const { tokenStore } = await import('@/shared/auth/index.ts');
const { apiFetch } = await import('./client.ts');

const originalFetch = globalThis.fetch;

interface RouteHandler {
  (input: RequestInfo | URL, init?: RequestInit): Promise<Response>;
}

const installFetch = (handler: RouteHandler): void => {
  globalThis.fetch = ((input, init) => handler(input, init)) as typeof fetch;
};

beforeAll(() => {
  // empty — placeholder so vitest sets up describe scope cleanly
});

afterEach(async () => {
  globalThis.fetch = originalFetch;
  tokenStore.clear();
  // Wait past the single-flight grace window (REFRESH_GRACE_MS=150ms in
  // client.ts) so the next test starts with a cleared `refreshFlight` slot
  // and doesn't accidentally short-circuit to the previous test's resolved
  // refresh promise.
  await new Promise((r) => setTimeout(r, 200));
});

describe('apiFetch single-flight refresh', () => {
  beforeEach(() => {
    tokenStore.clear();
    tokenStore.setRefresh('refresh-token-1');
  });

  it('coalesces concurrent 401s into a single /auth/refresh call', async () => {
    let refreshCalls = 0;
    let serial = 0;

    installFetch(async (input) => {
      const url = typeof input === 'string' ? input : input.toString();
      // Every protected request returns 401 the first time the access token is
      // missing; if the request comes back with a fresh access token it 200s.
      if (url.endsWith('/auth/refresh')) {
        refreshCalls++;
        // Simulate a server round-trip — give other 401s a chance to land
        // while the refresh is in flight.
        await new Promise((r) => setTimeout(r, 10));
        serial++;
        return new Response(
          JSON.stringify({
            accessToken: `access-${serial}`,
            refreshToken: `refresh-${serial + 1}`,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        );
      }
      const access = tokenStore.getAccess();
      if (!access) {
        return new Response(JSON.stringify({ error: 'unauthorized' }), {
          status: 401,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      return new Response(JSON.stringify({ ok: true, url }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      });
    });

    // Kick off three parallel protected GETs — none of them carry a bearer.
    const [me, decks, account] = await Promise.all([
      apiFetch<{ ok: boolean; url: string }>('/me'),
      apiFetch<{ ok: boolean; url: string }>('/decks'),
      apiFetch<{ ok: boolean; url: string }>('/account'),
    ]);

    expect(me.ok).toBe(true);
    expect(decks.ok).toBe(true);
    expect(account.ok).toBe(true);
    // Bug #2 regression: only ONE refresh call, even though three requests
    // 401'd concurrently.
    expect(refreshCalls).toBe(1);
  });

  it('runs a fresh refresh after the previous one settled', async () => {
    let refreshCalls = 0;

    installFetch(async (input) => {
      const url = typeof input === 'string' ? input : input.toString();
      if (url.endsWith('/auth/refresh')) {
        refreshCalls++;
        await new Promise((r) => setTimeout(r, 5));
        return new Response(
          JSON.stringify({
            accessToken: `access-${refreshCalls}`,
            refreshToken: `refresh-${refreshCalls + 1}`,
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        );
      }
      const access = tokenStore.getAccess();
      if (!access) {
        return new Response(JSON.stringify({ error: 'unauthorized' }), {
          status: 401,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    });

    await apiFetch<unknown>('/me');
    expect(refreshCalls).toBe(1);

    // Wipe the access token; force another 401 → refresh cycle. The
    // single-flight slot is held for REFRESH_GRACE_MS (150 ms) past settle
    // to coalesce sibling 401s, so we wait past that window before forcing
    // the next refresh.
    tokenStore.setAccess(null);
    await new Promise((r) => setTimeout(r, 200));
    await apiFetch<unknown>('/me');
    expect(refreshCalls).toBe(2);
  });

  it('does not retry forever when refresh keeps failing', async () => {
    let refreshCalls = 0;
    let protectedCalls = 0;

    installFetch(async (input) => {
      const url = typeof input === 'string' ? input : input.toString();
      if (url.endsWith('/auth/refresh')) {
        refreshCalls++;
        return new Response(JSON.stringify({ error: 'refresh_invalid' }), {
          status: 401,
          headers: { 'Content-Type': 'application/json' },
        });
      }
      protectedCalls++;
      return new Response(JSON.stringify({ error: 'unauthorized' }), {
        status: 401,
        headers: { 'Content-Type': 'application/json' },
      });
    });

    await expect(apiFetch('/me')).rejects.toThrow();
    // exactly one refresh attempt and exactly one initial protected call (no
    // post-failure retry storm).
    expect(refreshCalls).toBe(1);
    expect(protectedCalls).toBe(1);
  });
});
