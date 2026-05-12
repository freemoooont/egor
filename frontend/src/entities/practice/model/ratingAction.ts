import { action, withAsync, wrap } from '@reatom/core';

import { ratePracticeCard } from '@/shared/api/index.ts';

import type { Rating } from './rating.ts';

interface RateInput {
  sessionId: string;
  cardId: string;
  rating: Rating;
}

/**
 * `practice.session.rateCardAction` — submit a rating for a card in a session.
 *
 * The wire enum matches the domain enum (0/1/2). Errors propagate; the
 * deck-practice page surfaces them via `withAsync().error`.
 */
export const rateCardAction = action(async ({ sessionId, cardId, rating }: RateInput) => {
  await wrap(ratePracticeCard(sessionId, cardId, rating));
}, 'practice.session.rateCardAction').extend(withAsync());
