import { reatomComponent } from '@reatom/react';

import { rootRoute } from '@/app/router.ts';
import { Providers } from '@/app/providers.tsx';

// Side-effect import — registers every page route with the router.
import './routes.ts';

export const App = reatomComponent(() => {
  return <Providers>{rootRoute.render()}</Providers>;
}, 'App');
