import { computed, withAsyncData, wrap } from '@reatom/core';

import { ApiError, getPracticeResults, getDeckProgress } from '@/shared/api/index.ts';
import { tokenStore } from '@/shared/auth/index.ts';

import {
  resultsFromDto,
  progressFromDto,
  type PracticeResults,
  type DeckProgress,
} from './results.ts';

/**
 * `practice.results.fetch` — factory for an async-data computed bound to a
 * single sessionId. Pages create one of these per route mount; cleanup is
 * automatic via Reatom's reactive lifecycle.
 *
 * Accepts a getter for the sessionId so the computed re-runs when the URL
 * search param changes.
 */
export function reatomPracticeResults(getSessionId: () => string | null) {
  return computed(async (): Promise<PracticeResults | null> => {
    const sessionId = getSessionId();
    if (sessionId === null || sessionId.length === 0) return null;
    if (tokenStore.getAccess() === null && tokenStore.getRefresh() === null) return null;
    try {
      const dto = await wrap(getPracticeResults(sessionId));
      return resultsFromDto(dto);
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) return null;
      throw err;
    }
  }, 'practice.results.fetch').extend(
    withAsyncData({ initState: null as PracticeResults | null, status: true }),
  );
}

export function reatomDeckProgress(getDeckId: () => string | null) {
  return computed(async (): Promise<DeckProgress | null> => {
    const deckId = getDeckId();
    if (deckId === null || deckId.length === 0) return null;
    if (tokenStore.getAccess() === null && tokenStore.getRefresh() === null) return null;
    try {
      const dto = await wrap(getDeckProgress(deckId));
      return progressFromDto(dto);
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) return null;
      throw err;
    }
  }, 'practice.progress.fetch').extend(
    withAsyncData({ initState: null as DeckProgress | null, status: true }),
  );
}
