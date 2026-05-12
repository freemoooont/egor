import { rootRoute } from '@/shared/router/index.ts';
import { RegisterPage } from '../ui/RegisterPage.tsx';

export const authRegisterRoute = rootRoute.reatomRoute(
  {
    path: 'register',
    render() {
      return <RegisterPage />;
    },
  },
  'authRegister',
);
