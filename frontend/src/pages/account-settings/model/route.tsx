import { rootRoute } from '@/shared/router/index.ts';
import { AccountSettingsPage } from '../ui/AccountSettingsPage.tsx';

export const accountSettingsRoute = rootRoute.reatomRoute(
  {
    path: 'account',
    render() {
      return <AccountSettingsPage />;
    },
  },
  'accountSettings',
);
