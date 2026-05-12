/**
 * Flat path constants — convenient for `<a href={ROUTES.login}>` in shared UI.
 *
 * Lives in `shared/config` (and not `app/routes.ts`) because `app/routes.ts`
 * imports every page slice, and pulling `ROUTES` from there into widgets/UI
 * would create a circular import (widget → app/routes → page → widget).
 */
export const ROUTES = {
  login: '/login',
  register: '/register',
  decks: '/decks',
  account: '/account',
  deckNew: '/decks/new',
  deckPractice: (id: string) => `/decks/${id}/practice`,
  deckResults: (id: string) => `/decks/${id}/results`,
} as const;
