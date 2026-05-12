import { rootRoute } from '@/shared/router/index.ts';
import { LoginPage } from '../ui/LoginPage.tsx';

export const authLoginRoute = rootRoute.reatomRoute(
  {
    path: 'login',
    render() {
      return <LoginPage />;
    },
  },
  'authLogin',
);
