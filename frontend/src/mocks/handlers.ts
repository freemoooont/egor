import { http, HttpResponse } from 'msw';

/**
 * MSW handlers — auth + me + decks, in-memory store with localStorage persistence.
 *
 * Used in dev only (the worker is started from `src/main.tsx` behind an
 * `import.meta.env.DEV` check) and never bundled into production builds.
 *
 * Tokens are JWT-shaped (header.payload.signature) but the signature is the
 * literal string `dev`. They are decodable with `atob` for inspection but not
 * cryptographically valid — production traffic hits the real Go backend.
 *
 * Data persists in `localStorage` under `micocards.mocks.*` so a refresh keeps
 * decks, cards, and profile edits.
 */

interface MockUser {
  id: string;
  email: string;
  password: string;
  displayName: string;
  avatarRef: string | null;
  registeredAt: string;
}

interface MockCard {
  id: string;
  term: string;
  definition: string;
  ordinal: number;
}

interface MockDeck {
  id: string;
  ownerId: string;
  title: string;
  authorName: string;
  createdAt: string;
  cards: MockCard[];
}

type MockPracticeMode = 'tracked' | 'untracked';
type MockRating = 0 | 1 | 2;

interface MockPracticeSession {
  id: string;
  deckId: string;
  ownerId: string;
  mode: MockPracticeMode;
  startedAt: string;
  completedAt: string | null;
}

interface MockPracticeRating {
  sessionId: string;
  deckId: string;
  cardId: string;
  rating: MockRating;
  ratedAt: string;
}

const USERS_KEY = 'micocards.mocks.users';
const DECKS_KEY = 'micocards.mocks.decks';
const REFRESH_KEY = 'micocards.mocks.refreshTokens';
const SESSIONS_KEY = 'micocards.mocks.practiceSessions';
const RATINGS_KEY = 'micocards.mocks.practiceRatings';

function loadFromStorage<T>(key: string, fallback: T): T {
  try {
    const raw = globalThis.localStorage?.getItem(key);
    if (!raw) return fallback;
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

function saveToStorage(key: string, value: unknown): void {
  try {
    globalThis.localStorage?.setItem(key, JSON.stringify(value));
  } catch {
    // localStorage may be blocked; silent failure is fine.
  }
}

const users: MockUser[] = loadFromStorage<MockUser[]>(USERS_KEY, []);
const decks: MockDeck[] = loadFromStorage<MockDeck[]>(DECKS_KEY, []);
const sessions: MockPracticeSession[] = loadFromStorage<MockPracticeSession[]>(SESSIONS_KEY, []);
const ratings: MockPracticeRating[] = loadFromStorage<MockPracticeRating[]>(RATINGS_KEY, []);
const refreshTokens = new Map<string, { userId: string; familyId: string }>(
  loadFromStorage<Array<[string, { userId: string; familyId: string }]>>(REFRESH_KEY, []),
);

function persistRefreshTokens(): void {
  saveToStorage(REFRESH_KEY, Array.from(refreshTokens.entries()));
}

/**
 * Re-hydrate refresh tokens from localStorage. Called at the top of every
 * /auth/refresh request so a page reload that mints new tokens via login then
 * navigates can still validate older tokens. (HMR can also reset module
 * state.)
 */
function hydrateRefreshTokens(): void {
  const stored = loadFromStorage<Array<[string, { userId: string; familyId: string }]>>(
    REFRESH_KEY,
    [],
  );
  for (const [k, v] of stored) {
    if (!refreshTokens.has(k)) refreshTokens.set(k, v);
  }
}

function hydrateUsers(): void {
  const stored = loadFromStorage<MockUser[]>(USERS_KEY, []);
  for (const user of stored) {
    if (!users.some((u) => u.id === user.id)) users.push(user);
  }
}

function hydrateDecks(): void {
  const stored = loadFromStorage<MockDeck[]>(DECKS_KEY, []);
  for (const deck of stored) {
    if (!decks.some((d) => d.id === deck.id)) decks.push(deck);
  }
}

function persistUsers(): void {
  saveToStorage(USERS_KEY, users);
}

function persistDecks(): void {
  saveToStorage(DECKS_KEY, decks);
}

function hydrateSessions(): void {
  const stored = loadFromStorage<MockPracticeSession[]>(SESSIONS_KEY, []);
  for (const s of stored) {
    if (!sessions.some((x) => x.id === s.id)) sessions.push(s);
  }
}

function hydrateRatings(): void {
  const stored = loadFromStorage<MockPracticeRating[]>(RATINGS_KEY, []);
  for (const r of stored) {
    if (!ratings.some((x) => x.sessionId === r.sessionId && x.cardId === r.cardId && x.ratedAt === r.ratedAt)) {
      ratings.push(r);
    }
  }
}

function persistSessions(): void {
  saveToStorage(SESSIONS_KEY, sessions);
}

function persistRatings(): void {
  saveToStorage(RATINGS_KEY, ratings);
}

function b64url(input: string): string {
  return btoa(unescape(encodeURIComponent(input)))
    .replace(/=/g, '')
    .replace(/\+/g, '-')
    .replace(/\//g, '_');
}

function mintAccessToken(userId: string): string {
  const header = b64url(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const now = Math.floor(Date.now() / 1000);
  const payload = b64url(
    JSON.stringify({
      sub: userId,
      iat: now,
      exp: now + 15 * 60, // 15 minutes per ADR 0003
      jti: crypto.randomUUID(),
    }),
  );
  return `${header}.${payload}.dev`;
}

function mintRefreshToken(userId: string, familyId?: string): string {
  const id = crypto.randomUUID().replace(/-/g, '');
  const fam = familyId ?? crypto.randomUUID().replace(/-/g, '');
  refreshTokens.set(id, { userId, familyId: fam });
  persistRefreshTokens();
  return id;
}

interface AuthResultPayload {
  accessToken: string;
  refreshToken: string;
  accessTokenExpiresAt: string;
  refreshTokenExpiresAt: string;
  user: {
    id: string;
    email: string;
    displayName: string;
    avatarRef: string | null;
    registeredAt: string;
  };
}

function authResultFor(user: MockUser): AuthResultPayload {
  const accessToken = mintAccessToken(user.id);
  const refreshToken = mintRefreshToken(user.id);
  const now = Date.now();
  return {
    accessToken,
    refreshToken,
    accessTokenExpiresAt: new Date(now + 15 * 60 * 1000).toISOString(),
    refreshTokenExpiresAt: new Date(now + 7 * 24 * 60 * 60 * 1000).toISOString(),
    user: {
      id: user.id,
      email: user.email,
      displayName: user.displayName,
      avatarRef: user.avatarRef,
      registeredAt: user.registeredAt,
    },
  };
}

function errorResponse(status: number, code: string, message: string) {
  return HttpResponse.json({ error: code, message }, { status });
}

function authenticate(request: Request): MockUser | { error: ReturnType<typeof errorResponse> } {
  hydrateUsers();
  hydrateDecks();
  const auth = request.headers.get('authorization') ?? '';
  const match = /^Bearer (.+)$/i.exec(auth);
  if (!match) return { error: errorResponse(401, 'unauthorized', 'missing bearer token') };
  const parts = match[1]!.split('.');
  if (parts.length !== 3) return { error: errorResponse(401, 'unauthorized', 'malformed token') };
  let payload: { sub?: string; exp?: number };
  try {
    payload = JSON.parse(atob(parts[1]!.replace(/-/g, '+').replace(/_/g, '/'))) as {
      sub?: string;
      exp?: number;
    };
  } catch {
    return { error: errorResponse(401, 'unauthorized', 'unreadable token') };
  }
  if (!payload.sub) return { error: errorResponse(401, 'unauthorized', 'missing sub') };
  if (payload.exp && payload.exp * 1000 < Date.now()) {
    return { error: errorResponse(401, 'unauthorized', 'token expired') };
  }
  const user = users.find((u) => u.id === payload.sub);
  if (!user) return { error: errorResponse(404, 'user_not_found', 'no such user') };
  return user;
}

function deckPayload(deck: MockDeck) {
  return {
    id: deck.id,
    title: deck.title,
    authorName: deck.authorName,
    createdAt: deck.createdAt,
    termsCount: deck.cards.length,
    lessonsCount: deck.cards.length,
    cards: deck.cards.map((card) => ({
      id: card.id,
      term: card.term,
      definition: card.definition,
      ordinal: card.ordinal,
    })),
  };
}

const SAMPLE_GENERATED_CARDS = [
  { term: 'Atom', definition: 'Smallest unit of state in Reatom — an immutable, observable cell.' },
  { term: 'Action', definition: 'Callable, observable event used for imperative side-effects.' },
  { term: 'Computed', definition: 'Lazy derived state with automatic dependency tracking.' },
  { term: 'Effect', definition: 'A computed that auto-subscribes to run side-effects.' },
  { term: 'Wrap', definition: 'Helper that preserves async context across promise boundaries.' },
];

export const handlers = [
  http.get('/api/healthz', () => HttpResponse.json({ status: 'ok' })),

  /** RegisterUser — POST /api/auth/register (use-cases.md). */
  http.post('/api/auth/register', async ({ request }) => {
    const body = (await request.json().catch(() => null)) as
      | { email?: string; password?: string; displayName?: string }
      | null;
    const email = body?.email?.trim().toLowerCase();
    const password = body?.password ?? '';
    const displayName = body?.displayName?.trim() ?? '';

    if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      return errorResponse(422, 'invalid_email', 'invalid email');
    }
    if (password.length < 8) {
      return errorResponse(422, 'password_too_weak', 'password must be at least 8 characters');
    }
    if (displayName.length === 0 || displayName.length > 64) {
      return errorResponse(422, 'invalid_display_name', 'display name length must be 1..64');
    }
    if (users.some((u) => u.email === email)) {
      return errorResponse(409, 'email_taken', 'email already in use');
    }

    const user: MockUser = {
      id: crypto.randomUUID(),
      email,
      password,
      displayName,
      avatarRef: null,
      registeredAt: new Date().toISOString(),
    };
    users.push(user);
    persistUsers();
    return HttpResponse.json(authResultFor(user));
  }),

  /** LoginUser — POST /api/auth/login. */
  http.post('/api/auth/login', async ({ request }) => {
    hydrateUsers();
    const body = (await request.json().catch(() => null)) as
      | { email?: string; password?: string }
      | null;
    const email = body?.email?.trim().toLowerCase();
    const password = body?.password ?? '';

    if (!email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      return errorResponse(422, 'invalid_email', 'invalid email');
    }

    const existing = users.find((u) => u.email === email);
    // Convenience for dev — auto-create on login if email looks valid and there
    // are no users yet, so a fresh dev session can sign in without registering.
    const user: MockUser =
      existing ??
      (() => {
        const created: MockUser = {
          id: crypto.randomUUID(),
          email,
          password,
          displayName: email.split('@')[0] ?? 'Demo',
          avatarRef: null,
          registeredAt: new Date().toISOString(),
        };
        users.push(created);
        persistUsers();
        return created;
      })();

    // Per spec: login accepts anything ≥1 char. Don't validate password length;
    // do verify it matches if a user already existed.
    if (existing && existing.password !== password && password.length === 0) {
      return errorResponse(401, 'invalid_credentials', 'invalid email or password');
    }

    return HttpResponse.json(authResultFor(user));
  }),

  /** RefreshAccessToken — POST /api/auth/refresh. */
  http.post('/api/auth/refresh', async ({ request }) => {
    hydrateRefreshTokens();
    hydrateUsers();
    const body = (await request.json().catch(() => null)) as
      | { refreshToken?: string }
      | null;
    const presented = body?.refreshToken;
    if (!presented) {
      return errorResponse(401, 'refresh_invalid', 'missing refresh token');
    }
    const record = refreshTokens.get(presented);
    if (!record) {
      return errorResponse(401, 'refresh_invalid', 'unknown refresh token');
    }
    refreshTokens.delete(presented);
    persistRefreshTokens();
    const accessToken = mintAccessToken(record.userId);
    const refreshToken = mintRefreshToken(record.userId, record.familyId);
    const now = Date.now();
    return HttpResponse.json({
      accessToken,
      refreshToken,
      accessTokenExpiresAt: new Date(now + 15 * 60 * 1000).toISOString(),
      refreshTokenExpiresAt: new Date(now + 7 * 24 * 60 * 60 * 1000).toISOString(),
    });
  }),

  /** ChangePassword — POST /api/auth/change-password. */
  http.post('/api/auth/change-password', async ({ request }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    const body = (await request.json().catch(() => null)) as
      | { oldPassword?: string; newPassword?: string }
      | null;
    const oldPassword = body?.oldPassword ?? '';
    const newPassword = body?.newPassword ?? '';
    if (auth.password.length > 0 && auth.password !== oldPassword) {
      return errorResponse(401, 'invalid_credentials', 'old password does not match');
    }
    if (newPassword.length < 8) {
      return errorResponse(422, 'password_too_weak', 'password must be at least 8 characters');
    }
    auth.password = newPassword;
    persistUsers();
    return HttpResponse.json({ ok: true });
  }),

  /** GetCurrentUser — GET /api/me. */
  http.get('/api/me', ({ request }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    return HttpResponse.json({
      id: auth.id,
      email: auth.email,
      displayName: auth.displayName,
      avatarRef: auth.avatarRef,
      registeredAt: auth.registeredAt,
    });
  }),

  /** UpdateCurrentUser — PUT /api/me. */
  http.put('/api/me', async ({ request }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    const body = (await request.json().catch(() => null)) as
      | { email?: string; displayName?: string }
      | null;
    if (body?.email !== undefined) {
      const normalized = body.email.trim().toLowerCase();
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalized)) {
        return errorResponse(422, 'invalid_email', 'invalid email');
      }
      if (users.some((u) => u.email === normalized && u.id !== auth.id)) {
        return errorResponse(409, 'email_taken', 'email already in use');
      }
      auth.email = normalized;
    }
    if (body?.displayName !== undefined) {
      const dn = body.displayName.trim();
      if (dn.length === 0 || dn.length > 64) {
        return errorResponse(422, 'invalid_display_name', 'display name length must be 1..64');
      }
      auth.displayName = dn;
    }
    persistUsers();
    return HttpResponse.json({
      id: auth.id,
      email: auth.email,
      displayName: auth.displayName,
      avatarRef: auth.avatarRef,
      registeredAt: auth.registeredAt,
    });
  }),

  /** ListDecks — GET /api/decks. */
  http.get('/api/decks', ({ request }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    const items = decks.filter((d) => d.ownerId === auth.id).map(deckPayload);
    return HttpResponse.json({ items });
  }),

  /** GetDeck — GET /api/decks/:id. */
  http.get('/api/decks/:id', ({ request, params }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    const id = String(params.id);
    const deck = decks.find((d) => d.id === id && d.ownerId === auth.id);
    if (!deck) return errorResponse(404, 'deck_not_found', 'deck not found');
    return HttpResponse.json(deckPayload(deck));
  }),

  /** CreateDeck — POST /api/decks. */
  http.post('/api/decks', async ({ request }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    const body = (await request.json().catch(() => null)) as
      | { title?: string; cards?: Array<{ term?: string; definition?: string }> }
      | null;
    const title = (body?.title ?? '').trim();
    if (title.length === 0 || title.length > 120) {
      return errorResponse(422, 'invalid_title', 'title length must be 1..120');
    }
    const inputCards = Array.isArray(body?.cards) ? body!.cards : [];
    const cards: MockCard[] = inputCards.map((c, idx) => ({
      id: crypto.randomUUID(),
      term: (c?.term ?? '').trim(),
      definition: (c?.definition ?? '').trim(),
      ordinal: idx + 1,
    }));
    const deck: MockDeck = {
      id: crypto.randomUUID(),
      ownerId: auth.id,
      title,
      authorName: auth.displayName,
      createdAt: new Date().toISOString(),
      cards,
    };
    decks.push(deck);
    persistDecks();
    return HttpResponse.json(deckPayload(deck));
  }),

  /** UpdateDeck — PUT /api/decks/:id. */
  http.put('/api/decks/:id', async ({ request, params }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    const id = String(params.id);
    const deck = decks.find((d) => d.id === id && d.ownerId === auth.id);
    if (!deck) return errorResponse(404, 'deck_not_found', 'deck not found');
    const body = (await request.json().catch(() => null)) as
      | { title?: string; cards?: Array<{ term?: string; definition?: string }> }
      | null;
    if (body?.title !== undefined) {
      const t = body.title.trim();
      if (t.length === 0 || t.length > 120) {
        return errorResponse(422, 'invalid_title', 'title length must be 1..120');
      }
      deck.title = t;
    }
    if (Array.isArray(body?.cards)) {
      deck.cards = body!.cards.map((c, idx) => ({
        id: crypto.randomUUID(),
        term: (c?.term ?? '').trim(),
        definition: (c?.definition ?? '').trim(),
        ordinal: idx + 1,
      }));
    }
    persistDecks();
    return HttpResponse.json(deckPayload(deck));
  }),

  /** DeleteDeck — DELETE /api/decks/:id. */
  http.delete('/api/decks/:id', ({ request, params }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    const id = String(params.id);
    const idx = decks.findIndex((d) => d.id === id && d.ownerId === auth.id);
    if (idx < 0) return errorResponse(404, 'deck_not_found', 'deck not found');
    decks.splice(idx, 1);
    persistDecks();
    return new HttpResponse(null, { status: 204 });
  }),

  /** GenerateDeck — POST /api/decks/generate. Always succeeds in dev MSW. */
  http.post('/api/decks/generate', async ({ request }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    return HttpResponse.json({ cards: SAMPLE_GENERATED_CARDS });
  }),

  /** StartPracticeSession — POST /api/practice/sessions. */
  http.post('/api/practice/sessions', async ({ request }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    hydrateSessions();
    const body = (await request.json().catch(() => null)) as
      | { deckId?: string; mode?: 'tracked' | 'untracked' }
      | null;
    const deckId = (body?.deckId ?? '').trim();
    const mode: MockPracticeMode = body?.mode === 'untracked' ? 'untracked' : 'tracked';
    if (deckId.length === 0) {
      return errorResponse(422, 'invalid_deck_id', 'deck id is required');
    }
    const deck = decks.find((d) => d.id === deckId && d.ownerId === auth.id);
    if (!deck) return errorResponse(404, 'deck_not_found', 'deck not found');
    const session: MockPracticeSession = {
      id: crypto.randomUUID(),
      deckId,
      ownerId: auth.id,
      mode,
      startedAt: new Date().toISOString(),
      completedAt: null,
    };
    sessions.push(session);
    persistSessions();
    return HttpResponse.json({
      id: session.id,
      deckId: session.deckId,
      startedAt: session.startedAt,
      mode: session.mode,
    });
  }),

  /** RatePracticeCard — POST /api/practice/sessions/:id/ratings. */
  http.post('/api/practice/sessions/:id/ratings', async ({ request, params }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    hydrateSessions();
    hydrateRatings();
    const sessionId = String(params.id);
    const session = sessions.find((s) => s.id === sessionId && s.ownerId === auth.id);
    if (!session) return errorResponse(404, 'session_not_found', 'session not found');
    const body = (await request.json().catch(() => null)) as
      | { cardId?: string; rating?: number }
      | null;
    const cardId = (body?.cardId ?? '').trim();
    const ratingNum = Number(body?.rating);
    if (cardId.length === 0) {
      return errorResponse(422, 'invalid_card_id', 'card id is required');
    }
    if (ratingNum !== 0 && ratingNum !== 1 && ratingNum !== 2) {
      return errorResponse(422, 'invalid_rating', 'rating must be 0, 1, or 2');
    }
    ratings.push({
      sessionId,
      deckId: session.deckId,
      cardId,
      rating: ratingNum as MockRating,
      ratedAt: new Date().toISOString(),
    });
    persistRatings();
    return new HttpResponse(null, { status: 204 });
  }),

  /** FinishPracticeSession — POST /api/practice/sessions/:id/finish. */
  http.post('/api/practice/sessions/:id/finish', ({ request, params }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    hydrateSessions();
    const sessionId = String(params.id);
    const session = sessions.find((s) => s.id === sessionId && s.ownerId === auth.id);
    if (!session) return errorResponse(404, 'session_not_found', 'session not found');
    if (session.completedAt === null) {
      session.completedAt = new Date().toISOString();
      persistSessions();
    }
    return HttpResponse.json({ id: session.id, completedAt: session.completedAt });
  }),

  /** GetPracticeResults — GET /api/practice/sessions/:id/results. */
  http.get('/api/practice/sessions/:id/results', ({ request, params }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    hydrateSessions();
    hydrateRatings();
    const sessionId = String(params.id);
    const session = sessions.find((s) => s.id === sessionId && s.ownerId === auth.id);
    if (!session) return errorResponse(404, 'session_not_found', 'session not found');
    // Aggregate the LAST rating per card to avoid double-counting double taps.
    const lastByCard = new Map<string, MockRating>();
    for (const r of ratings.filter((x) => x.sessionId === sessionId)) {
      lastByCard.set(r.cardId, r.rating);
    }
    let knowCount = 0;
    let learningCount = 0;
    let dontKnowCount = 0;
    for (const rating of lastByCard.values()) {
      if (rating === 2) knowCount += 1;
      else if (rating === 1) learningCount += 1;
      else dontKnowCount += 1;
    }
    const total = knowCount + learningCount + dontKnowCount;
    return HttpResponse.json({
      deckId: session.deckId,
      knowCount,
      learningCount,
      dontKnowCount,
      total,
      completedAt: session.completedAt,
    });
  }),

  /** GetDeckProgress — GET /api/decks/:deckId/progress. */
  http.get('/api/decks/:deckId/progress', ({ request, params }) => {
    const auth = authenticate(request);
    if ('error' in auth) return auth.error;
    hydrateSessions();
    hydrateRatings();
    const deckId = String(params.deckId);
    const deck = decks.find((d) => d.id === deckId && d.ownerId === auth.id);
    if (!deck) return errorResponse(404, 'deck_not_found', 'deck not found');
    // Walk every rating across all sessions for this deck/owner; keep the last rating per cardId.
    const lastByCard = new Map<string, MockRating>();
    const ownedSessionIds = new Set(
      sessions.filter((s) => s.deckId === deckId && s.ownerId === auth.id).map((s) => s.id),
    );
    for (const r of ratings.filter((x) => ownedSessionIds.has(x.sessionId))) {
      lastByCard.set(r.cardId, r.rating);
    }
    let knowCount = 0;
    let learningCount = 0;
    let dontKnowCount = 0;
    for (const rating of lastByCard.values()) {
      if (rating === 2) knowCount += 1;
      else if (rating === 1) learningCount += 1;
      else dontKnowCount += 1;
    }
    return HttpResponse.json({
      deckId,
      knowCount,
      learningCount,
      dontKnowCount,
      total: deck.cards.length,
    });
  }),
];
