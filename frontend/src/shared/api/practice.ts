import { z } from 'zod';

import { apiFetch } from './client.ts';

/**
 * Practice endpoints — typed wrappers over `/api/practice/*` and the deck
 * progress aggregator. Mirrors the use-cases listed in the MVP spec.
 *
 * Rating enum is numeric on the wire (matches the backend's domain enum):
 *   0 = Don't know
 *   1 = Still learning
 *   2 = Know
 */

export type PracticeMode = 'tracked' | 'untracked';
export type PracticeRating = 0 | 1 | 2;

export const PRACTICE_RATING = {
  DontKnow: 0,
  StillLearning: 1,
  KnowKnow: 2,
} as const satisfies Record<string, PracticeRating>;

export const practiceSessionSchema = z.object({
  id: z.string(),
  deckId: z.string(),
  startedAt: z.string(),
  mode: z.union([z.literal('tracked'), z.literal('untracked')]),
});
export type PracticeSessionDto = z.infer<typeof practiceSessionSchema>;

// Bug #3 follow-up: the live API emits `{sessionId, deckId, mode,
// countDontKnow, countStillLearning, countKnowKnow, ratedCards, completedAt}`
// from POST /practice/sessions/:id/finish. The previous schema demanded a
// canonical `id` field which isn't present on the wire — every finish call
// rejected even though the server completed the session. Accept both shapes
// for defensive deserialization.
export const practiceFinishSchema = z.object({
  id: z.string().optional(),
  sessionId: z.string().optional(),
  completedAt: z.string(),
});
export type PracticeFinishDto = z.infer<typeof practiceFinishSchema>;

// Bug #3 follow-up: backend wire shape uses
// `{sessionId, deckId, mode, countDontKnow, countStillLearning, countKnowKnow,
// ratedCards, completedAt}`. The previous schema only knew the camelCase
// `knowCount` aliases. Accept BOTH spellings (the legacy aliases are
// optional) so the schema parses production responses cleanly.
export const practiceResultsSchema = z
  .object({
    sessionId: z.string().optional(),
    deckId: z.string(),
    mode: z.string().optional(),
    completedAt: z.string().nullable(),
    // Real wire fields (backend Go DTO).
    countDontKnow: z.number().optional(),
    countStillLearning: z.number().optional(),
    countKnowKnow: z.number().optional(),
    // Legacy / mock fields kept for back-compat with MSW handlers.
    knowCount: z.number().optional(),
    learningCount: z.number().optional(),
    dontKnowCount: z.number().optional(),
    total: z.number().optional(),
    ratedCards: z.array(z.unknown()).optional(),
  })
  .transform((v) => {
    const knowCount = v.knowCount ?? v.countKnowKnow ?? 0;
    const learningCount = v.learningCount ?? v.countStillLearning ?? 0;
    const dontKnowCount = v.dontKnowCount ?? v.countDontKnow ?? 0;
    const total = v.total ?? knowCount + learningCount + dontKnowCount;
    return {
      deckId: v.deckId,
      knowCount,
      learningCount,
      dontKnowCount,
      total,
      completedAt: v.completedAt,
    };
  });
export type PracticeResultsDto = z.infer<typeof practiceResultsSchema>;

export const practiceProgressSchema = z.object({
  deckId: z.string(),
  knowCount: z.number(),
  learningCount: z.number(),
  dontKnowCount: z.number(),
  total: z.number(),
});
export type PracticeProgressDto = z.infer<typeof practiceProgressSchema>;

export interface StartSessionInput {
  mode: PracticeMode;
}

export async function startPracticeSession(
  deckId: string,
  opts: StartSessionInput,
): Promise<PracticeSessionDto> {
  const raw = await apiFetch('/practice/sessions', {
    method: 'POST',
    body: JSON.stringify({ deckId, mode: opts.mode }),
  });
  return practiceSessionSchema.parse(raw);
}

export async function ratePracticeCard(
  sessionId: string,
  cardId: string,
  rating: PracticeRating,
): Promise<void> {
  await apiFetch(`/practice/sessions/${encodeURIComponent(sessionId)}/ratings`, {
    method: 'POST',
    body: JSON.stringify({ cardId, rating }),
  });
}

export async function finishPracticeSession(
  sessionId: string,
): Promise<PracticeFinishDto> {
  const raw = await apiFetch(`/practice/sessions/${encodeURIComponent(sessionId)}/finish`, {
    method: 'POST',
  });
  return practiceFinishSchema.parse(raw);
}

export async function getPracticeResults(
  sessionId: string,
): Promise<PracticeResultsDto> {
  const raw = await apiFetch(`/practice/sessions/${encodeURIComponent(sessionId)}/results`, {
    method: 'GET',
  });
  return practiceResultsSchema.parse(raw);
}

export async function getDeckProgress(
  deckId: string,
): Promise<PracticeProgressDto> {
  const raw = await apiFetch(`/decks/${encodeURIComponent(deckId)}/progress`, {
    method: 'GET',
  });
  return practiceProgressSchema.parse(raw);
}

export const practice = {
  startSession: startPracticeSession,
  rateCard: ratePracticeCard,
  finishSession: finishPracticeSession,
  getResults: getPracticeResults,
  getProgress: getDeckProgress,
};
