import { rootRoute } from '@/shared/router/index.ts';
import { DeckCreatePage } from '../ui/DeckCreatePage.tsx';

export const deckCreateRoute = rootRoute.reatomRoute(
  {
    path: 'decks/new',
    render() {
      return <DeckCreatePage />;
    },
  },
  'deckCreate',
);
