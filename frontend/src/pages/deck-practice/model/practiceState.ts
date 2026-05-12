import { atom, action, reatomBoolean, withAsync, wrap } from '@reatom/core';

import { ApiError } from '@/shared/api/index.ts';
import {
  Rating,
  rateCardAction,
  startSessionAction,
  finishSessionAction,
  activeSessionAtom,
} from '@/entities/practice/index.ts';

/**
 * Deck-practice client-side state model.
 *
 * Contains atoms / actions used by `DeckPracticePage`. The deck data itself
 * comes from the route loader (see `route.tsx`).
 */

/** Index of the visible card (0-based). Starts at 0 on every route mount. */
export const currentCardIdxAtom = atom<number>(0, 'practice.session.currentCardIdxAtom');

/** Whether the user has opted into "tracked" mode for this run. */
export const trackProgressAtom = reatomBoolean(false, 'practice.session.trackProgressAtom');

/** Whether the flashcard is currently showing the back face (definition). */
export const cardFlippedAtom = reatomBoolean(false, 'practice.session.cardFlippedAtom');

export const practiceErrorAtom = atom<string | null>(null, 'practice.session.errorAtom');

/**
 * Reset client state for a fresh deck visit. Called by the route loader.
 */
export const resetPracticeStateAction = action(() => {
  currentCardIdxAtom.set(0);
  cardFlippedAtom.setFalse();
  practiceErrorAtom.set(null);
}, 'practice.session.resetStateAction');

interface AdvanceOptions {
  total: number;
  onFinish?: () => void;
}

/**
 * Move to the next card. If we're past the end, fire `onFinish`.
 */
export const advanceCardAction = action(({ total, onFinish }: AdvanceOptions) => {
  cardFlippedAtom.setFalse();
  const idx = currentCardIdxAtom();
  const next = idx + 1;
  if (next >= total) {
    onFinish?.();
    return;
  }
  currentCardIdxAtom.set(next);
}, 'practice.session.advanceCardAction');

/**
 * Move to the previous card (no-op at idx=0).
 */
export const previousCardAction = action(() => {
  cardFlippedAtom.setFalse();
  const idx = currentCardIdxAtom();
  if (idx <= 0) return;
  currentCardIdxAtom.set(idx - 1);
}, 'practice.session.previousCardAction');

export const flipCardAction = action(() => {
  cardFlippedAtom.toggle();
}, 'practice.session.flipCardAction');

interface RateAndAdvanceInput {
  cardId: string;
  rating: typeof Rating.DontKnow | typeof Rating.StillLearning | typeof Rating.KnowKnow;
  total: number;
  onFinish?: () => void;
}

/**
 * Rate the current card and advance to the next one. Network failures are
 * captured into `practiceErrorAtom` but the UI still advances so the user is
 * not blocked.
 */
export const rateAndAdvanceAction = action(
  async ({ cardId, rating, total, onFinish }: RateAndAdvanceInput) => {
    const session = activeSessionAtom();
    practiceErrorAtom.set(null);
    if (session) {
      try {
        await wrap(rateCardAction({ sessionId: session.id, cardId, rating }));
      } catch (err) {
        if (err instanceof ApiError) {
          practiceErrorAtom.set(err.message || 'Не удалось сохранить оценку');
        } else {
          practiceErrorAtom.set('Не удалось сохранить оценку');
        }
      }
    }
    advanceCardAction({ total, onFinish });
  },
  'practice.session.rateAndAdvanceAction',
).extend(withAsync());

interface ToggleTrackingOptions {
  deckId: string;
  /** Called with the new sessionId when a session is started. */
  onSessionStarted?: (sessionId: string) => void;
  /** Called when tracking is turned off and we need to finish the active session. */
  onSessionFinished?: (sessionId: string) => void;
}

/**
 * Flip the toggle. When turning ON we start a tracked session; when turning
 * OFF we finish the active one. Errors are silenced into the local error atom.
 */
export const toggleTrackingAction = action(
  async ({ deckId, onSessionStarted, onSessionFinished }: ToggleTrackingOptions) => {
    practiceErrorAtom.set(null);
    const next = !trackProgressAtom();
    trackProgressAtom.set(next);
    if (next) {
      try {
        const session = await wrap(startSessionAction(deckId, 'tracked'));
        onSessionStarted?.(session.id);
      } catch (err) {
        if (err instanceof ApiError) {
          practiceErrorAtom.set(err.message || 'Не удалось начать сессию');
        } else {
          practiceErrorAtom.set('Не удалось начать сессию');
        }
        // revert
        trackProgressAtom.setFalse();
      }
    } else {
      const session = activeSessionAtom();
      if (session) {
        try {
          await wrap(finishSessionAction(session.id));
          onSessionFinished?.(session.id);
        } catch (err) {
          if (err instanceof ApiError) {
            practiceErrorAtom.set(err.message || 'Не удалось завершить сессию');
          } else {
            practiceErrorAtom.set('Не удалось завершить сессию');
          }
        }
      }
    }
  },
  'practice.session.toggleTrackingAction',
).extend(withAsync());

interface FinishNowOptions {
  onFinished: (sessionId: string) => void;
}

/**
 * Imperatively finish the current session (called when the user reaches the
 * end of the deck). No-op if no active session.
 */
export const finishNowAction = action(
  async ({ onFinished }: FinishNowOptions) => {
    const session = activeSessionAtom();
    if (!session) return;
    try {
      await wrap(finishSessionAction(session.id));
      onFinished(session.id);
    } catch (err) {
      if (err instanceof ApiError) {
        practiceErrorAtom.set(err.message || 'Не удалось завершить сессию');
      } else {
        practiceErrorAtom.set('Не удалось завершить сессию');
      }
    }
  },
  'practice.session.finishNowAction',
).extend(withAsync());
