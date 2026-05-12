/**
 * Re-export the root layout route from `shared/router/`.
 *
 * Pages register themselves under `rootRoute.reatomRoute(...)` — see
 * `app/routes.ts` for the side-effect imports that pull every page route into
 * the registry on app boot.
 *
 * The auth guard `effect` is started in `src/setup.ts` so it runs inside the
 * reatom root frame. See `@/shared/auth/guard.ts` for behavior.
 */
export { rootRoute } from '@/shared/router/index.ts';
