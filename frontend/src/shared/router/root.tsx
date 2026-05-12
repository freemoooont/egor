import { reatomRoute } from '@reatom/core';
import type { ReactNode } from 'react';

import { AppShell } from '@/widgets/app-shell/index.ts';

/**
 * Root layout route — wraps every page in the AppShell.
 * All page-level routes must nest under this one via
 * `rootRoute.reatomRoute({ path: '...', render() {...} }, 'name')`.
 *
 * `outlet()` returns one entry per registered child route (null for
 * unmatched). We render the first non-null match so sibling routes can
 * coexist regardless of registration order.
 */
export const rootRoute = reatomRoute(
  {
    layout: true,
    render(self: { outlet: () => ReactNode[] }) {
      const slots = self.outlet();
      // Render all matched slots; reatomRoute returns null for unmatched
      // child routes and the rendered React node for matched ones. Layouts
      // that nest pass through via their own outlet so order doesn't matter.
      return <AppShell>{slots.filter((slot) => slot != null)}</AppShell>;
    },
  } as never,
  'root',
);
