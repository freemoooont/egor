import { z } from 'zod';

import { apiFetch } from './client.ts';

/**
 * Decks endpoints — Zod-typed wrappers over `/api/decks/*`. Mirrors the
 * use-cases listed in `docs/backend/use-cases.md`. The MSW handlers in
 * `src/mocks/handlers.ts` implement the same shape for dev.
 */

export const cardSchema = z.object({
  id: z.string(),
  term: z.string(),
  definition: z.string(),
  ordinal: z.number(),
});
export type CardDto = z.infer<typeof cardSchema>;

// Bug #3 (surfaced during the post-fix smoke run): the live Go API returns
// `{ id, ownerId, title, cards, createdAt }` for both POST /api/decks and
// GET /api/decks; it does NOT emit `authorName` / `termsCount` /
// `lessonsCount`, which the previous schema treated as required and which
// caused every protected deck mutation to throw "Не удалось создать колоду"
// even when the server returned 201. Make the cosmetic fields optional so
// the schema accepts the real wire shape; the UI computes counts from the
// embedded `cards` array.
export const deckSchema = z.object({
  id: z.string(),
  title: z.string(),
  authorName: z.string().optional(),
  createdAt: z.string(),
  termsCount: z.number().optional(),
  lessonsCount: z.number().optional(),
  // Live API list endpoint surfaces `cardCount`; full deck endpoints emit
  // the embedded `cards` array. Either is enough to compute the term count.
  cardCount: z.number().optional(),
  cards: z.array(cardSchema).optional(),
});
export type DeckDto = z.infer<typeof deckSchema>;

// Bug #3 follow-up: the live API emits `{ decks: [...] }` from
// GET /api/decks; legacy MSW handlers used `{ items: [...] }`. Accept both.
export const deckListSchema = z
  .object({
    items: z.array(deckSchema).optional(),
    decks: z.array(deckSchema).optional(),
  })
  .transform((v) => ({ items: v.items ?? v.decks ?? [] }));
export type DeckListDto = z.infer<typeof deckListSchema>;

export interface DeckCardInput {
  term: string;
  definition: string;
}

export interface CreateDeckInput {
  title: string;
  cards: DeckCardInput[];
}

export interface UpdateDeckInput {
  title?: string;
  cards?: DeckCardInput[];
}

export interface GenerateDeckInput {
  topic: string;
}

export const generatedDeckSchema = z.object({
  cards: z.array(
    z.object({
      term: z.string(),
      definition: z.string(),
    }),
  ),
});
export type GeneratedDeckDto = z.infer<typeof generatedDeckSchema>;

export async function listDecks(): Promise<DeckListDto> {
  const raw = await apiFetch('/decks', { method: 'GET' });
  return deckListSchema.parse(raw);
}

export async function getDeck(id: string): Promise<DeckDto> {
  const raw = await apiFetch(`/decks/${encodeURIComponent(id)}`, { method: 'GET' });
  return deckSchema.parse(raw);
}

export async function createDeck(input: CreateDeckInput): Promise<DeckDto> {
  const raw = await apiFetch('/decks', {
    method: 'POST',
    body: JSON.stringify(input),
  });
  return deckSchema.parse(raw);
}

export async function updateDeck(id: string, input: UpdateDeckInput): Promise<DeckDto> {
  const raw = await apiFetch(`/decks/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
  return deckSchema.parse(raw);
}

export async function deleteDeck(id: string): Promise<void> {
  await apiFetch(`/decks/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function generateDeck(input: GenerateDeckInput): Promise<GeneratedDeckDto> {
  const raw = await apiFetch('/decks/generate', {
    method: 'POST',
    body: JSON.stringify(input),
  });
  return generatedDeckSchema.parse(raw);
}

export const decks = {
  list: listDecks,
  get: getDeck,
  create: createDeck,
  update: updateDeck,
  delete: deleteDeck,
  generate: generateDeck,
};
