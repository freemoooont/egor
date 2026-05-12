import { resolveApiBase } from '@/shared/config/index.ts';
import { tokenStore } from '@/shared/auth/index.ts';

const BASE_URL = resolveApiBase();

/**
 * How long the single-flight refresh slot is held past the refresh promise's
 * settlement. Sibling 401s that hadn't yet hit `attemptRefresh()` when we
 * kicked off the refresh have this window to share the same flight instead of
 * launching a duplicate `POST /auth/refresh` (which would rotate the token
 * family twice and trip `iam: refresh token reused` on the next call).
 */
const REFRESH_GRACE_MS = 150;

export class ApiError extends Error {
  readonly status: number;
  readonly body: unknown;
  /** Machine code from the canonical error envelope (`{ "error": "..." }`). */
  readonly code: string | null;
  constructor(status: number, message: string, body: unknown) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.body = body;
    const env = body as { error?: string } | null;
    this.code = env?.error ?? null;
  }
}

export interface ApiFetchInit extends RequestInit {
  /** Skip Authorization header even if a token is present (login/register). */
  anonymous?: boolean;
  /** Skip the 401-then-refresh-then-retry flow (used internally to avoid loops). */
  skipRefresh?: boolean;
  /** Idempotency-Key (ADR 0005) — propagated as `Idempotency-Key` header. */
  idempotencyKey?: string;
}

let refreshFlight: Promise<boolean> | null = null;

function buildHeaders(init: ApiFetchInit): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    Accept: 'application/json',
    ...((init.headers as Record<string, string> | undefined) ?? {}),
  };
  if (!init.anonymous) {
    const access = tokenStore.getAccess();
    if (access) headers['Authorization'] = `Bearer ${access}`;
  }
  if (init.idempotencyKey) headers['Idempotency-Key'] = init.idempotencyKey;
  return headers;
}

async function readJsonSafely(response: Response): Promise<unknown> {
  try {
    const text = await response.text();
    return text.length > 0 ? (JSON.parse(text) as unknown) : null;
  } catch {
    return null;
  }
}

function attemptRefresh(): Promise<boolean> {
  if (refreshFlight) return refreshFlight;

  const refresh = tokenStore.getRefresh();
  if (!refresh) return Promise.resolve(false);

  const flight = (async (): Promise<boolean> => {
    try {
      const response = await fetch(`${BASE_URL}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Accept: 'application/json' },
        body: JSON.stringify({ refreshToken: refresh }),
      });
      if (!response.ok) {
        tokenStore.clear();
        return false;
      }
      const body = (await response.json()) as {
        accessToken?: string;
        refreshToken?: string;
      };
      if (!body.accessToken || !body.refreshToken) {
        tokenStore.clear();
        return false;
      }
      tokenStore.setAccess(body.accessToken);
      tokenStore.setRefresh(body.refreshToken);
      return true;
    } catch {
      return false;
    }
  })();

  refreshFlight = flight;

  // Hold the flight slot for a small window past settle so any sibling 401
  // that hadn't yet landed when we kicked off the refresh joins THIS promise
  // instead of starting a duplicate POST /auth/refresh (which would rotate
  // the token family twice and trip "iam: refresh token reused" later).
  void flight.finally(() => {
    setTimeout(() => {
      if (refreshFlight === flight) refreshFlight = null;
    }, REFRESH_GRACE_MS);
  });

  return flight;
}

/**
 * `apiFetch` — thin wrapper around `fetch` that:
 *   1. prefixes the resolved API base URL;
 *   2. attaches the bearer token unless `anonymous: true`;
 *   3. on 401, attempts ONE silent refresh + retry (rotates the refresh family
 *      per ADR 0003);
 *   4. propagates `Idempotency-Key` (ADR 0005);
 *   5. throws an `ApiError` on non-2xx so `withAsync(Data)` can surface it.
 *
 * Use this from `computed(async () => ...).extend(withAsyncData())` for reads
 * and from `action(...).extend(withAsync())` for mutations — never imperative
 * `useEffect + fetch`. Reatom's `wrap` lives at the consumer side (forms /
 * `withAsyncData` / `withAsync`), not here.
 */
export async function apiFetch<T = unknown>(
  path: string,
  init: ApiFetchInit = {},
): Promise<T> {
  const { anonymous, headers: _h, skipRefresh, idempotencyKey: _i, ...rest } = init;
  void _h;
  void _i;

  const performRequest = (): Promise<Response> =>
    fetch(`${BASE_URL}${path}`, { ...rest, headers: buildHeaders(init) });

  let response = await performRequest();

  if (response.status === 401 && !skipRefresh && !anonymous) {
    if (await attemptRefresh()) response = await performRequest();
  }

  if (!response.ok) {
    const body = await readJsonSafely(response);
    const envMsg = (body as { message?: string } | null)?.message;
    throw new ApiError(
      response.status,
      envMsg ?? `API ${response.status}: ${response.statusText}`,
      body,
    );
  }
  if (response.status === 204) return undefined as T;
  return (await response.json()) as T;
}
