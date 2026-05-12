import { wrap } from '@reatom/core';

import { rootRoute } from '@/shared/router/index.ts';
import { ApiError, decks } from '@/shared/api/index.ts';
import { deckFromDto, type Deck } from '@/entities/deck/index.ts';

import { DeckPracticePage } from '../ui/DeckPracticePage.tsx';
import { resetPracticeStateAction } from './practiceState.ts';

/**
 * `/decks/:id/practice` — fetches the deck via the route loader so the page
 * can render the term/definition pairs without a separate effect. Reset
 * `currentCardIdxAtom` and flip state on every entry.
 */
export const deckPracticeRoute = rootRoute.reatomRoute(
  {
    path: 'decks/:id/practice',
    async loader({ id }: { id: string }): Promise<Deck | null> {
      // Reset client state so re-entering a deck always starts at card #1.
      resetPracticeStateAction();
      try {
        const dto = await wrap(decks.get(id));
        return deckFromDto(dto);
      } catch (err) {
        if (err instanceof ApiError && (err.status === 404 || err.status === 401)) {
          return null;
        }
        throw err;
      }
    },
    render() {
      return <DeckPracticePage />;
    },
  },
  'deckPractice',
);
