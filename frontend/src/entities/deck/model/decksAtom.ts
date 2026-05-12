import { atom, computed, withAsyncData, wrap } from '@reatom/core';

import { ApiError, listDecks } from '@/shared/api/index.ts';
import { tokenStore } from '@/shared/auth/index.ts';

import { deckFromDto, type Deck } from './deck.ts';

/**
 * `decksRefreshTickAtom` — bumped by mutations (create/delete) and by the
 * route loader so the computed below re-evaluates. Reatom's `computed` only
 * re-runs when one of its tracked atoms changes; without an explicit tracker
 * the list would never refresh after a mutation.
 */
export const decksRefreshTickAtom = atom(0, 'entities.decksList.refreshTick');

/**
 * `decksList` — async-data computed for `GET /api/decks`. Returns a sorted
 * list of decks owned by the authenticated user (by `createdAt` desc). The
 * computed is lazy: it only fetches when something actually subscribes (the
 * decks-list page or a widget that lists user decks).
 *
 * Reads `tokenStore` directly (the source of truth for the access token) and
 * the `decksRefreshTickAtom` to opt into re-fetches after mutations.
 */
export const decksListAtom = computed(async (): Promise<Deck[]> => {
  decksRefreshTickAtom();
  if (tokenStore.getAccess() === null && tokenStore.getRefresh() === null) return [];
  try {
    const dto = await wrap(listDecks());
    return dto.items
      .map(deckFromDto)
      .sort((a, b) => (a.createdAt < b.createdAt ? 1 : a.createdAt > b.createdAt ? -1 : 0));
  } catch (err) {
    if (err instanceof ApiError && err.status === 401) {
      return [];
    }
    throw err;
  }
}, 'entities.decksList').extend(withAsyncData({ initState: [] as Deck[], status: true }));
