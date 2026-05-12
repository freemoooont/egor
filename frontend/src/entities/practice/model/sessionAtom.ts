import { atom, action, withAsync, wrap } from '@reatom/core';

import {
  startPracticeSession,
  finishPracticeSession,
  type PracticeMode,
} from '@/shared/api/index.ts';

import { sessionFromDto, type PracticeSession } from './session.ts';

/**
 * `practice.session.activeAtom` — the currently active practice session, or
 * `null` if the user is in untracked mode (or no session has been started).
 *
 * The deck-practice page sets this on enter (when the toggle is ON) and
 * clears it on leave / finish.
 */
export const activeSessionAtom = atom<PracticeSession | null>(
  null,
  'practice.session.activeAtom',
);

/**
 * `practice.session.startAction` — start a new tracked session for a deck.
 */
export const startSessionAction = action(
  async (deckId: string, mode: PracticeMode = 'tracked'): Promise<PracticeSession> => {
    const dto = await wrap(startPracticeSession(deckId, { mode }));
    const session = sessionFromDto(dto);
    activeSessionAtom.set(session);
    return session;
  },
  'practice.session.startAction',
).extend(withAsync());

/**
 * `practice.session.finishAction` — finish a session by id. Clears the active
 * session atom on success.
 */
export const finishSessionAction = action(
  async (sessionId: string): Promise<{ id: string; completedAt: string }> => {
    const dto = await wrap(finishPracticeSession(sessionId));
    const current = activeSessionAtom();
    if (current && current.id === sessionId) {
      activeSessionAtom.set(null);
    }
    // The wire shape exposes either `id` or `sessionId`. Normalize to a single
    // canonical session id for downstream consumers.
    return { id: dto.id ?? dto.sessionId ?? sessionId, completedAt: dto.completedAt };
  },
  'practice.session.finishAction',
).extend(withAsync());

export const clearActiveSessionAction = action(() => {
  activeSessionAtom.set(null);
}, 'practice.session.clearAction');
