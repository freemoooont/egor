import { z } from 'zod';

import { rootRoute } from '@/shared/router/index.ts';
import { PracticeResultsPage } from '../ui/PracticeResultsPage.tsx';

/**
 * `/decks/:id/results?sessionId=<id>` — reads the sessionId from the search
 * string. The page surfaces a 404 state if no sessionId is given or the
 * results endpoint returns 404.
 */
export const practiceResultsRoute = rootRoute.reatomRoute(
  {
    path: 'decks/:id/results',
    search: z.object({ sessionId: z.string().optional() }),
    render() {
      return <PracticeResultsPage />;
    },
  },
  'practiceResults',
);
