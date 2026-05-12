import { rootRoute } from '@/shared/router/index.ts';
import { DecksListPage } from '../ui/DecksListPage.tsx';

/**
 * `/decks` — exact-only render. Reatom matches `path: 'decks'` as a prefix
 * (so `/decks/new` and `/decks/:id/practice` would also match) but we want
 * the list page only on `/decks` itself; sibling routes like deck-create
 * own their own render.
 */
export const decksListRoute = rootRoute.reatomRoute(
  {
    path: 'decks',
    render(self) {
      if (!(self as { exact: () => boolean }).exact()) return <></>;
      return <DecksListPage />;
    },
  },
  'decksList',
);
