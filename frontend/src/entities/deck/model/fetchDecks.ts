import { action, withAsync, wrap } from '@reatom/core';

import {
  createDeck,
  deleteDeck,
  generateDeck,
  type CreateDeckInput,
  type GenerateDeckInput,
  type GeneratedDeckDto,
} from '@/shared/api/index.ts';

import { deckFromDto, type Deck } from './deck.ts';
import { decksListAtom, decksRefreshTickAtom } from './decksAtom.ts';

/**
 * `decksAction` — namespaced bag of mutations over the decks resource.
 * Each individual action is wired with `withAsync` so consumers get
 * `.pending`, `.error`, `.onFulfill` etc. — the page slices use these for
 * disabling buttons, toasts, and after-success navigation.
 *
 * On a successful mutation the action triggers `decksListAtom.retry()` so any
 * subscriber refetches automatically.
 */
export const createDeckAction = action(async (input: CreateDeckInput): Promise<Deck> => {
  const dto = await wrap(createDeck(input));
  decksRefreshTickAtom.set((n) => n + 1);
  decksListAtom.retry();
  return deckFromDto(dto);
}, 'entities.decks.create').extend(withAsync());

export const deleteDeckAction = action(async (id: string): Promise<string> => {
  await wrap(deleteDeck(id));
  decksRefreshTickAtom.set((n) => n + 1);
  decksListAtom.retry();
  return id;
}, 'entities.decks.delete').extend(withAsync());

export const generateDeckAction = action(
  async (input: GenerateDeckInput): Promise<GeneratedDeckDto> => {
    const result = await wrap(generateDeck(input));
    return result;
  },
  'entities.decks.generate',
).extend(withAsync());

export const decksAction = {
  create: createDeckAction,
  delete: deleteDeckAction,
  generate: generateDeckAction,
};
